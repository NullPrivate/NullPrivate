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
	"strings"
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

// 当服务缺少 IconSVG 字段时使用的默认 SVG（来源: SVG Repo）。
const defaultIconSVG = `<svg fill="#000000" width="800px" height="800px" viewBox="0 -8 72 72" id="Layer_1" data-name="Layer 1" xmlns="http://www.w3.org/2000/svg"><title>check</title><path d="M61.07,12.9,57,8.84a2.93,2.93,0,0,0-4.21,0L28.91,32.73,19.2,23A3,3,0,0,0,15,23l-4.06,4.07a2.93,2.93,0,0,0,0,4.21L26.81,47.16a2.84,2.84,0,0,0,2.1.89A2.87,2.87,0,0,0,31,47.16l30.05-30a2.93,2.93,0,0,0,0-4.21Z"/></svg>`

// NewServiceLoader creates a new service loader
func NewServiceLoader(urls ServicesURLs, dataDir string, client *http.Client, logger *slog.Logger) *ServiceLoader {
	// 确保 http.Client 非空，避免早期预加载时触发空指针
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

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

	allServices, aggErr := s.loadFromAllURLs(ctx)

	if len(allServices) > 0 {
		s.services = allServices
		s.lastRefresh = time.Now()
		return s.services, nil
	}

	// 如果没有任何来源成功，且此前已有缓存内存副本，则返回缓存副本；否则返回错误
	if s.services != nil {
		return s.services, nil
	}

	if aggErr == nil {
		aggErr = fmt.Errorf("no services loaded from configured URLs")
	}

	return nil, aggErr
}

// ReloadServices forces a refresh from the configured remote URLs.  It keeps
// the last successful in-memory result untouched on failure.
func (s *ServiceLoader) ReloadServices(ctx context.Context) ([]blockedService, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureServiceDir(); err != nil {
		return nil, fmt.Errorf("failed to create service cache directory: %w", err)
	}

	allServices, aggErr := s.reloadFromAllURLs(ctx)
	if len(allServices) > 0 {
		s.services = allServices
		s.lastRefresh = time.Now()

		return s.services, nil
	}

	if aggErr == nil {
		aggErr = fmt.Errorf("no services loaded from configured URLs")
	}

	return nil, aggErr
}

// loadFromAllURLs iterates over configured URLs, loads services and aggregates errors.
func (s *ServiceLoader) loadFromAllURLs(ctx context.Context) ([]blockedService, error) {
	var allServices []blockedService
	var aggErr error
	for _, url := range s.urls {
		services, err := s.loadFromURL(ctx, url)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				s.logger.DebugContext(ctx, "load services canceled", slogutil.KeyError, err, "url", url)
			} else {
				s.logger.ErrorContext(ctx, "failed to load services from URL", slogutil.KeyError, err, "url", url)
			}
			if aggErr == nil {
				aggErr = err
			} else {
				aggErr = fmt.Errorf("%w; %v", aggErr, err)
			}
			continue
		}
		allServices = append(allServices, services...)
	}
	return allServices, aggErr
}

// reloadFromAllURLs forces downloading from all URLs and aggregates errors.
func (s *ServiceLoader) reloadFromAllURLs(ctx context.Context) ([]blockedService, error) {
	var allServices []blockedService
	var aggErr error
	for _, url := range s.urls {
		cacheFile := s.cacheFileName(url)
		services, err := s.downloadAndCache(ctx, url, cacheFile)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				s.logger.DebugContext(ctx, "reload services canceled", slogutil.KeyError, err, "url", url)
			} else {
				s.logger.ErrorContext(ctx, "failed to reload services from URL", slogutil.KeyError, err, "url", url)
			}
			if aggErr == nil {
				aggErr = err
			} else {
				aggErr = fmt.Errorf("%w; %v", aggErr, err)
			}

			continue
		}

		allServices = append(allServices, services...)
	}

	return allServices, aggErr
}

// GetBlockedServices gets all loaded blocked services
func (s *ServiceLoader) GetBlockedServices(ctx context.Context) []blockedService {
	// Read current services under read lock, then release before potentially
	// acquiring the write lock inside LoadServices to avoid self-deadlock.
	s.mu.RLock()
	services := s.services
	s.mu.RUnlock()

	if services != nil {
		return services
	}

	// If not loaded yet, try to load without holding the read lock.
	services, err := s.LoadServices(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to load services", slogutil.KeyError, err)
		return blockedServices
	}

	if len(services) == 0 {
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

	// 若本地已有缓存且未超过3天，直接使用本地缓存
	if cacheExists && time.Since(cacheInfo.ModTime()) < 3*24*time.Hour {
		return s.loadFromFile(cacheFile)
	}

	// 若本地有缓存但已过期：尝试下载，失败则回退到本地旧缓存
	if cacheExists {
		services, dlErr := s.downloadAndCache(ctx, url, cacheFile)
		if dlErr != nil {
			s.logger.WarnContext(ctx, "download failed; using stale cache", slogutil.KeyError, dlErr, "file", cacheFile)
			return s.loadFromFile(cacheFile)
		}
		return services, nil
	}

	// 本地无缓存：下载；失败则返回错误
	services, dlErr := s.downloadAndCache(ctx, url, cacheFile)
	if dlErr != nil {
		return nil, dlErr
	}
	return services, nil
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
		icon := service.IconSVG
		if strings.TrimSpace(icon) == "" {
			icon = defaultIconSVG
		}
		services = append(services, blockedService{
			ID:      service.ID,
			Name:    service.Name,
			IconSVG: []byte(icon),
			Rules:   service.Rules,
		})
	}

	return services
}

// loadBlockedServicesFromLoader loads blocked services through loader.
func loadBlockedServicesFromLoader(
	ctx context.Context,
	loader *ServiceLoader,
) (services []blockedService, err error) {
	if loader == nil {
		return nil, nil
	}

	services, err = loader.LoadServices(ctx)
	if err != nil {
		return nil, err
	}

	return services, nil
}

// PreloadServiceCatalog 在启动早期预加载服务源，并更新全局的 serviceRules。
// 这样在 clients 初始化阶段调用 SanitizeBlockedServiceIDs 时，能够识别来自
// 动态服务源（service_urls）的 ID，避免被误判为未知而剔除。
//
// 注意：这是一个幂等的便捷函数，允许在 DNS 过滤模块创建之前调用。
func PreloadServiceCatalog(ctx context.Context, conf *Config, logger *slog.Logger) {
	if conf == nil {
		return
	}

	ensureBlockedServicesInitialized()

	slogger := slog.Default()
	if logger != nil {
		slogger = logger
	}

	urls := conf.ServiceURLs
	if len(urls) == 0 {
		activateBuiltinCatalog(true)

		return
	}

	loader := NewServiceLoader(urls, conf.DataDir, conf.HTTPClient, slogger)
	rememberBlockedServicesLoader(urls, loader)

	// 带超时的同步加载，避免阻塞启动过久
	ctx2, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	services, err := loadBlockedServicesFromLoader(ctx2, loader)
	if err != nil {
		slogger.ErrorContext(ctx, "filtering: preload services failed at startup", slogutil.KeyError, err)

		return
	}

	activateDynamicCatalog(urls, loader, services)
}
