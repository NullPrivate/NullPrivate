package filtering

import (
	"fmt"
	"net/netip"
	"sync/atomic"
	"testing"

	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	testutil.DiscardLogOutput(m)
}

const (
	sbBlocked = "wmconvirus.narod.ru"
	pcBlocked = "pornhub.com"
)

// Helpers.

func newForTest(t testing.TB, c *Config, filters []Filter) (f *DNSFilter, setts *Settings) {
	setts = &Settings{
		ProtectionEnabled: true,
		FilteringEnabled:  true,
	}
	if c != nil {
		c.SafeBrowsingCacheSize = 10000
		c.ParentalCacheSize = 10000
		c.SafeSearchCacheSize = 1000
		c.CacheTime = 30
		setts.SafeSearchEnabled = c.SafeSearchConf.Enabled
		setts.SafeBrowsingEnabled = c.SafeBrowsingEnabled
		setts.ParentalEnabled = c.ParentalEnabled
	} else {
		// It must not be nil.
		c = &Config{}
	}
	f, err := New(c, filters)
	require.NoError(t, err)

	return f, setts
}

// countingChecker 记录调用次数，并提供可匹配的结果以验证产品策略的短路行为。
type countingChecker struct {
	calls atomic.Uint64
}

func (c *countingChecker) Check(_ string) (block bool, err error) {
	c.calls.Add(1)

	return true, nil
}

func (d *DNSFilter) checkMatchIP(t *testing.T, hostname, ip string, qtype uint16, setts *Settings) {
	t.Helper()

	res, err := d.CheckHost(hostname, qtype, setts)
	require.NoErrorf(t, err, "host %q", hostname, err)
	require.NotEmpty(t, res.Rules, "host %q", hostname)

	assert.Truef(t, res.IsFiltered, "host %q", hostname)

	r := res.Rules[0]
	require.NotNilf(t, r.IP, "Expected ip %s to match, actual: %v", ip, r.IP)

	assert.Equalf(t, ip, r.IP.String(), "host %q", hostname)
}

func (d *DNSFilter) checkMatchEmpty(t *testing.T, hostname string, setts *Settings) {
	t.Helper()

	res, err := d.CheckHost(hostname, dns.TypeA, setts)
	require.NoErrorf(t, err, "host %q", hostname)

	assert.Falsef(t, res.IsFiltered, "host %q", hostname)
	assert.Equal(t, NotFilteredNotFound, res.Reason)
}

func TestDNSFilter_CheckHost_hostRules(t *testing.T) {
	addr := "216.239.38.120"
	addr6 := "::1"
	text := fmt.Sprintf(`  %s  google.com www.google.com   # enforce google's safesearch
%s  ipv6.com
0.0.0.0 block.com
0.0.0.1 host2
0.0.0.2 host2
::1 host2
`,
		addr, addr6)
	filters := []Filter{{
		ID: 0, Data: []byte(text),
	}}
	d, setts := newForTest(t, nil, filters)
	t.Cleanup(d.Close)

	d.checkMatchIP(t, "google.com", addr, dns.TypeA, setts)
	d.checkMatchIP(t, "www.google.com", addr, dns.TypeA, setts)
	d.checkMatchEmpty(t, "subdomain.google.com", setts)
	d.checkMatchEmpty(t, "example.org", setts)

	// IPv4 match.
	d.checkMatchIP(t, "block.com", "0.0.0.0", dns.TypeA, setts)

	// Empty IPv6.
	res, err := d.CheckHost("block.com", dns.TypeAAAA, setts)
	require.NoError(t, err)

	assert.True(t, res.IsFiltered)

	require.Len(t, res.Rules, 1)

	assert.Equal(t, "0.0.0.0 block.com", res.Rules[0].Text)
	assert.Empty(t, res.Rules[0].IP)

	// IPv6 match.
	d.checkMatchIP(t, "ipv6.com", addr6, dns.TypeAAAA, setts)

	// Empty IPv4.
	res, err = d.CheckHost("ipv6.com", dns.TypeA, setts)
	require.NoError(t, err)

	assert.True(t, res.IsFiltered)

	require.Len(t, res.Rules, 1)

	assert.Equal(t, "::1  ipv6.com", res.Rules[0].Text)
	assert.Empty(t, res.Rules[0].IP)

	// Two IPv4, both must be returned.
	res, err = d.CheckHost("host2", dns.TypeA, setts)
	require.NoError(t, err)

	assert.True(t, res.IsFiltered)

	require.Len(t, res.Rules, 2)

	assert.Equal(t, res.Rules[0].IP, netip.AddrFrom4([4]byte{0, 0, 0, 1}))
	assert.Equal(t, res.Rules[1].IP, netip.AddrFrom4([4]byte{0, 0, 0, 2}))

	// One IPv6 address.
	res, err = d.CheckHost("host2", dns.TypeAAAA, setts)
	require.NoError(t, err)

	assert.True(t, res.IsFiltered)

	require.Len(t, res.Rules, 1)

	assert.Equal(t, res.Rules[0].IP, netutil.IPv6Localhost())
}

// Safe Browsing.

func TestSafeBrowsing(t *testing.T) {
	sbChecker := &countingChecker{}

	d, setts := newForTest(t, &Config{
		SafeBrowsingEnabled: true,
		SafeBrowsingChecker: sbChecker,
	}, nil)
	t.Cleanup(d.Close)

	d.checkMatchEmpty(t, sbBlocked, setts)
	d.checkMatchEmpty(t, "test."+sbBlocked, setts)
	d.checkMatchEmpty(t, "yandex.ru", setts)
	d.checkMatchEmpty(t, pcBlocked, setts)

	// 重复请求同样由产品策略放行。
	d.checkMatchEmpty(t, sbBlocked, setts)
	d.checkMatchEmpty(t, pcBlocked, setts)
	assert.Zero(t, sbChecker.calls.Load())
}

func TestParallelSB(t *testing.T) {
	sbChecker := &countingChecker{}
	d, setts := newForTest(t, &Config{
		SafeBrowsingEnabled: true,
		SafeBrowsingChecker: sbChecker,
	}, nil)
	t.Cleanup(d.Close)

	t.Run("group", func(t *testing.T) {
		for i := range 100 {
			t.Run(fmt.Sprintf("aaa%d", i), func(t *testing.T) {
				t.Parallel()
				d.checkMatchEmpty(t, sbBlocked, setts)
				d.checkMatchEmpty(t, "test."+sbBlocked, setts)
				d.checkMatchEmpty(t, "yandex.ru", setts)
				d.checkMatchEmpty(t, pcBlocked, setts)
			})
		}
	})
	assert.Zero(t, sbChecker.calls.Load())
}

// Parental.

func TestParentalControl(t *testing.T) {
	pcChecker := &countingChecker{}
	d, setts := newForTest(t, &Config{
		ParentalEnabled:        true,
		ParentalControlChecker: pcChecker,
	}, nil)
	t.Cleanup(d.Close)

	d.checkMatchEmpty(t, pcBlocked, setts)
	d.checkMatchEmpty(t, "www."+pcBlocked, setts)
	d.checkMatchEmpty(t, "www.yandex.ru", setts)
	d.checkMatchEmpty(t, "yandex.ru", setts)
	d.checkMatchEmpty(t, "api.jquery.com", setts)

	// 重复请求同样由产品策略放行。
	d.checkMatchEmpty(t, pcBlocked, setts)
	d.checkMatchEmpty(t, "yandex.ru", setts)
	assert.Zero(t, pcChecker.calls.Load())
}

// TestDNSFilter_productPolicy 验证历史配置、直接请求及有效规则共同工作。
func TestDNSFilter_productPolicy(t *testing.T) {
	sbChecker, pcChecker := &countingChecker{}, &countingChecker{}
	conf := &Config{
		ProtectionEnabled:      true,
		FilteringEnabled:       true,
		SafeBrowsingEnabled:    true,
		ParentalEnabled:        true,
		SafeBrowsingChecker:    sbChecker,
		ParentalControlChecker: pcChecker,
		SafeSearchConf:         SafeSearchConfig{Enabled: true, Google: true},
	}
	d, setts := newForTest(t, conf, []Filter{{ID: 1, Data: []byte("||ads.example^\n")}})
	t.Cleanup(d.Close)
	d.SetEnabled(true)

	assert.False(t, conf.SafeBrowsingEnabled)
	assert.False(t, conf.ParentalEnabled)
	require.True(t, setts.SafeBrowsingEnabled)
	require.True(t, setts.ParentalEnabled)

	// 模拟内存中保留的历史开关，验证请求、状态和持久化边界各自遵循产品策略。
	d.confMu.Lock()
	d.conf.SafeBrowsingEnabled = true
	d.conf.ParentalEnabled = true
	d.confMu.Unlock()

	for _, host := range []string{sbBlocked, pcBlocked, "www.example.org"} {
		d.checkMatchEmpty(t, host, setts)
	}
	res, err := d.CheckHost("ads.example", dns.TypeA, setts)
	require.NoError(t, err)
	assert.True(t, res.IsFiltered)
	assert.Equal(t, FilteredBlockList, res.Reason)
	assert.Zero(t, sbChecker.calls.Load())
	assert.Zero(t, pcChecker.calls.Load())

	settings := d.Settings()
	assert.False(t, settings.SafeBrowsingEnabled)
	assert.False(t, settings.ParentalEnabled)
	assert.True(t, settings.FilteringEnabled)
	assert.True(t, settings.SafeSearchEnabled)

	var disk Config
	d.WriteDiskConfig(&disk)
	assert.False(t, disk.SafeBrowsingEnabled)
	assert.False(t, disk.ParentalEnabled)
	assert.True(t, disk.ProtectionEnabled)
	assert.True(t, disk.FilteringEnabled)
	assert.Equal(t, conf.SafeSearchConf, disk.SafeSearchConf)
}

// Filtering.

func TestMatching(t *testing.T) {
	const nl = "\n"
	const (
		blockingRules  = `||example.org^` + nl
		allowlistRules = `||example.org^` + nl + `@@||test.example.org` + nl
		importantRules = `@@||example.org^` + nl + `||test.example.org^$important` + nl
		regexRules     = `/example\.org/` + nl + `@@||test.example.org^` + nl
		maskRules      = `test*.example.org^` + nl + `exam*.com` + nl
		dnstypeRules   = `||example.org^$dnstype=AAAA` + nl + `@@||test.example.org^` + nl
	)
	testCases := []struct {
		name           string
		rules          string
		host           string
		wantReason     Reason
		wantIsFiltered bool
		wantDNSType    uint16
	}{{
		name:           "sanity",
		rules:          "||doubleclick.net^",
		host:           "www.doubleclick.net",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "sanity",
		rules:          "||doubleclick.net^",
		host:           "nodoubleclick.net",
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "sanity",
		rules:          "||doubleclick.net^",
		host:           "doubleclick.net.ru",
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "sanity",
		rules:          "||doubleclick.net^",
		host:           sbBlocked,
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "blocking",
		rules:          blockingRules,
		host:           "example.org",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "blocking",
		rules:          blockingRules,
		host:           "test.example.org",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "blocking",
		rules:          blockingRules,
		host:           "test.test.example.org",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "blocking",
		rules:          blockingRules,
		host:           "testexample.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "blocking",
		rules:          blockingRules,
		host:           "onemoreexample.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "allowlist",
		rules:          allowlistRules,
		host:           "example.org",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "allowlist",
		rules:          allowlistRules,
		host:           "test.example.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredAllowList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "allowlist",
		rules:          allowlistRules,
		host:           "test.test.example.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredAllowList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "allowlist",
		rules:          allowlistRules,
		host:           "testexample.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "allowlist",
		rules:          allowlistRules,
		host:           "onemoreexample.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "important",
		rules:          importantRules,
		host:           "example.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredAllowList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "important",
		rules:          importantRules,
		host:           "test.example.org",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "important",
		rules:          importantRules,
		host:           "test.test.example.org",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "important",
		rules:          importantRules,
		host:           "testexample.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "important",
		rules:          importantRules,
		host:           "onemoreexample.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "regex",
		rules:          regexRules,
		host:           "example.org",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "regex",
		rules:          regexRules,
		host:           "test.example.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredAllowList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "regex",
		rules:          regexRules,
		host:           "test.test.example.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredAllowList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "regex",
		rules:          regexRules,
		host:           "testexample.org",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "regex",
		rules:          regexRules,
		host:           "onemoreexample.org",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "mask",
		rules:          maskRules,
		host:           "test.example.org",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "mask",
		rules:          maskRules,
		host:           "test2.example.org",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "mask",
		rules:          maskRules,
		host:           "example.com",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "mask",
		rules:          maskRules,
		host:           "exampleeee.com",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "mask",
		rules:          maskRules,
		host:           "onemoreexamsite.com",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "mask",
		rules:          maskRules,
		host:           "example.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "mask",
		rules:          maskRules,
		host:           "testexample.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "mask",
		rules:          maskRules,
		host:           "example.co.uk",
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "dnstype",
		rules:          dnstypeRules,
		host:           "onemoreexample.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "dnstype",
		rules:          dnstypeRules,
		host:           "example.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredNotFound,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "dnstype",
		rules:          dnstypeRules,
		host:           "example.org",
		wantIsFiltered: true,
		wantReason:     FilteredBlockList,
		wantDNSType:    dns.TypeAAAA,
	}, {
		name:           "dnstype",
		rules:          dnstypeRules,
		host:           "test.example.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredAllowList,
		wantDNSType:    dns.TypeA,
	}, {
		name:           "dnstype",
		rules:          dnstypeRules,
		host:           "test.example.org",
		wantIsFiltered: false,
		wantReason:     NotFilteredAllowList,
		wantDNSType:    dns.TypeAAAA,
	}}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s-%s", tc.name, tc.host), func(t *testing.T) {
			filters := []Filter{{ID: 0, Data: []byte(tc.rules)}}
			d, setts := newForTest(t, nil, filters)
			t.Cleanup(d.Close)

			res, err := d.CheckHost(tc.host, tc.wantDNSType, setts)
			require.NoError(t, err)

			assert.Equalf(t, tc.wantIsFiltered, res.IsFiltered, "Hostname %s has wrong result (%v must be %v)", tc.host, res.IsFiltered, tc.wantIsFiltered)
			assert.Equalf(t, tc.wantReason, res.Reason, "Hostname %s has wrong reason (%v must be %v)", tc.host, res.Reason, tc.wantReason)
		})
	}
}

func TestWhitelist(t *testing.T) {
	rules := `||host1^
||host2^
`
	filters := []Filter{{
		ID: 0, Data: []byte(rules),
	}}

	whiteRules := `||host1^
||host3^
`
	whiteFilters := []Filter{{
		ID: 0, Data: []byte(whiteRules),
	}}
	d, setts := newForTest(t, nil, filters)

	err := d.setFilters(filters, whiteFilters, false)
	require.NoError(t, err)

	t.Cleanup(d.Close)

	// Matched by white filter.
	res, err := d.CheckHost("host1", dns.TypeA, setts)
	require.NoError(t, err)

	assert.False(t, res.IsFiltered)
	assert.Equal(t, res.Reason, NotFilteredAllowList)

	require.Len(t, res.Rules, 1)

	assert.Equal(t, "||host1^", res.Rules[0].Text)

	// Not matched by white filter, but matched by block filter.
	res, err = d.CheckHost("host2", dns.TypeA, setts)
	require.NoError(t, err)

	assert.True(t, res.IsFiltered)
	assert.Equal(t, res.Reason, FilteredBlockList)

	require.Len(t, res.Rules, 1)

	assert.Equal(t, "||host2^", res.Rules[0].Text)
}

// Client Settings.

func applyClientSettings(setts *Settings) {
	setts.FilteringEnabled = false
	setts.ParentalEnabled = false
	setts.SafeBrowsingEnabled = true

	rule, _ := rules.NewNetworkRule("||facebook.com^", 0)
	s := ServiceEntry{}
	s.Name = "facebook"
	s.Rules = []*rules.NetworkRule{rule}
	setts.ServicesRules = append(setts.ServicesRules, s)
}

func TestClientSettings(t *testing.T) {
	sbChecker, pcChecker := &countingChecker{}, &countingChecker{}
	d, setts := newForTest(t,
		&Config{
			ParentalEnabled:        true,
			SafeBrowsingEnabled:    false,
			SafeBrowsingChecker:    sbChecker,
			ParentalControlChecker: pcChecker,
		},
		[]Filter{{
			ID: 0, Data: []byte("||example.org^\n"),
		}},
	)
	t.Cleanup(d.Close)

	type testCase struct {
		name   string
		host   string
		before Reason
		after  Reason
	}
	testCases := []testCase{{
		name:   "filters",
		host:   "example.org",
		before: FilteredBlockList,
		after:  NotFilteredNotFound,
	}, {
		name:   "parental",
		host:   pcBlocked,
		before: NotFilteredNotFound,
		after:  NotFilteredNotFound,
	}, {
		name:   "safebrowsing",
		host:   sbBlocked,
		before: NotFilteredNotFound,
		after:  NotFilteredNotFound,
	}, {
		name:   "additional_rules",
		host:   "facebook.com",
		before: NotFilteredNotFound,
		after:  FilteredBlockedService,
	}}

	makeTester := func(tc testCase, wantReason Reason) func(t *testing.T) {
		return func(t *testing.T) {
			t.Helper()

			r, err := d.CheckHost(tc.host, dns.TypeA, setts)
			require.NoError(t, err)

			assert.Equal(t, wantReason != NotFilteredNotFound, r.IsFiltered)
			assert.Equal(t, wantReason, r.Reason)
		}
	}

	// Check behaviour without any per-client settings, then apply per-client
	// settings and check behavior once again.
	for _, tc := range testCases {
		t.Run(tc.name+"/global", makeTester(tc, tc.before))
	}

	applyClientSettings(setts)

	for _, tc := range testCases {
		t.Run(tc.name+"/client", makeTester(tc, tc.after))
	}
	assert.Zero(t, sbChecker.calls.Load())
	assert.Zero(t, pcChecker.calls.Load())
}

// BenchmarkCheckHost_productPolicy 衡量产品策略下的正常放行路径。
// 哈希前缀算法及缓存行为由 hashprefix 包的测试独立验证。
func BenchmarkCheckHost_productPolicy(b *testing.B) {
	sbChecker, pcChecker := &countingChecker{}, &countingChecker{}
	d, setts := newForTest(b, &Config{
		SafeBrowsingEnabled:    true,
		ParentalEnabled:        true,
		SafeBrowsingChecker:    sbChecker,
		ParentalControlChecker: pcChecker,
	}, nil)
	b.Cleanup(d.Close)

	var res Result
	var err error
	b.ReportAllocs()
	for b.Loop() {
		res, err = d.CheckHost(sbBlocked, dns.TypeA, setts)
	}

	require.NoError(b, err)
	assert.Equal(b, Result{}, res)
	assert.Zero(b, sbChecker.calls.Load())
	assert.Zero(b, pcChecker.calls.Load())
}

func BenchmarkCheckHost_productPolicy_parallel(b *testing.B) {
	sbChecker, pcChecker := &countingChecker{}, &countingChecker{}
	d, setts := newForTest(b, &Config{
		SafeBrowsingEnabled:    true,
		ParentalEnabled:        true,
		SafeBrowsingChecker:    sbChecker,
		ParentalControlChecker: pcChecker,
	}, nil)
	b.Cleanup(d.Close)

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var res Result
		var err error
		for pb.Next() {
			res, err = d.CheckHost(sbBlocked, dns.TypeA, setts)
		}
		assert.NoError(b, err)
		assert.Equal(b, Result{}, res)
	})
	b.StopTimer()
	assert.Zero(b, sbChecker.calls.Load())
	assert.Zero(b, pcChecker.calls.Load())
}
