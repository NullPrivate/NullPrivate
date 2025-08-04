package querylog

import (
	"context"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilteringStatusStr(t *testing.T) {
	t.Run("newLogEntry_sets_status", func(t *testing.T) {
		testCases := []struct {
			name           string
			reason         filtering.Reason
			isFiltered     bool
			expectedStatus string
		}{
			{
				name:           "blocked",
				reason:         filtering.FilteredBlockList,
				isFiltered:     true,
				expectedStatus: filteringStatusBlocked,
			},
			{
				name:           "blocked_parental",
				reason:         filtering.FilteredParental,
				isFiltered:     true,
				expectedStatus: filteringStatusBlockedParental,
			},
			{
				name:           "blocked_safebrowsing",
				reason:         filtering.FilteredSafeBrowsing,
				isFiltered:     true,
				expectedStatus: filteringStatusBlockedSafebrowsing,
			},
			{
				name:           "blocked_service",
				reason:         filtering.FilteredBlockedService,
				isFiltered:     true,
				expectedStatus: filteringStatusBlockedService,
			},
			{
				name:           "safe_search",
				reason:         filtering.FilteredSafeSearch,
				isFiltered:     true,
				expectedStatus: filteringStatusSafeSearch,
			},
			{
				name:           "whitelisted",
				reason:         filtering.NotFilteredAllowList,
				isFiltered:     false,
				expectedStatus: filteringStatusWhitelisted,
			},
			{
				name:           "rewritten",
				reason:         filtering.Rewritten,
				isFiltered:     false,
				expectedStatus: filteringStatusRewritten,
			},
			{
				name:           "processed",
				reason:         filtering.NotFilteredNotFound,
				isFiltered:     false,
				expectedStatus: filteringStatusProcessed,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				logger := slog.Default()
				ctx := testutil.ContextWithTimeout(t, testTimeout)

				// Create a DNS question
				q := dns.Msg{}
				q.Question = []dns.Question{{
					Name:   "example.com.",
					Qtype:  dns.TypeA,
					Qclass: dns.ClassINET,
				}}

				// Create AddParams with the test case filtering result
				params := &AddParams{
					Question: &q,
					ClientID: "client1",
					ClientIP: net.IPv4(127, 0, 0, 1),
					Result: &filtering.Result{
						Reason:     tc.reason,
						IsFiltered: tc.isFiltered,
					},
				}

				// Create a new log entry
				entry := newLogEntry(ctx, logger, params)

				// Verify that FilteringStatusStr is set correctly
				assert.Equal(t, tc.expectedStatus, entry.FilteringStatusStr)
			})
		}
	})

	t.Run("entryToJSON_includes_status", func(t *testing.T) {
		logger := slog.Default()

		// Create a log entry with a known filtering status
		entry := &logEntry{
			Time:               time.Now(),
			QHost:              "example.com",
			QType:              "A",
			QClass:             "IN",
			ClientID:           "client1",
			IP:                 net.IPv4(127, 0, 0, 1),
			FilteringStatusStr: filteringStatusBlocked,
			Result: filtering.Result{
				Reason:     filtering.FilteredBlockList,
				IsFiltered: true,
			},
		}

		// Create a queryLog instance
		ql := &queryLog{
			logger: logger,
		}

		// Convert the entry to JSON
		jsonObj := ql.entryToJSON(context.Background(), entry, func(ip net.IP) {})

		// Verify that the filtering_status field is included in the JSON
		statusVal, ok := jsonObj["filtering_status"]
		require.True(t, ok, "filtering_status field should be present in JSON")
		assert.Equal(t, filteringStatusBlocked, statusVal)
	})

	t.Run("quickMatch_uses_status", func(t *testing.T) {
		testCases := []struct {
			name           string
			criterionValue string
			statusStr      string
			expectedMatch  bool
		}{
			{
				name:           "exact_match",
				criterionValue: filteringStatusBlocked,
				statusStr:      filteringStatusBlocked,
				expectedMatch:  true,
			},
			{
				name:           "no_match",
				criterionValue: filteringStatusBlocked,
				statusStr:      filteringStatusWhitelisted,
				expectedMatch:  false,
			},
			{
				name:           "all_matches_any",
				criterionValue: filteringStatusAll,
				statusStr:      filteringStatusBlocked,
				expectedMatch:  true,
			},
			{
				name:           "filtered_matches_blocked",
				criterionValue: filteringStatusFiltered,
				statusStr:      filteringStatusBlocked,
				expectedMatch:  true,
			},
			{
				name:           "filtered_matches_rewritten",
				criterionValue: filteringStatusFiltered,
				statusStr:      filteringStatusRewritten,
				expectedMatch:  true,
			},
			{
				name:           "filtered_doesnt_match_whitelisted",
				criterionValue: filteringStatusFiltered,
				statusStr:      filteringStatusWhitelisted,
				expectedMatch:  false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Create a search criterion for filtering status
				criterion := &searchCriterion{
					value:         tc.criterionValue,
					criterionType: ctFilteringStatus,
				}

				// Create a JSON line with the filtering status
				jsonLine := `{"QH":"example.com","FS":"` + tc.statusStr + `"}`

				// Test the quickMatch function
				result := criterion.quickMatch(
					context.Background(),
					slog.Default(),
					jsonLine,
					func(ctx context.Context, logger *slog.Logger, clientID, ip string) *Client { return nil },
				)

				assert.Equal(t, tc.expectedMatch, result)
			})
		}
	})
}
