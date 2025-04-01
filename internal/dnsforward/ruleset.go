// Package dnsforward contains a DNS forwarding server.
package dnsforward

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/stringutil"
)

const defaultRulesetsDir = "data/rulesets"

const ruleCacheExpire = 7 * 24 * time.Hour

// rulesetManager manages the download and parsing of rulesets.
type rulesetManager struct {
	rulesetsDir string
}

// newRulesetManager creates a new rulesetManager with the specified directory.
// If dir is empty, the default directory is used.
func newRulesetManager(dir string) *rulesetManager {
	if dir == "" {
		dir = defaultRulesetsDir
	}
	return &rulesetManager{rulesetsDir: dir}
}

// ensureRulesetsDir ensures that the rulesets directory exists.
func (m *rulesetManager) ensureRulesetsDir() error {
	return os.MkdirAll(m.rulesetsDir, 0o750)
}

// isURLAllowed checks if the URL is safe to download from.
// It implements basic validation to prevent potential security issues.
func isURLAllowed(rawURL string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parsing URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme %q", parsedURL.Scheme)
	}

	return nil
}

// downloadRuleset downloads a ruleset from the URL and saves it to a file.
// If the file already exists and is not older than ruleCacheExpire, it is not downloaded again.
func (m *rulesetManager) downloadRuleset(rawURL string) (string, error) {
	if err := m.ensureRulesetsDir(); err != nil {
		return "", fmt.Errorf("creating rulesets directory: %w", err)
	}

	// Security check for URL
	if err := isURLAllowed(rawURL); err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	// Generate a filename based on the URL
	safeFilename := filenameFromURL(rawURL)
	filename := filepath.Join(m.rulesetsDir, safeFilename)

	// Check if the file already exists and is fresh
	if m.isFileExistAndFresh(filename) {
		log.Debug("dnsforward: ruleset %s is fresh, not downloading", rawURL)
		return filename, nil
	}

	log.Debug("dnsforward: downloading ruleset from %s", rawURL)
	return m.fetchAndSaveRuleset(rawURL, filename)
}

// isFileExistAndFresh checks if the file exists and is not older than ruleCacheExpire.
func (m *rulesetManager) isFileExistAndFresh(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) < ruleCacheExpire
}

// fetchAndSaveRuleset downloads a ruleset from URL and saves it to filename.
func (m *rulesetManager) fetchAndSaveRuleset(rawURL, filename string) (string, error) {
	// Create a proper HTTP client for this request
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := m.makeRequest(client, rawURL)
	if err != nil {
		return "", err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Error("dnsforward: failed to close response body: %s", closeErr)
		}
	}()

	return m.saveRulesetToFile(resp.Body, filename)
}

// makeRequest creates and executes an HTTP request.
func (m *rulesetManager) makeRequest(client *http.Client, rawURL string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, doErr := client.Do(req)
	if doErr != nil {
		return nil, fmt.Errorf("downloading ruleset: %w", doErr)
	}

	if resp.StatusCode != http.StatusOK {
		closeErr := resp.Body.Close()
		if closeErr != nil {
			log.Error("dnsforward: failed to close response body: %s", closeErr)
		}
		return nil, fmt.Errorf("downloading ruleset: HTTP status %d", resp.StatusCode)
	}

	return resp, nil
}

// saveRulesetToFile saves the ruleset content to a file.
func (m *rulesetManager) saveRulesetToFile(content io.Reader, filename string) (string, error) {
	f, err := os.OpenFile(filepath.Clean(filename), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", fmt.Errorf("creating ruleset file: %w", err)
	}

	// Use defer with named return values to ensure file closure
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("closing ruleset file: %w", closeErr)
		}

		if err != nil {
			if removeErr := os.Remove(filepath.Clean(filename)); removeErr != nil {
				log.Error("dnsforward: failed to remove incomplete ruleset file %s: %s", filename, removeErr)
			} else {
				log.Debug("dnsforward: removed incomplete ruleset file %s, error: %s", filename, err)
			}
		}
	}()

	if _, err = io.Copy(f, content); err != nil {
		return "", fmt.Errorf("writing ruleset file: %w", err)
	}

	return filename, nil
}

// filenameFromURL generates a safe filename from a URL.
func filenameFromURL(url string) string {
	// Remove common prefixes
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")

	// Replace special characters with underscore
	replacer := strings.NewReplacer(
		"/", "_",
		":", "_",
		"?", "_",
		"&", "_",
		"=", "_",
		" ", "_",
		"..", "_", // Prevent path traversal
		"\\", "_", // Prevent path traversal
	)
	return replacer.Replace(url)
}

// verifyFilePath checks if the given path is within the rulesetsDir directory.
func (m *rulesetManager) verifyFilePath(path string) error {
	absRulesetsDir, err := filepath.Abs(m.rulesetsDir)
	if err != nil {
		return fmt.Errorf("getting absolute path: %w", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("getting absolute path: %w", err)
	}

	if !strings.HasPrefix(absPath, absRulesetsDir) {
		return fmt.Errorf("path %q is outside of rulesets directory %q", path, m.rulesetsDir)
	}

	return nil
}

// parseRuleset parses a ruleset file and returns a list of domains.
func (m *rulesetManager) parseRuleset(filename string) ([]string, error) {
	// Security check for path traversal
	if err := m.verifyFilePath(filename); err != nil {
		return nil, err
	}

	f, openErr := os.Open(filepath.Clean(filename))
	if openErr != nil {
		return nil, fmt.Errorf("opening ruleset file: %w", openErr)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			log.Error("dnsforward: failed to close ruleset file: %s", closeErr)
		}
	}()

	var domains []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		domains = append(domains, line)
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return nil, fmt.Errorf("scanning ruleset file: %w", scanErr)
	}

	return domains, nil
}

// downloadAndParseRuleset downloads and parses a single ruleset file.
// It returns the list of domains from the ruleset or nil if there was an error.
func (m *rulesetManager) downloadAndParseRuleset(rulesetURL string) (domains []string) {
	if IsCommentOrEmpty(rulesetURL) {
		return nil
	}

	filename, err := m.downloadRuleset(rulesetURL)
	if err != nil {
		log.Error("dnsforward: failed to download ruleset %s: %s", rulesetURL, err)
		return nil
	}

	domains, err = m.parseRuleset(filename)
	if err != nil {
		log.Error("dnsforward: failed to parse ruleset %s: %s", filename, err)
		return nil
	}

	log.Debug("dnsforward: parsed %d domains from ruleset %s", len(domains), rulesetURL)
	return domains
}

// prepareAlternateUpstreams prepares alternate upstream configurations based on ruleset files and alternate DNS settings.
func prepareAlternateUpstreams(alternateDNS, alternateRulesets []string, rulesetsDir string, opts *upstream.Options) (*proxy.UpstreamConfig, error) {
	// Early validation checks
	if len(alternateDNS) == 0 || len(alternateRulesets) == 0 {
		return nil, nil
	}

	alternateDNS = stringutil.FilterOut(alternateDNS, IsCommentOrEmpty)
	if len(alternateDNS) == 0 {
		return nil, nil
	}

	// Create a ruleset manager and collect domains from all rulesets
	manager := newRulesetManager(rulesetsDir)
	var allDomains []string

	for _, rulesetURL := range alternateRulesets {
		domains := manager.downloadAndParseRuleset(rulesetURL)
		if len(domains) > 0 {
			allDomains = append(allDomains, domains...)
		}
	}

	if len(allDomains) == 0 {
		return nil, nil
	}

	return createUpstreamConfig(alternateDNS, allDomains, opts)
}

// createUpstreamConfig creates an UpstreamConfig with domain-specific rules
func createUpstreamConfig(alternateDNS, domains []string, opts *upstream.Options) (*proxy.UpstreamConfig, error) {
	uc := &proxy.UpstreamConfig{
		Upstreams:               []upstream.Upstream{},
		DomainReservedUpstreams: map[string][]upstream.Upstream{},
	}

	upstreams, err := proxy.ParseUpstreamsConfig(alternateDNS, opts)
	if err != nil {
		return nil, fmt.Errorf("parsing alternate DNS upstreams: %w", err)
	}

	// Add each domain to the domain-specific upstreams
	for _, domain := range domains {
		uc.DomainReservedUpstreams[domain] = upstreams.Upstreams
	}

	return uc, nil
}
