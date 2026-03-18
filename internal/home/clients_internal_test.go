package home

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/schedule"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/require"
)

// newClientsContainer is a helper that creates a new clients container for
// tests.
func newClientsContainer(t *testing.T) (c *clientsContainer) {
	t.Helper()

	c = &clientsContainer{
		testing: true,
	}

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	err := c.Init(
		ctx,
		testLogger,
		nil,
		client.EmptyDHCP{},
		nil,
		nil,
		&filtering.Config{},
		newSignalHandler(nil, nil),
	)

	require.NoError(t, err)

	return c
}

func TestClientObject_prepareBlockedServices_preservesPendingDynamicIDs(t *testing.T) {
	filtering.PreloadServiceCatalog(context.Background(), &filtering.Config{}, nil)

	o := &clientObject{
		BlockedServices: &filtering.BlockedServices{
			Schedule: schedule.EmptyWeekly(),
			IDs:      []string{"dynamic_service"},
		},
	}

	svcs, err := o.prepareBlockedServices(testLogger, "pending-client", filtering.ServicesURLs{"http://pending.example/services.json"})
	require.NoError(t, err)
	require.Equal(t, []string{"dynamic_service"}, svcs.IDs)
}

func TestClientsContainer_normalizeBlockedServices(t *testing.T) {
	filtering.PreloadServiceCatalog(context.Background(), &filtering.Config{}, nil)

	serviceURL := filteringTestServiceURL(t)
	filtering.PreloadServiceCatalog(context.Background(), &filtering.Config{
		DataDir:     t.TempDir(),
		ServiceURLs: []string{serviceURL},
		HTTPClient: &http.Client{
			Timeout: testTimeout,
		},
	}, nil)

	clients := newClientsContainer(t)
	ctx := testutil.ContextWithTimeout(t, testTimeout)

	cli := newPersistentClientWithIDs(t, "dynamic-client", []string{testClientIP1})
	cli.BlockedServices = &filtering.BlockedServices{
		Schedule: schedule.EmptyWeekly(),
		IDs:      []string{"dynamic_service"},
	}

	err := clients.storage.Add(ctx, cli)
	require.NoError(t, err)

	filtering.PreloadServiceCatalog(context.Background(), &filtering.Config{}, nil)

	require.True(t, clients.normalizeBlockedServices())

	var got *client.Persistent
	clients.storage.RangeByName(func(c *client.Persistent) (cont bool) {
		got = c.ShallowClone()

		return true
	})

	require.NotNil(t, got)
	require.Empty(t, got.BlockedServices.IDs)
}

func filteringTestServiceURL(t *testing.T) string {
	t.Helper()

	return filteringServeHTTPLocally(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`{
			"blocked_services": [{
				"id": "dynamic_service",
				"name": "Dynamic Service",
				"rules": ["||dynamic.example^"]
			}]
		}`))
		require.NoError(t, err)
	}))
}

func filteringServeHTTPLocally(t *testing.T, h http.Handler) string {
	t.Helper()

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	return srv.URL
}
