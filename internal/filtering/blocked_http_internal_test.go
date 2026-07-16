package filtering

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/schedule"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDNSFilter_handleBlockedServicesAll_recoversPendingCatalog(t *testing.T) {
	resetBlockedServicesStartupState(t)

	var hits atomic.Int32

	serviceURL := serveHTTPLocally(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)

		_, err := w.Write([]byte(`{
			"blocked_services": [{
				"id": "dynamic_service",
				"name": "Dynamic Service",
				"icon_base64": "AQID",
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
	}, nil)
	require.NoError(t, err)
	t.Cleanup(d.Close)

	r := httptest.NewRequest(http.MethodGet, "/control/blocked_services/all", nil)
	w := httptest.NewRecorder()

	d.handleBlockedServicesAll(w, r)
	require.Equal(t, http.StatusOK, w.Code)
	assert.NotZero(t, hits.Load())
	assert.True(t, IsBlockedServicesCatalogReady(d.cloneConfiguredServiceURLs()))

	var resp struct {
		BlockedServices []blockedService `json:"blocked_services"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotEmpty(t, resp.BlockedServices)
	assert.Equal(t, "dynamic_service", resp.BlockedServices[0].ID)
	assert.Equal(t, []byte{1, 2, 3}, resp.BlockedServices[0].IconBase64)
}

func TestDNSFilter_handleBlockedServicesUpdate_clearsPendingDynamicIDsOnExplicitEmpty(t *testing.T) {
	resetBlockedServicesStartupState(t)

	conf := &Config{
		DataDir: t.TempDir(),
		ServiceURLs: []string{serveHTTPLocally(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "upstream failure", http.StatusInternalServerError)
		}))},
		ConfigModified: func() {},
		HTTPClient: &http.Client{
			Timeout: testTimeout,
		},
		BlockedServices: &BlockedServices{
			Schedule: schedule.EmptyWeekly(),
			IDs:      []string{"dynamic_service"},
		},
	}

	d, err := New(conf, nil)
	require.NoError(t, err)
	t.Cleanup(d.Close)

	body := bytes.NewBufferString(`{"ids":[],"schedule":{"time_zone":"UTC"}}`)
	r := httptest.NewRequest(http.MethodPut, "/control/blocked_services/update", body)
	w := httptest.NewRecorder()

	d.handleBlockedServicesUpdate(w, r)
	require.Equal(t, http.StatusOK, w.Code)

	require.Empty(t, d.conf.BlockedServices.IDs)
}

func TestDNSFilter_handleBlockedServicesUpdate_preservesPendingDynamicIDsWhenIDsOmitted(t *testing.T) {
	resetBlockedServicesStartupState(t)

	conf := &Config{
		DataDir: t.TempDir(),
		ServiceURLs: []string{serveHTTPLocally(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "upstream failure", http.StatusInternalServerError)
		}))},
		ConfigModified: func() {},
		HTTPClient: &http.Client{
			Timeout: testTimeout,
		},
		BlockedServices: &BlockedServices{
			Schedule: schedule.EmptyWeekly(),
			IDs:      []string{"dynamic_service"},
		},
	}

	d, err := New(conf, nil)
	require.NoError(t, err)
	t.Cleanup(d.Close)

	body := bytes.NewBufferString(`{"schedule":{"time_zone":"UTC"}}`)
	r := httptest.NewRequest(http.MethodPut, "/control/blocked_services/update", body)
	w := httptest.NewRecorder()

	d.handleBlockedServicesUpdate(w, r)
	require.Equal(t, http.StatusOK, w.Code)

	require.Equal(t, []string{"dynamic_service"}, d.conf.BlockedServices.IDs)
}

func TestDNSFilter_handleBlockedServicesUpdate_replacesPendingDynamicIDsOnExplicitList(t *testing.T) {
	resetBlockedServicesStartupState(t)

	conf := &Config{
		DataDir: t.TempDir(),
		ServiceURLs: []string{serveHTTPLocally(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "upstream failure", http.StatusInternalServerError)
		}))},
		ConfigModified: func() {},
		HTTPClient: &http.Client{
			Timeout: testTimeout,
		},
		BlockedServices: &BlockedServices{
			Schedule: schedule.EmptyWeekly(),
			IDs:      []string{"dynamic_service"},
		},
	}

	d, err := New(conf, nil)
	require.NoError(t, err)
	t.Cleanup(d.Close)

	body := bytes.NewBufferString(`{"ids":["new_dynamic"],"schedule":{"time_zone":"UTC"}}`)
	r := httptest.NewRequest(http.MethodPut, "/control/blocked_services/update", body)
	w := httptest.NewRecorder()

	d.handleBlockedServicesUpdate(w, r)
	require.Equal(t, http.StatusOK, w.Code)

	require.Equal(t, []string{"new_dynamic"}, d.conf.BlockedServices.IDs)
}

func TestDNSFilter_handleBlockedServicesSet_clearsPendingDynamicIDsOnExplicitEmpty(t *testing.T) {
	resetBlockedServicesStartupState(t)

	conf := &Config{
		DataDir: t.TempDir(),
		ServiceURLs: []string{serveHTTPLocally(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "upstream failure", http.StatusInternalServerError)
		}))},
		ConfigModified: func() {},
		HTTPClient: &http.Client{
			Timeout: testTimeout,
		},
		BlockedServices: &BlockedServices{
			Schedule: schedule.EmptyWeekly(),
			IDs:      []string{"dynamic_service"},
		},
	}

	d, err := New(conf, nil)
	require.NoError(t, err)
	t.Cleanup(d.Close)

	r := httptest.NewRequest(http.MethodPost, "/control/blocked_services/set", bytes.NewBufferString(`[]`))
	w := httptest.NewRecorder()

	d.handleBlockedServicesSet(w, r)
	require.Equal(t, http.StatusOK, w.Code)

	require.Empty(t, d.conf.BlockedServices.IDs)
}

func TestDNSFilter_handleBlockedServicesUpdate_readyPreservesWhenIDsOmittedAndClearsOnExplicitEmpty(t *testing.T) {
	resetBlockedServicesStartupState(t)

	conf := &Config{
		DataDir:        t.TempDir(),
		ConfigModified: func() {},
		BlockedServices: &BlockedServices{
			Schedule: schedule.EmptyWeekly(),
			IDs:      []string{"bilibili"},
		},
	}

	d, err := New(conf, nil)
	require.NoError(t, err)
	t.Cleanup(d.Close)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(
		http.MethodPut,
		"/control/blocked_services/update",
		bytes.NewBufferString(`{"schedule":{"time_zone":"UTC"}}`),
	)
	d.handleBlockedServicesUpdate(w, r)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, []string{"bilibili"}, d.conf.BlockedServices.IDs)

	w = httptest.NewRecorder()
	r = httptest.NewRequest(
		http.MethodPut,
		"/control/blocked_services/update",
		bytes.NewBufferString(`{"ids":[],"schedule":{"time_zone":"UTC"}}`),
	)
	d.handleBlockedServicesUpdate(w, r)
	require.Equal(t, http.StatusOK, w.Code)
	require.Empty(t, d.conf.BlockedServices.IDs)
}

func TestDNSFilter_handleServiceURLsSet_rejectsFailedCatalogAndKeepsState(t *testing.T) {
	resetBlockedServicesStartupState(t)

	const (
		dynamicID   = "dynamic_service"
		dynamicHost = "dynamic.example"
	)

	currentURL := serveHTTPLocally(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`{
			"blocked_services": [{
				"id": "dynamic_service",
				"name": "Dynamic Service",
				"rules": ["||dynamic.example^"]
			}]
		}`))
		require.NoError(t, err)
	}))

	nextURL := serveHTTPLocally(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream failure", http.StatusInternalServerError)
	}))

	conf := &Config{
		DataDir:     t.TempDir(),
		ServiceURLs: []string{currentURL},
		HTTPClient: &http.Client{
			Timeout: testTimeout,
		},
		BlockedServices: &BlockedServices{
			Schedule: schedule.EmptyWeekly(),
			IDs:      []string{dynamicID},
		},
	}

	PreloadServiceCatalog(t.Context(), conf, nil)

	d, err := New(conf, nil)
	require.NoError(t, err)
	t.Cleanup(d.Close)

	body := bytes.NewBufferString(`{"service_urls":["` + nextURL + `"]}`)
	r := httptest.NewRequest(http.MethodPost, "/control/blocked_services/urls/set", body)
	w := httptest.NewRecorder()

	d.handleServiceURLsSet(w, r)
	require.Equal(t, http.StatusBadGateway, w.Code)
	assert.Equal(t, []string{currentURL}, []string(d.cloneConfiguredServiceURLs()))

	setts := &Settings{ProtectionEnabled: true}
	d.ApplyBlockedServices(setts)

	res, matchErr := matchBlockedServicesRules(dynamicHost, dns.TypeA, setts)
	require.NoError(t, matchErr)
	assert.Equal(t, FilteredBlockedService, res.Reason)
	assert.Equal(t, dynamicID, res.ServiceName)
}

func TestDNSFilter_handleServiceURLsSet_emptySwitchesToBuiltin(t *testing.T) {
	resetBlockedServicesStartupState(t)

	const dynamicID = "dynamic_service"

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

	d, err := New(conf, nil)
	require.NoError(t, err)
	t.Cleanup(d.Close)

	r := httptest.NewRequest(http.MethodPost, "/control/blocked_services/urls/set", bytes.NewBufferString(`{"service_urls":[]}`))
	w := httptest.NewRecorder()

	d.handleServiceURLsSet(w, r)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, d.cloneConfiguredServiceURLs())
	assert.Empty(t, d.conf.BlockedServices.IDs)

	setts := &Settings{ProtectionEnabled: true}
	d.ApplyBlockedServices(setts)
	assert.Empty(t, setts.ServicesRules)
}

func TestDNSFilter_handleBlockedServicesReload_failureKeepsActiveCatalog(t *testing.T) {
	resetBlockedServicesStartupState(t)

	const (
		dynamicID   = "dynamic_service"
		dynamicHost = "dynamic.example"
	)

	var fail atomic.Bool

	serviceURL := serveHTTPLocally(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if fail.Load() {
			http.Error(w, "upstream failure", http.StatusInternalServerError)

			return
		}

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

	d, err := New(conf, nil)
	require.NoError(t, err)
	t.Cleanup(d.Close)

	fail.Store(true)

	r := httptest.NewRequest(http.MethodPost, "/control/blocked_services/reload", nil)
	w := httptest.NewRecorder()

	d.handleBlockedServicesReload(w, r)
	require.Equal(t, http.StatusBadGateway, w.Code)

	setts := &Settings{ProtectionEnabled: true}
	d.ApplyBlockedServices(setts)

	res, matchErr := matchBlockedServicesRules(dynamicHost, dns.TypeA, setts)
	require.NoError(t, matchErr)
	assert.Equal(t, FilteredBlockedService, res.Reason)
	assert.Equal(t, dynamicID, res.ServiceName)
}
