//go:build integration && linux

package home

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestInstallUnprivileged 验证普通用户从首次启动到登录及 DNS 查询的完整安装流程。
func TestInstallUnprivileged(t *testing.T) {
	if os.Getuid() == 0 || os.Geteuid() == 0 {
		t.Skip("此集成测试需要普通用户运行，以验证首次启动无需 root 权限")
	}
	requireInstallUnprivileged(t, os.Getpid())

	root, err := filepath.Abs("../..")
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(root, "build", "static", "install.html"))
	require.NoError(t, err, "构建需要已有的 build/static 前端资源")

	binary := filepath.Join(t.TempDir(), "NullPrivate")
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, ".")
	build.Dir = root
	output, err := build.CombinedOutput()
	require.NoError(t, err, "编译当前源码：\n%s", output)

	for _, tc := range []struct {
		name   string
		rebind bool
	}{
		{name: "web_port_unchanged"},
		{name: "web_port_rebound", rebind: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ports := reserveInstallPorts(t, 3)
			initialPort, webPort, dnsPort := ports[0], ports[0], ports[1]
			if tc.rebind {
				webPort = ports[2]
			}

			workDir := t.TempDir()
			configPath := filepath.Join(workDir, "AdGuardHome.yaml")
			done := startInstallProcess(t, binary, workDir, configPath, initialPort)
			jar, err := cookiejar.New(nil)
			require.NoError(t, err)
			transport := &http.Transport{Proxy: nil}
			t.Cleanup(transport.CloseIdleConnections)
			client := &http.Client{
				Transport: transport,
				Jar:       jar,
				Timeout:   10 * time.Second,
				CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
					return http.ErrUseLastResponse
				},
			}
			initialURL := fmt.Sprintf("http://127.0.0.1:%d", initialPort)
			webURL := fmt.Sprintf("http://127.0.0.1:%d", webPort)

			waitInstallPage(t, client, done, initialURL+"/install.html")
			resp, _ := installRequest(t, client, http.MethodGet, initialURL+"/", nil)
			require.Equal(t, http.StatusFound, resp.StatusCode)
			require.Equal(t, "/install.html", resp.Header.Get("Location"))

			web := map[string]any{"ip": "127.0.0.1", "port": webPort}
			dnsConfig := map[string]any{"ip": "127.0.0.1", "port": dnsPort}
			resp, body := installRequest(t, client, http.MethodPost,
				initialURL+"/control/install/check_config", map[string]any{
					"web": web, "dns": dnsConfig, "set_static_ip": false,
				})
			require.Equal(t, http.StatusOK, resp.StatusCode, "%s", body)
			var checked struct {
				Web *struct {
					Status string `json:"status"`
				} `json:"web"`
				DNS *struct {
					Status     string `json:"status"`
					CanAutofix bool   `json:"can_autofix"`
				} `json:"dns"`
			}
			require.NoError(t, json.Unmarshal(body, &checked))
			require.NotNil(t, checked.Web)
			require.NotNil(t, checked.DNS)
			require.Empty(t, checked.Web.Status)
			require.Empty(t, checked.DNS.Status)
			require.False(t, checked.DNS.CanAutofix)

			const username, password = "installer", "Install-integration-123!"
			resp, body = installRequest(t, client, http.MethodPost,
				initialURL+"/control/install/configure", map[string]any{
					"web": web, "dns": dnsConfig, "username": username, "password": password,
				})
			require.Equal(t, http.StatusOK, resp.StatusCode, "%s", body)
			waitInstallPage(t, client, done, webURL+"/login.html")

			// 安装生成的配置应保存在可写临时目录中，并记录最终监听端口。
			configData, err := os.ReadFile(configPath)
			require.NoError(t, err)
			var saved struct {
				HTTP struct {
					Address string `yaml:"address"`
				} `yaml:"http"`
				DNS struct {
					Port int `yaml:"port"`
				} `yaml:"dns"`
			}
			require.NoError(t, yaml.Unmarshal(configData, &saved))
			require.Equal(t, fmt.Sprintf("127.0.0.1:%d", webPort), saved.HTTP.Address)
			require.Equal(t, dnsPort, saved.DNS.Port)

			resp, body = installRequest(t, client, http.MethodPost, webURL+"/control/login",
				map[string]string{"name": username, "password": password})
			require.Equal(t, http.StatusOK, resp.StatusCode, "%s", body)
			cookieURL, err := url.Parse(webURL)
			require.NoError(t, err)
			var session string
			for _, cookie := range jar.Cookies(cookieURL) {
				if cookie.Name == "agh_session" {
					session = cookie.Value
				}
			}
			require.NotEmpty(t, session, "登录应返回会话 Cookie")

			// 重写使两种传输协议的验证均使用本地响应，无需公网 DNS。
			resp, body = installRequest(t, client, http.MethodPost, webURL+"/control/rewrite/add",
				map[string]string{"domain": "test.invalid", "answer": "192.0.2.123"})
			require.Equal(t, http.StatusOK, resp.StatusCode, "%s", body)
			for _, network := range []string{"udp", "tcp"} {
				t.Run(network, func(t *testing.T) {
					query := new(dns.Msg)
					query.SetQuestion("test.invalid.", dns.TypeA)
					dnsClient := &dns.Client{Net: network, Timeout: 3 * time.Second}
					answer, _, err := dnsClient.ExchangeContext(t.Context(), query,
						fmt.Sprintf("127.0.0.1:%d", dnsPort))
					require.NoError(t, err)
					require.Equal(t, dns.RcodeSuccess, answer.Rcode)
					require.Len(t, answer.Answer, 1)
					a, ok := answer.Answer[0].(*dns.A)
					require.True(t, ok, "应返回 A 记录，实际为 %T", answer.Answer[0])
					require.Equal(t, "test.invalid.", a.Hdr.Name)
					require.Equal(t, "192.0.2.123", a.A.String())
				})
			}
		})
	}
}

// requireInstallUnprivileged 同时检查测试进程及真实服务进程的 Linux 权限。
func requireInstallUnprivileged(t *testing.T, pid int) {
	t.Helper()
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	require.NoError(t, err)
	fields := map[string][]string{}
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(line, ":")
		if ok {
			fields[key] = strings.Fields(value)
		}
	}
	require.Len(t, fields["Uid"], 4)
	for _, uid := range fields["Uid"] {
		require.NotEqual(t, "0", uid, "进程应使用普通用户 UID")
	}
	for _, key := range []string{"CapPrm", "CapEff", "CapAmb"} {
		require.Len(t, fields[key], 1)
		capabilities, err := strconv.ParseUint(fields[key][0], 16, 64)
		require.NoError(t, err)
		require.Zero(t, capabilities, "进程 %d 的 %s 应为空", pid, key)
	}
}

// reserveInstallPorts 同时预检本地 TCP/UDP，保留全部监听后统一释放，确保端口互异。
func reserveInstallPorts(t *testing.T, count int) (ports []int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for len(ports) < count && time.Now().Before(deadline) {
		tcp, err := net.Listen("tcp4", "127.0.0.1:0")
		require.NoError(t, err)
		port := tcp.Addr().(*net.TCPAddr).Port
		udp, err := net.ListenPacket("udp4", tcp.Addr().String())
		if err != nil || port <= 1024 {
			_ = tcp.Close()
			if udp != nil {
				_ = udp.Close()
			}
			continue
		}
		defer tcp.Close()
		defer udp.Close()
		ports = append(ports, port)
	}
	require.Len(t, ports, count, "应能分配互异的本地高位端口")
	return ports
}

// startInstallProcess 将输出留在临时文件中，清理时终止并回收子进程，失败时展示日志。
func startInstallProcess(t *testing.T, binary, workDir, configPath string, port int) <-chan struct{} {
	t.Helper()
	logPath := filepath.Join(workDir, "process.log")
	logFile, err := os.Create(logPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = logFile.Close()
		if t.Failed() {
			data, readErr := os.ReadFile(logPath)
			t.Logf("子进程日志（读取错误：%v）：\n%s", readErr, data)
		}
	})
	cmd := exec.Command(binary, "--config", configPath, "--work-dir", workDir,
		"--web-addr", fmt.Sprintf("127.0.0.1:%d", port), "--no-check-update")
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "NULLPRIVATE_CONFIG_POSTGRES_ENABLED=false")
	cmd.SysProcAttr = &syscall.SysProcAttr{AmbientCaps: nil}
	cmd.Stdout, cmd.Stderr = logFile, logFile
	require.NoError(t, cmd.Start())
	done := make(chan struct{})
	var waitErr error
	go func() {
		waitErr = cmd.Wait()
		close(done)
	}()
	t.Cleanup(func() {
		select {
		case <-done:
		default:
			_ = cmd.Process.Signal(syscall.SIGTERM)
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				_ = cmd.Process.Kill()
				<-done
				t.Error("子进程未在清理期限内退出，已强制终止并回收")
			}
		}
		if waitErr != nil {
			t.Errorf("子进程退出：%v", waitErr)
		}
	})
	requireInstallUnprivileged(t, cmd.Process.Pid)
	return done
}

func installRequest(t *testing.T, client *http.Client, method, endpoint string, payload any) (*http.Response, []byte) {
	t.Helper()
	var data []byte
	var err error
	if payload != nil {
		data, err = json.Marshal(payload)
		require.NoError(t, err)
	}
	req, err := http.NewRequestWithContext(t.Context(), method, endpoint, bytes.NewReader(data))
	require.NoError(t, err)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp, body
}

// waitInstallPage 用截止时间轮询页面就绪，同时及时报告子进程提前退出。
func waitInstallPage(t *testing.T, client *http.Client, done <-chan struct{}, endpoint string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	var lastErr error
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		require.NoError(t, err)
		resp, err := client.Do(req)
		lastErr = err
		if err == nil {
			body, readErr := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if readErr == nil && resp.StatusCode == http.StatusOK &&
				strings.Contains(strings.ToLower(string(body)), "<html") {
				return
			}
			lastErr = fmt.Errorf("状态码 %d，读取错误 %v，正文 %.200s", resp.StatusCode, readErr, body)
		}
		select {
		case <-done:
			t.Fatalf("等待 %s 时子进程已退出：%v", endpoint, lastErr)
		case <-ctx.Done():
			t.Fatalf("等待 %s 超时：%v", endpoint, lastErr)
		case <-ticker.C:
		}
	}
}
