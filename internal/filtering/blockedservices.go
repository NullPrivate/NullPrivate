package filtering

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
)

// ServicesURLs is the configuration for blocked services URLs
type ServicesURLs []string

// ServiceLoader is responsible for loading and caching blocked services files
type ServiceLoader struct {
	// urls stores the configured service file URLs
	urls ServicesURLs
	// dataDir is the directory for caching service files
	dataDir string
	// services stores the loaded services
	services []blockedService
	// lastRefresh records the most recent update time
	lastRefresh time.Time
	// mu protects the loading process for concurrent safety
	mu sync.RWMutex
	// client is used for downloading service files
	client *http.Client
	// logger is used for logging
	logger *slog.Logger
}

// NewServiceLoader creates a new service loader
func NewServiceLoader(urls ServicesURLs, dataDir string, client *http.Client, logger *slog.Logger) *ServiceLoader {
	return &ServiceLoader{
		urls:     urls,
		dataDir:  dataDir,
		services: nil,
		client:   client,
		logger:   logger,
	}
}

// ServicesDir returns the cache directory for service files
func (s *ServiceLoader) ServicesDir() string {
	return filepath.Join(s.dataDir, "services")
}

// ensureServiceDir ensures that the service file cache directory exists
func (s *ServiceLoader) ensureServiceDir() error {
	dir := s.ServicesDir()
	return os.MkdirAll(dir, 0o755)
}

// LoadServices loads all configured service files
func (s *ServiceLoader) LoadServices(ctx context.Context) ([]blockedService, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// If already loaded and still valid, return the cached version
	if s.services != nil && time.Since(s.lastRefresh) < 7*24*time.Hour {
		return s.services, nil
	}

	if err := s.ensureServiceDir(); err != nil {
		return nil, fmt.Errorf("failed to create service cache directory: %w", err)
	}

	var allServices []blockedService
	for _, url := range s.urls {
		services, err := s.loadFromURL(ctx, url)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to load services from URL", slogutil.KeyError, err, "url", url)
			continue
		}
		allServices = append(allServices, services...)
	}

	if len(allServices) > 0 {
		s.services = allServices
		s.lastRefresh = time.Now()
	}

	return s.services, nil
}

// GetBlockedServices gets all loaded blocked services
func (s *ServiceLoader) GetBlockedServices(ctx context.Context) []blockedService {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.services != nil {
		return s.services
	}

	// If not loaded yet, try to load
	services, err := s.LoadServices(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to load services", slogutil.KeyError, err)
		// If loading fails, return the built-in service list
		return blockedServices
	}

	return services
}

// loadFromURL loads services from a URL, using cache if valid
func (s *ServiceLoader) loadFromURL(ctx context.Context, url string) ([]blockedService, error) {
	cacheFile := s.cacheFileName(url)
	cacheExists, cacheInfo, err := s.checkCache(cacheFile)
	if err != nil {
		return nil, fmt.Errorf("failed to check cache: %w", err)
	}

	// If cache exists and is less than 3 days old, use it
	if cacheExists && time.Since(cacheInfo.ModTime()) < 3*24*time.Hour {
		return s.loadFromFile(cacheFile)
	}

	// Download and update cache
	return s.downloadAndCache(ctx, url, cacheFile)
}

// cacheFileName generates a cache filename based on the URL
func (s *ServiceLoader) cacheFileName(url string) string {
	// Using a simple method to generate filename, production may need more complex handling
	fileName := fmt.Sprintf("services_%d.json", hash(url))
	return filepath.Join(s.ServicesDir(), fileName)
}

// hash simply converts a URL to an integer
func hash(s string) uint32 {
	h := uint32(0)
	for i := 0; i < len(s); i++ {
		h = h*31 + uint32(s[i])
	}
	return h
}

// checkCache checks if a cache file exists
func (s *ServiceLoader) checkCache(cacheFile string) (bool, os.FileInfo, error) {
	info, err := os.Stat(cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, info, nil
}

// loadFromFile loads services from a file
func (s *ServiceLoader) loadFromFile(filename string) ([]blockedService, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read service file: %w", err)
	}

	var hlServicesData hlServices
	if unmarshalErr := json.Unmarshal(data, &hlServicesData); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to parse service file: %w", unmarshalErr)
	}

	// Convert hlServicesService to blockedService
	services := convertToBlockedServices(hlServicesData.BlockedServices)
	return services, nil
}

// downloadAndCache downloads service files and caches them
func (s *ServiceLoader) downloadAndCache(ctx context.Context, url, cacheFile string) ([]blockedService, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() {
		err = errors.WithDeferred(err, resp.Body.Close())
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request returned error status: %d", resp.StatusCode)
	}

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read HTTP response: %w", readErr)
	}
	// Save to cache
	if writeErr := os.WriteFile(cacheFile, body, 0o644); writeErr != nil {
		s.logger.ErrorContext(ctx, "failed to write cache", slogutil.KeyError, writeErr, "file", cacheFile)
	}

	var hlServicesData hlServices
	if unmarshalErr := json.Unmarshal(body, &hlServicesData); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to parse service file: %w", unmarshalErr)
	}

	// Convert hlServicesService list to blockedService list
	services := convertToBlockedServices(hlServicesData.BlockedServices)
	return services, nil
}

// convertToBlockedServices converts an hlServicesService list to a blockedService list
func convertToBlockedServices(hlServices []*hlServicesService) []blockedService {
	services := make([]blockedService, 0, len(hlServices))

	for _, service := range hlServices {
		services = append(services, blockedService{
			ID:      service.ID,
			Name:    service.Name,
			IconSVG: []byte(service.IconSVG),
			Rules:   service.Rules,
		})
	}

	return services
}
