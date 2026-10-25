package filtering

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/schedule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDNSFilter_handleFilteringSetURL(t *testing.T) {
	filtersDir := t.TempDir()

	var goodRulesEndpoint, anotherGoodRulesEndpoint, badRulesEndpoint string
	for _, rulesSource := range []struct {
		endpoint *string
		content  []byte
	}{{
		endpoint: &goodRulesEndpoint,
		content:  []byte(`||example.org^`),
	}, {
		endpoint: &anotherGoodRulesEndpoint,
		content:  []byte(`||example.com^`),
	}, {
		endpoint: &badRulesEndpoint,
		content:  []byte(`<html></html>`),
	}} {
		*rulesSource.endpoint = serveFiltersLocally(t, rulesSource.content)
	}

	testCases := []struct {
		name     string
		wantBody string
		oldURL   string
		newName  string
		newURL   string
		initial  []FilterYAML
	}{{
		name:     "success",
		wantBody: "",
		oldURL:   goodRulesEndpoint,
		newName:  "default_one",
		newURL:   anotherGoodRulesEndpoint,
		initial: []FilterYAML{{
			Enabled: true,
			URL:     goodRulesEndpoint,
			Name:    "default_one",
			white:   false,
		}},
	}, {
		name:     "non-existing",
		wantBody: "url doesn't exist\n",
		oldURL:   anotherGoodRulesEndpoint,
		newName:  "default_one",
		newURL:   goodRulesEndpoint,
		initial: []FilterYAML{{
			Enabled: true,
			URL:     goodRulesEndpoint,
			Name:    "default_one",
			white:   false,
		}},
	}, {
		name:     "existing",
		wantBody: "url already exists\n",
		oldURL:   goodRulesEndpoint,
		newName:  "default_one",
		newURL:   anotherGoodRulesEndpoint,
		initial: []FilterYAML{{
			Enabled: true,
			URL:     goodRulesEndpoint,
			Name:    "default_one",
			white:   false,
		}, {
			Enabled: true,
			URL:     anotherGoodRulesEndpoint,
			Name:    "another_default_one",
			white:   false,
		}},
	}, {
		name:     "bad_rules",
		wantBody: "data is HTML, not plain text\n",
		oldURL:   goodRulesEndpoint,
		newName:  "default_one",
		newURL:   badRulesEndpoint,
		initial: []FilterYAML{{
			Enabled: true,
			URL:     goodRulesEndpoint,
			Name:    "default_one",
			white:   false,
		}},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			confModifiedCalled := false
			d, err := New(&Config{
				FilteringEnabled: true,
				Filters:          tc.initial,
				HTTPClient: &http.Client{
					Timeout: 5 * time.Second,
				},
				ConfigModified: func() { confModifiedCalled = true },
				DataDir:        filtersDir,
			}, nil)
			require.NoError(t, err)
			t.Cleanup(d.Close)

			d.Start()

			reqData := &filterURLReq{
				Data: &filterURLReqData{
					// Leave the name of an existing list.
					Name:    tc.newName,
					URL:     tc.newURL,
					Enabled: true,
				},
				URL:       tc.oldURL,
				Whitelist: false,
			}
			data, err := json.Marshal(reqData)
			require.NoError(t, err)

			r := httptest.NewRequest(http.MethodPost, "http://example.org", bytes.NewReader(data))
			w := httptest.NewRecorder()

			d.handleFilteringSetURL(w, r)
			assert.Equal(t, tc.wantBody, w.Body.String())

			// For the moment the non-empty response body only contains occurred
			// error, so the configuration shouldn't be written.
			assert.Equal(t, tc.wantBody == "", confModifiedCalled)
		})
	}
}

func TestDNSFilter_handleSafeBrowsingStatus(t *testing.T) {
	testCompatibilityStatus(t, "safebrowsing", func(c *Config, enabled bool) {
		c.SafeBrowsingEnabled = enabled
	})
}

func TestDNSFilter_handleParentalStatus(t *testing.T) {
	testCompatibilityStatus(t, "parental", func(c *Config, enabled bool) {
		c.ParentalEnabled = enabled
	})
}

// testCompatibilityStatus 复用两组既有 API 分支，验证兼容状态和配置写入语义。
func testCompatibilityStatus(t *testing.T, feature string, configure func(*Config, bool)) {
	t.Helper()
	baseURL := "/control/" + feature
	statusURL := baseURL + "/status"

	testCases := []struct {
		name         string
		action       string
		enabled      bool
		wantCode     int
		wantModified int
	}{{
		name:     "enable_off",
		action:   "enable",
		enabled:  false,
		wantCode: http.StatusForbidden,
	}, {
		name:     "enable_on",
		action:   "enable",
		enabled:  true,
		wantCode: http.StatusForbidden,
	}, {
		name:         "disable_on",
		action:       "disable",
		enabled:      true,
		wantCode:     http.StatusOK,
		wantModified: 1,
	}, {
		name:         "disable_off",
		action:       "disable",
		enabled:      false,
		wantCode:     http.StatusOK,
		wantModified: 1,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handlers := make(map[string]http.Handler)
			modified := 0
			conf := &Config{
				ConfigModified: func() {
					modified++
				},
				DataDir: t.TempDir(),
				HTTPRegister: func(_, url string, handler http.HandlerFunc) {
					handlers[url] = handler
				},
			}
			configure(conf, tc.enabled)
			d, err := New(conf, nil)
			require.NoError(t, err)
			t.Cleanup(d.Close)

			// 保留既有 on/off 分支的内存状态，直接验证兼容 API 的策略边界。
			d.confMu.Lock()
			configure(d.conf, tc.enabled)
			d.confMu.Unlock()

			d.RegisterFilteringHandlers()
			require.NotEmpty(t, handlers)
			require.Contains(t, handlers, statusURL)

			checkStatus := func() {
				t.Helper()
				w := httptest.NewRecorder()
				handlers[statusURL].ServeHTTP(w, httptest.NewRequest(http.MethodGet, statusURL, nil))
				require.Equal(t, http.StatusOK, w.Code)
				assert.JSONEq(t, `{"enabled":false}`, w.Body.String())
			}

			checkStatus()
			url := baseURL + "/" + tc.action
			require.Contains(t, handlers, url)
			w := httptest.NewRecorder()
			handlers[url].ServeHTTP(w, httptest.NewRequest(http.MethodPost, url, nil))
			assert.Equal(t, tc.wantCode, w.Code)
			checkStatus()
			assert.Equal(t, tc.wantModified, modified)
		})
	}
}

func TestDNSFilter_HandleCheckHost(t *testing.T) {
	const (
		cliName = "client_name"
		cliID   = "client_id"

		notFilteredHost = "not.filterd.example"
		allowedHost     = "allowed.example"
		blockedHost     = "blocked.example"
		cliHost         = "client.example"
		qTypeHost       = "qtype.example"
		cliQTypeHost    = "cli.qtype.example"

		target          = "/control/check_host"
		hostFmt         = target + "?name=%s"
		hostCliFmt      = hostFmt + "&client=%s"
		hostQTypeFmt    = hostFmt + "&qtype=%s"
		hostCliQTypeFmt = hostCliFmt + "&qtype=%s"

		allowedRuleFmt         = "@@||%s^"
		blockedRuleFmt         = "||%s^"
		blockedRuleCliFmt      = blockedRuleFmt + "$client=%s"
		blockedRuleQTypeFmt    = blockedRuleFmt + "$dnstype=%s"
		blockedRuleCliQTypeFmt = blockedRuleCliFmt + ",dnstype=%s"
	)

	var (
		allowedRule            = fmt.Sprintf(allowedRuleFmt, allowedHost)
		blockedRule            = fmt.Sprintf(blockedRuleFmt, blockedHost)
		blockedClientRule      = fmt.Sprintf(blockedRuleCliFmt, cliHost, cliName)
		blockedQTypeRule       = fmt.Sprintf(blockedRuleQTypeFmt, qTypeHost, "CNAME")
		blockedClientQTypeRule = fmt.Sprintf(blockedRuleCliQTypeFmt, cliQTypeHost, cliName, "CNAME")

		notFilteredURL        = fmt.Sprintf(hostFmt, notFilteredHost)
		allowedURL            = fmt.Sprintf(hostFmt, allowedHost)
		blockedURL            = fmt.Sprintf(hostFmt, blockedHost)
		blockedClientURL      = fmt.Sprintf(hostCliFmt, cliHost, cliID)
		allowedQTypeURL       = fmt.Sprintf(hostQTypeFmt, qTypeHost, "AAAA")
		blockedQTypeURL       = fmt.Sprintf(hostQTypeFmt, qTypeHost, "CNAME")
		allowedClientQTypeURL = fmt.Sprintf(hostCliQTypeFmt, cliQTypeHost, cliID, "AAAA")
		blockedClientQTypeURL = fmt.Sprintf(hostCliQTypeFmt, cliQTypeHost, cliID, "CNAME")
	)

	rules := []string{
		allowedRule,
		blockedRule,
		blockedClientRule,
		blockedQTypeRule,
		blockedClientQTypeRule,
	}
	rulesData := strings.Join(rules, "\n")

	filters := []Filter{{
		ID: 0, Data: []byte(rulesData),
	}}

	clientNames := map[string]string{
		cliID: cliName,
	}

	dnsFilter, err := New(&Config{
		BlockedServices: &BlockedServices{
			Schedule: schedule.EmptyWeekly(),
		},
		ApplyClientFiltering: func(clientID string, cliAddr netip.Addr, setts *Settings) {
			setts.ClientName = clientNames[clientID]
		},
	}, filters)
	require.NoError(t, err)

	testCases := []struct {
		name string
		url  string
		want *checkHostResp
	}{{
		name: "not_filtered",
		url:  notFilteredURL,
		want: &checkHostResp{
			Reason: reasonNames[NotFilteredNotFound],
			Rule:   "",
			Rules:  []*checkHostRespRule{},
		},
	}, {
		name: "allowed",
		url:  allowedURL,
		want: &checkHostResp{
			Reason: reasonNames[NotFilteredAllowList],
			Rule:   allowedRule,
			Rules: []*checkHostRespRule{{
				Text: allowedRule,
			}},
		},
	}, {
		name: "blocked",
		url:  blockedURL,
		want: &checkHostResp{
			Reason: reasonNames[FilteredBlockList],
			Rule:   blockedRule,
			Rules: []*checkHostRespRule{{
				Text: blockedRule,
			}},
		},
	}, {
		name: "blocked_client",
		url:  blockedClientURL,
		want: &checkHostResp{
			Reason: reasonNames[FilteredBlockList],
			Rule:   blockedClientRule,
			Rules: []*checkHostRespRule{{
				Text: blockedClientRule,
			}},
		},
	}, {
		name: "allowed_qtype",
		url:  allowedQTypeURL,
		want: &checkHostResp{
			Reason: reasonNames[NotFilteredNotFound],
			Rule:   "",
			Rules:  []*checkHostRespRule{},
		},
	}, {
		name: "blocked_qtype",
		url:  blockedQTypeURL,
		want: &checkHostResp{
			Reason: reasonNames[FilteredBlockList],
			Rule:   blockedQTypeRule,
			Rules: []*checkHostRespRule{{
				Text: blockedQTypeRule,
			}},
		},
	}, {
		name: "blocked_client_qtype",
		url:  blockedClientQTypeURL,
		want: &checkHostResp{
			Reason: reasonNames[FilteredBlockList],
			Rule:   blockedClientQTypeRule,
			Rules: []*checkHostRespRule{{
				Text: blockedClientQTypeRule,
			}},
		},
	}, {
		name: "allowed_client_qtype",
		url:  allowedClientQTypeURL,
		want: &checkHostResp{
			Reason: reasonNames[NotFilteredNotFound],
			Rule:   "",
			Rules:  []*checkHostRespRule{},
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tc.url, nil)
			w := httptest.NewRecorder()

			dnsFilter.handleCheckHost(w, r)

			res := &checkHostResp{}
			err = json.NewDecoder(w.Body).Decode(res)
			require.NoError(t, err)

			assert.Equal(t, tc.want, res)
		})
	}
}
