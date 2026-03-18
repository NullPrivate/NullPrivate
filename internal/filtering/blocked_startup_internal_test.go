package filtering

import (
	"net/http"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/schedule"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetBlockedServicesStartupState(t *testing.T) {
	t.Helper()

	blockedCatalogMu.Lock()
	blockedCatalog = blockedServicesCatalog{}
	blockedCatalogMu.Unlock()

	t.Cleanup(func() {
		blockedCatalogMu.Lock()
		blockedCatalog = blockedServicesCatalog{}
		blockedCatalogMu.Unlock()

		InitModule()
	})
}

func TestDNSFilter_New_preservesBuiltInBlockedServiceIDsWithoutInitModule(t *testing.T) {
	resetBlockedServicesStartupState(t)

	const (
		builtInID   = "bilibili"
		builtInHost = "www.bilibili.com"
	)

	d, err := New(&Config{
		DataDir: t.TempDir(),
		BlockedServices: &BlockedServices{
			Schedule: schedule.EmptyWeekly(),
			IDs:      []string{builtInID},
		},
	}, nil)
	require.NoError(t, err)
	t.Cleanup(d.Close)

	assert.Equal(t, []string{builtInID}, d.conf.BlockedServices.IDs)

	setts := &Settings{
		ProtectionEnabled: true,
	}
	d.ApplyBlockedServices(setts)

	res, matchErr := matchBlockedServicesRules(builtInHost, dns.TypeA, setts)
	require.NoError(t, matchErr)
	assert.Equal(t, FilteredBlockedService, res.Reason)
	assert.Equal(t, builtInID, res.ServiceName)
}

func TestDNSFilter_New_preservesDynamicBlockedServiceIDs(t *testing.T) {
	resetBlockedServicesStartupState(t)

	const (
		dynamicID   = "dynamic_service"
		dynamicHost = "dynamic.example"
	)

	serviceURL := serveHTTPLocally(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`{
			"blocked_services": [{
				"id": "dynamic_service",
				"name": "Dynamic Service",
				"rules": ["||dynamic.example^"]
			}]
		}`))
		require.NoError(t, err)
	}))

	d, err := New(&Config{
		DataDir:     t.TempDir(),
		ServiceURLs: []string{serviceURL},
		HTTPClient: &http.Client{
			Timeout: testTimeout,
		},
		BlockedServices: &BlockedServices{
			Schedule: schedule.EmptyWeekly(),
			IDs:      []string{dynamicID},
		},
	}, nil)
	require.NoError(t, err)
	t.Cleanup(d.Close)

	assert.Equal(t, []string{dynamicID}, d.conf.BlockedServices.IDs)
	assert.False(t, IsBlockedServicesCatalogReady(d.conf.ServiceURLs))

	setts := &Settings{
		ProtectionEnabled: true,
	}
	d.ApplyBlockedServices(setts)

	res, matchErr := matchBlockedServicesRules(dynamicHost, dns.TypeA, setts)
	require.NoError(t, matchErr)
	assert.Zero(t, res.Reason)
}

func TestDNSFilter_New_keepsBuiltInBlockedServicesWhenDynamicRefreshFails(t *testing.T) {
	resetBlockedServicesStartupState(t)

	const (
		builtInID   = "bilibili"
		builtInHost = "www.bilibili.com"
	)

	serviceURL := serveHTTPLocally(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream failure", http.StatusInternalServerError)
	}))

	d, err := New(&Config{
		DataDir:     t.TempDir(),
		ServiceURLs: []string{serviceURL},
		HTTPClient: &http.Client{
			Timeout: testTimeout,
		},
		BlockedServices: &BlockedServices{
			Schedule: schedule.EmptyWeekly(),
			IDs:      []string{builtInID},
		},
	}, nil)
	require.NoError(t, err)
	t.Cleanup(d.Close)

	assert.Equal(t, []string{builtInID}, d.conf.BlockedServices.IDs)

	setts := &Settings{
		ProtectionEnabled: true,
	}
	d.ApplyBlockedServices(setts)

	res, matchErr := matchBlockedServicesRules(builtInHost, dns.TypeA, setts)
	require.NoError(t, matchErr)
	assert.Equal(t, FilteredBlockedService, res.Reason)
	assert.Equal(t, builtInID, res.ServiceName)
}

func TestPreloadServiceCatalog_activatesDynamicCatalog(t *testing.T) {
	resetBlockedServicesStartupState(t)

	const (
		dynamicID   = "dynamic_service"
		dynamicHost = "dynamic.example"
	)

	serviceURL := serveHTTPLocally(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`{
			"blocked_services": [{
				"id": "dynamic_service",
				"name": "Dynamic Service",
				"rules": ["||dynamic.example^"]
			}]
		}`))
		require.NoError(t, err)
	}))

	conf := &Config{
		DataDir:     t.TempDir(),
		ServiceURLs: []string{serviceURL},
		HTTPClient: &http.Client{
			Timeout: testTimeout,
		},
		BlockedServices: &BlockedServices{
			Schedule: schedule.EmptyWeekly(),
			IDs:      []string{dynamicID},
		},
	}

	PreloadServiceCatalog(t.Context(), conf, nil)
	assert.True(t, IsBlockedServicesCatalogReady(conf.ServiceURLs))

	d, err := New(conf, nil)
	require.NoError(t, err)
	t.Cleanup(d.Close)

	setts := &Settings{ProtectionEnabled: true}
	d.ApplyBlockedServices(setts)

	res, matchErr := matchBlockedServicesRules(dynamicHost, dns.TypeA, setts)
	require.NoError(t, matchErr)
	assert.Equal(t, FilteredBlockedService, res.Reason)
	assert.Equal(t, dynamicID, res.ServiceName)
}
