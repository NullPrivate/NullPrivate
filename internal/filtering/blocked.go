package filtering

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/jqknono/AdGuardHome/internal/aghhttp"
	"github.com/jqknono/AdGuardHome/internal/filtering/rulelist"
	"github.com/jqknono/AdGuardHome/internal/schedule"
)

// 首先，创建一个互斥锁保护 serviceLoader
var serviceLoaderMu sync.RWMutex

// serviceRules maps a service ID to its filtering rules.
var serviceRules map[string][]*rules.NetworkRule

// serviceIDs contains service IDs sorted alphabetically.
var serviceIDs []string

// serviceLoader 保存动态服务加载器实例
var serviceLoader *ServiceLoader

// initBlockedServices initializes package-level blocked service data.
func initBlockedServices() {
	l := len(blockedServices)
	serviceIDs = make([]string, l)
	serviceRules = make(map[string][]*rules.NetworkRule, l)
	for i, s := range blockedServices {
		netRules := make([]*rules.NetworkRule, 0, len(s.Rules))
		for _, text := range s.Rules {
			rule, err := rules.NewNetworkRule(text, rulelist.URLFilterIDBlockedService)
			if err != nil {
				log.Error("parsing blocked service %q rule %q: %s", s.ID, text, err)
				continue
			}
			netRules = append(netRules, rule)
		}
		serviceIDs[i] = s.ID
		serviceRules[s.ID] = netRules
	}
	slices.Sort(serviceIDs)
	log.Debug("filtering: initialized %d services", l)
}

// initServiceLoader 初始化服务加载器
func (d *DNSFilter) initServiceLoader(ctx context.Context) {
	if len(d.conf.ServiceURLs) == 0 {
		// use default "https://adguardteam.github.io/HostlistsRegistry/assets/services.json"
		d.conf.ServiceURLs = []string{"https://adguardteam.github.io/HostlistsRegistry/assets/services.json"}
	}

	logger := slog.Default()
	if d.logger != nil {
		logger = d.logger
	}

	newLoader := NewServiceLoader(
		d.conf.ServiceURLs,
		d.conf.DataDir,
		d.conf.HTTPClient,
		logger,
	)

	// 使用锁保护写操作
	serviceLoaderMu.Lock()
	serviceLoader = newLoader
	serviceLoaderMu.Unlock()

	// 预加载服务
	go func() {
		// 在goroutine中使用读锁访问 serviceLoader
		serviceLoaderMu.RLock()
		loader := serviceLoader
		serviceLoaderMu.RUnlock()

		if loader != nil {
			_, err := loader.LoadServices(ctx)
			if err != nil {
				log.Error("filtering: failed to load services: %s", err)
			}
		}
	}()
}

// updateBlockedServicesFromLoader 从加载器中更新服务规则
func updateBlockedServicesFromLoader(ctx context.Context) {
	serviceLoaderMu.RLock()
	loader := serviceLoader
	serviceLoaderMu.RUnlock()

	if loader == nil {
		return
	}
	services := loader.GetBlockedServices(ctx)
	if len(services) == 0 {
		log.Debug("filtering: no services loaded from URLs")
		return
	}

	// 更新服务规则
	newServiceIDs := make([]string, len(services))
	newServiceRules := make(map[string][]*rules.NetworkRule, len(services))

	for i, s := range services {
		netRules := make([]*rules.NetworkRule, 0, len(s.Rules))
		for _, text := range s.Rules {
			rule, err := rules.NewNetworkRule(text, rulelist.URLFilterIDBlockedService)
			if err != nil {
				log.Error("parsing blocked service %q rule %q: %s", s.ID, text, err)
				continue
			}
			netRules = append(netRules, rule)
		}
		newServiceIDs[i] = s.ID
		newServiceRules[s.ID] = netRules
	}

	slices.Sort(newServiceIDs)

	// 使用读写锁保护全局变量
	serviceIDs = newServiceIDs
	serviceRules = newServiceRules

	log.Debug("filtering: updated %d services from dynamic sources", len(services))
}

// BlockedServices is the configuration of blocked services.
type BlockedServices struct {
	// Schedule is blocked services schedule for every day of the week.
	Schedule *schedule.Weekly `json:"schedule" yaml:"schedule"`
	// IDs is the names of blocked services.
	IDs []string `json:"ids" yaml:"ids"`
}

// Clone returns a deep copy of blocked services.
func (s *BlockedServices) Clone() (c *BlockedServices) {
	if s == nil {
		return nil
	}
	return &BlockedServices{
		Schedule: s.Schedule.Clone(),
		IDs:      slices.Clone(s.IDs),
	}
}

// Validate returns an error if blocked services contain unknown service ID.  s
// must not be nil.
func (s *BlockedServices) Validate() (err error) {
	for _, id := range s.IDs {
		_, ok := serviceRules[id]
		if !ok {
			return fmt.Errorf("unknown blocked-service %q", id)
		}
	}
	return nil
}

// ApplyBlockedServices - set blocked services settings for this DNS request
func (d *DNSFilter) ApplyBlockedServices(setts *Settings) {
	d.confMu.RLock()
	defer d.confMu.RUnlock()

	setts.ServicesRules = []ServiceEntry{}
	bsvc := d.conf.BlockedServices

	// 在应用服务前动态更新服务规则
	updateBlockedServicesFromLoader(context.Background())

	// TODO(s.chzhen):  Use startTime from [dnsforward.dnsContext].
	if bsvc != nil && !bsvc.Schedule.Contains(time.Now()) {
		d.ApplyBlockedServicesList(setts, bsvc.IDs)
	}
}

// ApplyBlockedServicesList appends filtering rules to the settings.
func (d *DNSFilter) ApplyBlockedServicesList(setts *Settings, list []string) {
	for _, name := range list {
		rules, ok := serviceRules[name]
		if !ok {
			log.Error("unknown service name: %s", name)
			continue
		}
		setts.ServicesRules = append(setts.ServicesRules, ServiceEntry{
			Name:  name,
			Rules: rules,
		})
	}
}

func (d *DNSFilter) handleBlockedServicesIDs(w http.ResponseWriter, r *http.Request) {
	// 在获取规则列表前动态更新服务规则
	updateBlockedServicesFromLoader(r.Context())

	aghhttp.WriteJSONResponseOK(w, r, serviceIDs)
}

func (d *DNSFilter) handleBlockedServicesAll(w http.ResponseWriter, r *http.Request) {
	// 在获取规则列表前动态更新服务规则
	updateBlockedServicesFromLoader(r.Context())

	allServices := blockedServices
	if serviceLoader != nil {
		allServices = serviceLoader.GetBlockedServices(r.Context())
	}

	aghhttp.WriteJSONResponseOK(w, r, struct {
		BlockedServices []blockedService `json:"blocked_services"`
	}{
		BlockedServices: allServices,
	})
}

// handleBlockedServicesList is the handler for the GET
// /control/blocked_services/list HTTP API.
//
// Deprecated:  Use handleBlockedServicesGet.
func (d *DNSFilter) handleBlockedServicesList(w http.ResponseWriter, r *http.Request) {
	var list []string
	func() {
		d.confMu.Lock()
		defer d.confMu.Unlock()
		if d.conf.BlockedServices != nil {
			list = d.conf.BlockedServices.IDs
		} else {
			list = []string{}
		}
	}()
	aghhttp.WriteJSONResponseOK(w, r, list)
}

// handleBlockedServicesSet is the handler for the POST
// /control/blocked_services/set HTTP API.
//
// Deprecated:  Use handleBlockedServicesUpdate.
func (d *DNSFilter) handleBlockedServicesSet(w http.ResponseWriter, r *http.Request) {
	list := []string{}
	err := json.NewDecoder(r.Body).Decode(&list)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "json.Decode: %s", err)
		return
	}

	// 在设置规则前动态更新服务规则
	updateBlockedServicesFromLoader(r.Context())

	func() {
		d.confMu.Lock()
		defer d.confMu.Unlock()
		if d.conf.BlockedServices == nil {
			d.conf.BlockedServices = &BlockedServices{
				Schedule: schedule.EmptyWeekly(),
				IDs:      list,
			}
		} else {
			d.conf.BlockedServices.IDs = list
		}
		log.Debug("Updated blocked services list: %d", len(list))
	}()
	d.conf.ConfigModified()
}

// handleBlockedServicesGet is the handler for the GET
// /control/blocked_services/get HTTP API.
func (d *DNSFilter) handleBlockedServicesGet(w http.ResponseWriter, r *http.Request) {
	var bsvc *BlockedServices
	func() {
		d.confMu.RLock()
		defer d.confMu.RUnlock()
		if d.conf.BlockedServices != nil {
			bsvc = d.conf.BlockedServices.Clone()
		} else {
			bsvc = &BlockedServices{
				Schedule: schedule.EmptyWeekly(),
				IDs:      []string{},
			}
		}
	}()
	aghhttp.WriteJSONResponseOK(w, r, bsvc)
}

// handleBlockedServicesUpdate is the handler for the PUT
// /control/blocked_services/update HTTP API.
func (d *DNSFilter) handleBlockedServicesUpdate(w http.ResponseWriter, r *http.Request) {
	bsvc := &BlockedServices{}
	err := json.NewDecoder(r.Body).Decode(bsvc)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "json.Decode: %s", err)
		return
	}

	// 在更新规则前动态更新服务规则
	updateBlockedServicesFromLoader(r.Context())

	err = bsvc.Validate()
	if err != nil {
		aghhttp.Error(r, w, http.StatusUnprocessableEntity, "validating: %s", err)
		return
	}
	if bsvc.Schedule == nil {
		bsvc.Schedule = schedule.EmptyWeekly()
	}
	func() {
		d.confMu.Lock()
		defer d.confMu.Unlock()
		d.conf.BlockedServices = bsvc
	}()
	log.Debug("updated blocked services schedule: %d", len(bsvc.IDs))
	d.conf.ConfigModified()
}

// handleBlockedServicesReload is the handler for the POST
// /control/blocked_services/reload HTTP API
func (d *DNSFilter) handleBlockedServicesReload(w http.ResponseWriter, r *http.Request) {
	if serviceLoader == nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "service loader not initialized")
		return
	}

	_, err := serviceLoader.LoadServices(r.Context())
	if err != nil {
		aghhttp.Error(r, w, http.StatusInternalServerError, "failed to reload services: %s", err)
		return
	}

	updateBlockedServicesFromLoader(r.Context())

	aghhttp.WriteJSONResponseOK(w, r, struct {
		Status  string `json:"status"`
		Count   int    `json:"count"`
		Message string `json:"message"`
	}{
		Status:  "ok",
		Count:   len(serviceIDs),
		Message: "服务已重新加载",
	})
}

// handleServiceURLsGet 获取当前配置的 service_urls
func (d *DNSFilter) handleServiceURLsGet(w http.ResponseWriter, r *http.Request) {
	var urls []string
	func() {
		d.confMu.RLock()
		defer d.confMu.RUnlock()
		if d.conf.ServiceURLs != nil {
			urls = slices.Clone(d.conf.ServiceURLs)
		} else {
			urls = []string{}
		}
	}()
	aghhttp.WriteJSONResponseOK(w, r, struct {
		ServiceURLs []string `json:"service_urls"`
	}{
		ServiceURLs: urls,
	})
}

// handleServiceURLsSet 设置 service_urls
func (d *DNSFilter) handleServiceURLsSet(w http.ResponseWriter, r *http.Request) {
	var data struct {
		ServiceURLs []string `json:"service_urls"`
	}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "json.Decode: %s", err)
		return
	}

	if len(data.ServiceURLs) == 0 {
		// Use default value
		data.ServiceURLs = []string{"https://adguardteam.github.io/HostlistsRegistry/assets/services.json"}
	}

	func() {
		d.confMu.Lock()
		defer d.confMu.Unlock()
		d.conf.ServiceURLs = data.ServiceURLs
	}()

	// Reinitialize service loader
	d.initServiceLoader(r.Context())

	// Reload services
	if serviceLoader != nil {
		_, loadErr := serviceLoader.LoadServices(r.Context())
		if loadErr != nil {
			log.Error("failed to reload services: %s", loadErr)
		}
		updateBlockedServicesFromLoader(r.Context())
	}

	log.Debug("Updated service URLs: %d", len(data.ServiceURLs))
	d.conf.ConfigModified()

	aghhttp.WriteJSONResponseOK(w, r, struct {
		Status  string   `json:"status"`
		URLs    []string `json:"urls"`
		Message string   `json:"message"`
	}{
		Status:  "ok",
		URLs:    data.ServiceURLs,
		Message: "Service URLs updated",
	})
}
