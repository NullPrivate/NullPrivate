package home

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/schedule"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// productPolicyClient 创建包含独立过滤、SafeSearch、缓存和日志选项的合成旧配置。
func productPolicyClient() *clientObject {
	return &clientObject{
		Name:      "policy-client",
		UID:       client.MustNewUID(),
		IDs:       []string{"192.0.2.10"},
		Tags:      []string{"device_pc"},
		Upstreams: []string{"192.0.2.53"},
		SafeSearchConf: filtering.SafeSearchConfig{
			Enabled: true,
			Yandex:  true,
			Google:  true,
		},
		BlockedServices: &filtering.BlockedServices{
			Schedule: schedule.EmptyWeekly(),
			IDs:      []string{},
		},
		FilteringEnabled:      true,
		SafeBrowsingEnabled:   true,
		ParentalEnabled:       true,
		UpstreamsCacheEnabled: true,
		UpstreamsCacheSize:    4096,
		IgnoreQueryLog:        true,
		IgnoreStatistics:      true,
	}
}

// requireProductPolicySafeSearch 执行内置静态规则，验证真实 SafeSearch 结果。
func requireProductPolicySafeSearch(t *testing.T, ss filtering.SafeSearch) {
	t.Helper()

	require.NotNil(t, ss)
	ctx := testutil.ContextWithTimeout(t, testTimeout)
	res, err := ss.CheckHost(ctx, "yandex.ru", dns.TypeA)
	require.NoError(t, err)
	require.True(t, res.IsFiltered)
	require.Equal(t, filtering.FilteredSafeSearch, res.Reason)
	require.Len(t, res.Rules, 1)
	require.Equal(t, netip.MustParseAddr("213.180.193.56"), res.Rules[0].IP)
}

// productPolicyRequest 独立构造 API 数据，保留调用方提交的旧开关值。
func productPolicyRequest(o *clientObject) clientJSON {
	return clientJSON{
		Name:                     o.Name,
		IDs:                      o.IDs,
		Tags:                     o.Tags,
		Upstreams:                o.Upstreams,
		SafeSearchConf:           &o.SafeSearchConf,
		SafeSearchEnabled:        o.SafeSearchConf.Enabled,
		Schedule:                 o.BlockedServices.Schedule,
		BlockedServices:          o.BlockedServices.IDs,
		UseGlobalSettings:        o.UseGlobalSettings,
		UseGlobalBlockedServices: o.UseGlobalBlockedServices,
		FilteringEnabled:         o.FilteringEnabled,
		SafeBrowsingEnabled:      o.SafeBrowsingEnabled,
		ParentalEnabled:          o.ParentalEnabled,
		UpstreamsCacheEnabled:    aghalg.BoolToNullBool(o.UpstreamsCacheEnabled),
		UpstreamsCacheSize:       o.UpstreamsCacheSize,
		IgnoreQueryLog:           aghalg.BoolToNullBool(o.IgnoreQueryLog),
		IgnoreStatistics:         aghalg.BoolToNullBool(o.IgnoreStatistics),
	}
}

// requireProductPolicyRoundTrip 集成验证 GET 回显及配置保存、重新加载路径。
// 保存使用 forConfig 与 YAML 内存往返，测试容器沿用 testing 模式。
func requireProductPolicyRoundTrip(t *testing.T, clients *clientsContainer, source *clientObject) {
	t.Helper()

	want := *source
	want.SafeBrowsingEnabled = false
	want.ParentalEnabled = false

	rw := httptest.NewRecorder()
	clients.handleGetClients(rw, httptest.NewRequest(http.MethodGet, "/control/clients", nil))
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())

	// 同时验证字段确实存在且为 false，避免解码后的零值掩盖字段缺失。
	var wire struct {
		Clients []map[string]json.RawMessage `json:"clients"`
	}
	require.NoError(t, json.Unmarshal(rw.Body.Bytes(), &wire))
	require.Len(t, wire.Clients, 1)
	for _, key := range []string{"safebrowsing_enabled", "parental_enabled"} {
		require.JSONEq(t, "false", string(wire.Clients[0][key]))
	}

	var response clientListJSON
	require.NoError(t, json.Unmarshal(rw.Body.Bytes(), &response))
	require.Len(t, response.Clients, 1)
	expected := productPolicyRequest(&want)
	require.Equal(t, expected, *response.Clients[0])

	objects := clients.forConfig()
	require.Equal(t, []*clientObject{&want}, objects)
	data, err := yaml.Marshal(objects)
	require.NoError(t, err)
	var savedFields []map[string]any
	require.NoError(t, yaml.Unmarshal(data, &savedFields))
	require.Len(t, savedFields, 1)
	require.Equal(t, false, savedFields[0]["safebrowsing_enabled"])
	require.Equal(t, false, savedFields[0]["parental_enabled"])

	var saved []*clientObject
	require.NoError(t, yaml.Unmarshal(data, &saved))
	require.Equal(t, []*clientObject{&want}, saved)
	ctx := testutil.ContextWithTimeout(t, testTimeout)
	reloaded, err := saved[0].toPersistent(ctx, testLogger, 4096, time.Minute, nil)
	require.NoError(t, err)
	require.False(t, reloaded.SafeBrowsingEnabled)
	require.False(t, reloaded.ParentalEnabled)
	require.Equal(t, want.UID, reloaded.UID)
	require.Equal(t, expected, *clientToJSON(reloaded))
	requireProductPolicySafeSearch(t, reloaded.SafeSearch)
}

// TestProductPolicyIntegration_LegacyClient 覆盖旧 YAML 到持久客户端及输出的迁移。
func TestProductPolicyIntegration_LegacyClient(t *testing.T) {
	clients := newClientsContainer(t)
	t.Cleanup(func() { require.NoError(t, clients.storage.Shutdown(t.Context())) })
	ctx := testutil.ContextWithTimeout(t, testTimeout)
	legacy := productPolicyClient()

	data, err := yaml.Marshal([]*clientObject{legacy})
	require.NoError(t, err)
	var loaded []*clientObject
	require.NoError(t, yaml.Unmarshal(data, &loaded))
	require.Len(t, loaded, 1)
	require.True(t, loaded[0].SafeBrowsingEnabled)
	require.True(t, loaded[0].ParentalEnabled)

	persistent, err := loaded[0].toPersistent(ctx, testLogger, 4096, time.Minute, nil)
	require.NoError(t, err)
	require.False(t, persistent.SafeBrowsingEnabled)
	require.False(t, persistent.ParentalEnabled)
	requireProductPolicySafeSearch(t, persistent.SafeSearch)
	require.NoError(t, clients.storage.Add(ctx, persistent))
	requireProductPolicyRoundTrip(t, clients, legacy)
}

// TestProductPolicyIntegration_LegacyStoredClient 验证输出边界也独立归一化旧内存状态。
func TestProductPolicyIntegration_LegacyStoredClient(t *testing.T) {
	clients := newClientsContainer(t)
	t.Cleanup(func() { require.NoError(t, clients.storage.Shutdown(t.Context())) })
	ctx := testutil.ContextWithTimeout(t, testTimeout)
	legacy := productPolicyClient()
	persistent, err := legacy.toPersistent(ctx, testLogger, 4096, time.Minute, nil)
	require.NoError(t, err)
	persistent.SafeBrowsingEnabled = true
	persistent.ParentalEnabled = true
	require.NoError(t, clients.storage.Add(ctx, persistent))

	requireProductPolicyRoundTrip(t, clients, legacy)
	stored := clients.persistentByName(legacy.Name)
	require.NotNil(t, stored)
	require.True(t, stored.SafeBrowsingEnabled)
	require.True(t, stored.ParentalEnabled)
}

// TestProductPolicyIntegration_ClientAPI 覆盖成功新增和更新，并校验其他选项保持原意。
func TestProductPolicyIntegration_ClientAPI(t *testing.T) {
	clients := newClientsContainer(t)
	t.Cleanup(func() { require.NoError(t, clients.storage.Shutdown(t.Context())) })
	setTestFilters(t, nil)
	legacy := productPolicyClient()

	for _, operation := range []string{"add", "update"} {
		t.Run(operation, func(t *testing.T) {
			var payload any = productPolicyRequest(legacy)
			handler := clients.handleAddClient
			if operation == "update" {
				oldName := legacy.Name
				legacy.Name = "policy-client-updated"
				legacy.IDs = []string{"192.0.2.20"}
				legacy.IgnoreQueryLog = false
				legacy.IgnoreStatistics = false
				legacy.UseGlobalBlockedServices = true
				legacy.UpstreamsCacheSize = 8192
				payload = updateJSON{Name: oldName, Data: productPolicyRequest(legacy)}
				handler = clients.handleUpdateClient
			}

			body, err := json.Marshal(payload)
			require.NoError(t, err)
			rw := httptest.NewRecorder()
			handler(rw, httptest.NewRequest(http.MethodPost, "/control/clients/"+operation, bytes.NewReader(body)))
			require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())

			stored := clients.persistentByName(legacy.Name)
			require.NotNil(t, stored)
			require.False(t, stored.SafeBrowsingEnabled)
			require.False(t, stored.ParentalEnabled)
			if operation == "add" {
				require.NotEqual(t, client.UID{}, stored.UID)
				legacy.UID = stored.UID
			}
			require.Equal(t, legacy.UID, stored.UID)
			requireProductPolicySafeSearch(t, stored.SafeSearch)
			requireProductPolicyRoundTrip(t, clients, legacy)
		})
	}
}

// TestProductPolicyIntegration_FilteringInitialization 验证兼容旧专属 upstream 的成功初始化。
// 初始化仅创建内存对象；SafeSearch 使用内置规则，HTTP 客户端保持未调用状态。
func TestProductPolicyIntegration_FilteringInitialization(t *testing.T) {
	ctx := testutil.ContextWithTimeout(t, testTimeout)
	conf := &filtering.Config{
		SafeBrowsingEnabled: true,
		ParentalEnabled:     true,
		SafeBrowsingServer:  "https://[legacy-safebrowsing.invalid",
		ParentalServer:      "https://[legacy-parental.invalid",
		FilteringEnabled:    true,
		ProtectionEnabled:   true,
		SafeSearchConf: filtering.SafeSearchConfig{
			Enabled: true,
			Yandex:  true,
		},
		SafeSearchCacheSize: 4096,
		CacheTime:           1,
	}
	wantSafeSearch := conf.SafeSearchConf

	require.NoError(t, setupDNSFilteringConf(ctx, testLogger, conf, &tlsManager{}))
	t.Cleanup(conf.HTTPClient.CloseIdleConnections)
	require.False(t, conf.SafeBrowsingEnabled)
	require.False(t, conf.ParentalEnabled)
	require.Nil(t, conf.SafeBrowsingChecker)
	require.Nil(t, conf.ParentalControlChecker)
	require.True(t, conf.FilteringEnabled)
	require.True(t, conf.ProtectionEnabled)
	require.Equal(t, wantSafeSearch, conf.SafeSearchConf)
	requireProductPolicySafeSearch(t, conf.SafeSearch)
}
