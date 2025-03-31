package dnsforward

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilenameFromURL(t *testing.T) {
	testCases := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "Simple URL",
			url:      "example.com",
			expected: "example.com",
		},
		{
			name:     "With HTTP prefix",
			url:      "http://example.com",
			expected: "example.com",
		},
		{
			name:     "With HTTPS prefix",
			url:      "https://example.com",
			expected: "example.com",
		},
		{
			name:     "With path and parameters",
			url:      "https://example.com/path/to/resource?param=value",
			expected: "example.com_path_to_resource_param_value",
		},
		{
			name:     "With special characters",
			url:      "https://example.com/some:path?param=value&other=123",
			expected: "example.com_some_path_param_value_other_123",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := filenameFromURL(tc.url)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseRuleset(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	manager := newRulesetManager(tempDir)

	// Create test file
	testFilePath := filepath.Join(tempDir, "test_ruleset.txt")
	testContent := `
# This is a comment
example.com
# Another comment
test.org

invalid.com # Inline comment will be preserved
   trimmed.com   
`

	err := os.WriteFile(testFilePath, []byte(testContent), 0o644)
	require.NoError(t, err)

	// Test parsing
	domains, err := manager.parseRuleset(testFilePath)
	require.NoError(t, err)

	expected := []string{
		"example.com",
		"test.org",
		"invalid.com # Inline comment will be preserved",
		"trimmed.com",
	}
	assert.ElementsMatch(t, expected, domains)

	// Test non-existent file
	_, err = manager.parseRuleset(filepath.Join(tempDir, "nonexistent.txt"))
	assert.Error(t, err)
}

func TestDownloadRuleset(t *testing.T) {
	// Setup test server
	testContent := "example.com\ntest.org"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, testContent)
	}))
	defer server.Close()

	// Create temporary directory
	tempDir := t.TempDir()
	manager := newRulesetManager(tempDir)

	// Test download
	filename, err := manager.downloadRuleset(server.URL)
	require.NoError(t, err)

	// Verify file content
	content, err := os.ReadFile(filename)
	require.NoError(t, err)
	assert.Equal(t, testContent, string(content))

	// Test HTTP error
	invalidServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer invalidServer.Close()

	_, err = manager.downloadRuleset(invalidServer.URL)
	assert.Error(t, err)
}

func TestPrepareAlternateUpstreams(t *testing.T) {
	// Setup test server
	testContent := "example.com\ntest.org"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, testContent)
	}))
	defer server.Close()

	// Create temporary directory
	tempDir := t.TempDir()

	// Test parameters
	alternateDNS := []string{"8.8.8.8"}
	alternateRulesets := []string{server.URL}
	opts := &upstream.Options{}

	// Test configuration generation
	config, err := prepareAlternateUpstreams(alternateDNS, alternateRulesets, tempDir, opts)
	require.NoError(t, err)
	assert.NotNil(t, config)

	// Verify generated configuration
	assert.Len(t, config.DomainReservedUpstreams, 2)
	assert.Contains(t, config.DomainReservedUpstreams, "example.com")
	assert.Contains(t, config.DomainReservedUpstreams, "test.org")

	// Test empty parameters
	config, err = prepareAlternateUpstreams([]string{}, alternateRulesets, tempDir, opts)
	assert.Nil(t, config)
	assert.Nil(t, err)

	config, err = prepareAlternateUpstreams(alternateDNS, []string{}, tempDir, opts)
	assert.Nil(t, config)
	assert.Nil(t, err)
}
