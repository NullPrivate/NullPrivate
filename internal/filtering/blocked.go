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

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering/rulelist"
	"github.com/AdguardTeam/AdGuardHome/internal/schedule"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/urlfilter/rules"
)

type blockedCatalogSource string

const (
	blockedCatalogSourceBuiltin blockedCatalogSource = "builtin"
	blockedCatalogSourceDynamic blockedCatalogSource = "dynamic"
)

// blockedServicesCatalog is the single source of truth for the process-wide
// blocked-services catalog state.
type blockedServicesCatalog struct {
	ids    []string
	rules  map[string][]*rules.NetworkRule
	loader *ServiceLoader

	services []blockedService

	source blockedCatalogSource

	// configuredServiceURLs are the URLs associated with loader and the current
	// desired dynamic source.  The active catalog only matches them when source
	// is dynamic and activeServiceURLs are equal to the current config.
	configuredServiceURLs ServicesURLs
	activeServiceURLs     ServicesURLs
}

var blockedCatalogMu sync.RWMutex

var blockedCatalog blockedServicesCatalog

func cloneBlockedServices(services []blockedService) (cloned []blockedService) {
	if len(services) == 0 {
		return nil
	}

	cloned = make([]blockedService, len(services))
	copy(cloned, services)

	return cloned
}

func compileBlockedServices(
	services []blockedService,
) (ids []string, serviceRules map[string][]*rules.NetworkRule) {
	l := len(services)
	ids = make([]string, l)
	serviceRules = make(map[string][]*rules.NetworkRule, l)
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

		ids[i] = s.ID
		serviceRules[s.ID] = netRules
	}

	slices.Sort(ids)

	return ids, serviceRules
}

func setBuiltinCatalogLocked(clearDynamic bool) {
	ids, serviceRules := compileBlockedServices(blockedServices)

	blockedCatalog.ids = ids
	blockedCatalog.rules = serviceRules
	blockedCatalog.services = cloneBlockedServices(blockedServices)
	blockedCatalog.source = blockedCatalogSourceBuiltin
	blockedCatalog.activeServiceURLs = nil
	if clearDynamic {
		blockedCatalog.loader = nil
		blockedCatalog.configuredServiceURLs = nil
	}
}

func activateBuiltinCatalog(clearDynamic bool) {
	blockedCatalogMu.Lock()
	defer blockedCatalogMu.Unlock()

	setBuiltinCatalogLocked(clearDynamic)
}

func rememberBlockedServicesLoader(urls ServicesURLs, loader *ServiceLoader) {
	blockedCatalogMu.Lock()
	defer blockedCatalogMu.Unlock()

	blockedCatalog.loader = loader
	blockedCatalog.configuredServiceURLs = slices.Clone(urls)
}

func activateDynamicCatalog(
	urls ServicesURLs,
	loader *ServiceLoader,
	services []blockedService,
) {
	ids, serviceRules := compileBlockedServices(services)

	blockedCatalogMu.Lock()
	defer blockedCatalogMu.Unlock()

	blockedCatalog.ids = ids
	blockedCatalog.rules = serviceRules
	blockedCatalog.services = cloneBlockedServices(services)
	blockedCatalog.source = blockedCatalogSourceDynamic
	blockedCatalog.loader = loader
	blockedCatalog.configuredServiceURLs = slices.Clone(urls)
	blockedCatalog.activeServiceURLs = slices.Clone(urls)
}

func blockedServicesCatalogSnapshot() (catalog blockedServicesCatalog) {
	ensureBlockedServicesInitialized()

	blockedCatalogMu.RLock()
	defer blockedCatalogMu.RUnlock()

	catalog = blockedCatalog
	catalog.ids = slices.Clone(blockedCatalog.ids)
	catalog.services = cloneBlockedServices(blockedCatalog.services)
	catalog.configuredServiceURLs = slices.Clone(blockedCatalog.configuredServiceURLs)
	catalog.activeServiceURLs = slices.Clone(blockedCatalog.activeServiceURLs)

	return catalog
}

func currentBlockedServicesLoader(urls ServicesURLs) (loader *ServiceLoader) {
	blockedCatalogMu.RLock()
	defer blockedCatalogMu.RUnlock()

	if blockedCatalog.loader == nil || !slices.Equal(blockedCatalog.configuredServiceURLs, urls) {
		return nil
	}

	return blockedCatalog.loader
}

func isBlockedServicesCatalogReady(urls ServicesURLs) (ok bool) {
	if len(urls) == 0 {
		return true
	}

	blockedCatalogMu.RLock()
	defer blockedCatalogMu.RUnlock()

	return blockedCatalog.source == blockedCatalogSourceDynamic &&
		slices.Equal(blockedCatalog.activeServiceURLs, urls)
}

// IsBlockedServicesCatalogReady reports whether the current process-wide
// blocked-services catalog is already resolved for urls.
func IsBlockedServicesCatalogReady(urls ServicesURLs) (ok bool) {
	return isBlockedServicesCatalogReady(urls)
}

// BlockedServicesCatalogReady reports whether the catalog for the current
// filter configuration is already active without triggering any warmup.
func (d *DNSFilter) BlockedServicesCatalogReady() (ok bool) {
	urls := d.cloneConfiguredServiceURLs()
	if len(urls) == 0 {
		return true
	}

	return isBlockedServicesCatalogReady(urls)
}

// ensureBlockedServicesInitialized makes sure blocked-service rules are always
// backed by at least the built-in catalog.
func ensureBlockedServicesInitialized() {
	blockedCatalogMu.Lock()
	defer blockedCatalogMu.Unlock()

	if blockedCatalog.rules != nil {
		return
	}

	setBuiltinCatalogLocked(false)
}

// initBlockedServices initializes package-level blocked service data.
func initBlockedServices() {
	activateBuiltinCatalog(false)
	log.Debug("filtering: initialized %d services", len(blockedServices))
}

// getOrInitServiceLoader returns the current service loader, initializing it
// from the DNS filter configuration if needed.
func (d *DNSFilter) getOrInitServiceLoader() (loader *ServiceLoader) {
	urls := slices.Clone(d.conf.ServiceURLs)
	loader = currentBlockedServicesLoader(urls)
	if loader != nil {
		return loader
	}

	loader = d.initServiceLoader()

	return loader
}

// initServiceLoader initializes the process-wide loader for the current
// service_urls configuration without performing network I/O.
func (d *DNSFilter) initServiceLoader() (loader *ServiceLoader) {
	urls := slices.Clone(d.conf.ServiceURLs)
	if len(urls) == 0 {
		return nil
	}

	logger := slog.Default()
	if d.logger != nil {
		logger = d.logger
	}

	loader = NewServiceLoader(
		urls,
		d.conf.DataDir,
		d.conf.HTTPClient,
		logger,
	)

	rememberBlockedServicesLoader(urls, loader)

	return loader
}

func (d *DNSFilter) cloneConfiguredServiceURLs() (urls ServicesURLs) {
	d.confMu.RLock()
	defer d.confMu.RUnlock()

	return slices.Clone(d.conf.ServiceURLs)
}

func (d *DNSFilter) normalizeBlockedServicesAfterCatalogChange() (changed bool) {
	d.confMu.Lock()
	if d.conf.BlockedServices != nil && len(d.conf.BlockedServices.IDs) > 0 {
		kept, dropped := SanitizeBlockedServiceIDs(d.conf.BlockedServices.IDs)
		if len(dropped) > 0 {
			log.Error("filtering: removed unknown blocked-service ids after catalog change: %v", dropped)
			d.conf.BlockedServices.IDs = kept
			changed = true
		}
	}
	d.confMu.Unlock()

	if d.conf.NormalizeBlockedServices != nil {
		changed = d.conf.NormalizeBlockedServices() || changed
	}

	return changed
}

// EnsureBlockedServicesCatalog makes a best effort to activate the dynamic
// blocked-services catalog for the current service_urls configuration.
func (d *DNSFilter) EnsureBlockedServicesCatalog() (ready bool) {
	urls := d.cloneConfiguredServiceURLs()
	if len(urls) == 0 {
		ensureBlockedServicesInitialized()

		return true
	}

	if isBlockedServicesCatalogReady(urls) {
		return true
	}

	loader := d.getOrInitServiceLoader()
	if loader == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	services, err := loadBlockedServicesFromLoader(ctx, loader)
	if err != nil {
		log.Error("filtering: blocked-services warmup failed: %s", err)

		return false
	}

	currentURLs := d.cloneConfiguredServiceURLs()
	if !slices.Equal(currentURLs, urls) {
		return isBlockedServicesCatalogReady(currentURLs)
	}

	activateDynamicCatalog(urls, loader, services)
	if d.normalizeBlockedServicesAfterCatalogChange() && d.conf.ConfigModified != nil {
		d.conf.ConfigModified()
	}

	return true
}

// filterKnownServiceIDs 过滤出当前已知（可用）的服务 ID。
// 返回值：第一个为保留的已知 ID，第二个为被丢弃的未知 ID。
func filterKnownServiceIDs(list []string) (kept, dropped []string) {
	if len(list) == 0 {
		return nil, nil
	}

	blockedCatalogMu.RLock()
	defer blockedCatalogMu.RUnlock()

	kept = make([]string, 0, len(list))
	dropped = make([]string, 0)
	for _, id := range list {
		if _, ok := blockedCatalog.rules[id]; ok {
			kept = append(kept, id)
		} else {
			dropped = append(dropped, id)
		}
	}
	return kept, dropped
}

// SanitizeBlockedServiceIDs 导出方法：用于在启动或加载配置时，
// 将未知的服务 ID 从列表中剔除，避免因校验失败阻止系统启动。
// 注意：该函数不会返回错误，仅返回保留与丢弃的列表。
func SanitizeBlockedServiceIDs(list []string) (kept, dropped []string) {
	return filterKnownServiceIDs(list)
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
	blockedCatalogMu.RLock()
	defer blockedCatalogMu.RUnlock()

	for _, id := range s.IDs {
		_, ok := blockedCatalog.rules[id]
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

	// TODO(s.chzhen):  Use startTime from [dnsforward.dnsContext].
	if !bsvc.Schedule.Contains(time.Now()) {
		d.ApplyBlockedServicesList(setts, bsvc.IDs)
	}
}

// ApplyBlockedServicesList appends filtering rules to the settings.
func (d *DNSFilter) ApplyBlockedServicesList(setts *Settings, list []string) {
	blockedCatalogMu.RLock()
	defer blockedCatalogMu.RUnlock()

	for _, name := range list {
		rules, ok := blockedCatalog.rules[name]
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
	_ = d.EnsureBlockedServicesCatalog()

	ids := blockedServicesCatalogSnapshot().ids
	aghhttp.WriteJSONResponseOK(w, r, ids)
}

func (d *DNSFilter) handleBlockedServicesAll(w http.ResponseWriter, r *http.Request) {
	_ = d.EnsureBlockedServicesCatalog()

	aghhttp.WriteJSONResponseOK(w, r, struct {
		BlockedServices []blockedService `json:"blocked_services"`
	}{
		BlockedServices: blockedServicesCatalogSnapshot().services,
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

	ids := list
	if d.EnsureBlockedServicesCatalog() {
		// 规范化为 id 并丢弃未知项
		var dropped []string
		ids, dropped = SanitizeBlockedServiceIDs(list)
		if len(dropped) > 0 {
			log.Debug("blocked_services.set: dropping unknown ids: %v", dropped)
		}
	}

	func() {
		d.confMu.Lock()
		defer d.confMu.Unlock()
		if d.conf.BlockedServices == nil {
			d.conf.BlockedServices = &BlockedServices{
				Schedule: schedule.EmptyWeekly(),
				IDs:      ids,
			}
		} else {
			d.conf.BlockedServices.IDs = ids
		}
		log.Debug("Updated blocked services list: %d", len(ids))
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
	// 保证 JSON 返回中 `ids` 不为 null
	if bsvc != nil && bsvc.IDs == nil {
		bsvc.IDs = []string{}
	}
	aghhttp.WriteJSONResponseOK(w, r, bsvc)
}

type blockedServicesUpdateRequest struct {
	Schedule *schedule.Weekly `json:"schedule"`
	IDs      []string         `json:"ids"`

	idsSet bool
}

func (req *blockedServicesUpdateRequest) UnmarshalJSON(data []byte) (err error) {
	type blockedServicesUpdateRequestJSON struct {
		Schedule *schedule.Weekly `json:"schedule"`
		IDs      []string         `json:"ids"`
	}

	decoded := &blockedServicesUpdateRequestJSON{}
	err = json.Unmarshal(data, decoded)
	if err != nil {
		return err
	}

	fields := map[string]json.RawMessage{}
	err = json.Unmarshal(data, &fields)
	if err != nil {
		return err
	}

	req.Schedule = decoded.Schedule
	req.IDs = decoded.IDs
	_, req.idsSet = fields["ids"]

	return nil
}

func (d *DNSFilter) currentBlockedServiceIDs() (ids []string) {
	d.confMu.RLock()
	defer d.confMu.RUnlock()

	if d.conf.BlockedServices == nil {
		return nil
	}

	return slices.Clone(d.conf.BlockedServices.IDs)
}

// handleBlockedServicesUpdate is the handler for the PUT
// /control/blocked_services/update HTTP API.
func (d *DNSFilter) handleBlockedServicesUpdate(w http.ResponseWriter, r *http.Request) {
	req := &blockedServicesUpdateRequest{}
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "json.Decode: %s", err)
		return
	}

	ids := req.IDs
	if !req.idsSet {
		ids = d.currentBlockedServiceIDs()
	}

	bsvc := &BlockedServices{
		Schedule: req.Schedule,
		IDs:      ids,
	}

	if d.EnsureBlockedServicesCatalog() && req.idsSet {
		// 规范化并过滤请求中的服务：仅按 ID 丢弃未知，避免 422。
		kept, dropped := SanitizeBlockedServiceIDs(bsvc.IDs)
		if len(dropped) > 0 {
			log.Debug("blocked_services.update: dropping unknown ids: %v", dropped)
		}
		bsvc.IDs = kept

		err = bsvc.Validate()
		if err != nil {
			aghhttp.Error(r, w, http.StatusUnprocessableEntity, "validating: %s", err)
			return
		}
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
	urls := d.cloneConfiguredServiceURLs()

	// 若未配置服务配置源：不视为错误，回退并保留/刷新内置服务
	if len(urls) == 0 {
		activateBuiltinCatalog(true)
		if d.normalizeBlockedServicesAfterCatalogChange() && d.conf.ConfigModified != nil {
			d.conf.ConfigModified()
		}

		aghhttp.WriteJSONResponseOK(w, r, struct {
			Status  string `json:"status"`
			Count   int    `json:"count"`
			Message string `json:"message"`
		}{
			Status:  "ok",
			Count:   len(blockedServicesCatalogSnapshot().ids),
			Message: "未配置服务源，已使用内置服务",
		})
		return
	}

	loader := currentBlockedServicesLoader(urls)
	if loader == nil {
		loader = d.initServiceLoader()
	}
	if loader == nil {
		aghhttp.Error(r, w, http.StatusBadGateway, "reloading blocked services catalog: no service loader")
		return
	}

	// 使用后台超时上下文，避免请求被客户端中断导致 context canceled
	reloadCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	services, err := loader.ReloadServices(reloadCtx)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadGateway, "reloading blocked services catalog: %s", err)
		return
	}

	activateDynamicCatalog(urls, loader, services)
	if d.normalizeBlockedServicesAfterCatalogChange() && d.conf.ConfigModified != nil {
		d.conf.ConfigModified()
	}

	aghhttp.WriteJSONResponseOK(w, r, struct {
		Status  string `json:"status"`
		Count   int    `json:"count"`
		Message string `json:"message"`
	}{
		Status:  "ok",
		Count:   len(blockedServicesCatalogSnapshot().ids),
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

	urls := ServicesURLs(data.ServiceURLs)

	if len(urls) == 0 {
		func() {
			d.confMu.Lock()
			defer d.confMu.Unlock()
			d.conf.ServiceURLs = nil
		}()

		activateBuiltinCatalog(true)
		d.normalizeBlockedServicesAfterCatalogChange()
		log.Debug("Updated service URLs: 0")
		if d.conf.ConfigModified != nil {
			d.conf.ConfigModified()
		}

		aghhttp.WriteJSONResponseOK(w, r, struct {
			Status  string   `json:"status"`
			URLs    []string `json:"urls"`
			Message string   `json:"message"`
		}{
			Status:  "ok",
			URLs:    []string{},
			Message: "Service URLs updated",
		})
		return
	}

	logger := slog.Default()
	if d.logger != nil {
		logger = d.logger
	}

	loader := NewServiceLoader(urls, d.conf.DataDir, d.conf.HTTPClient, logger)
	reloadCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	services, loadErr := loader.ReloadServices(reloadCtx)
	if loadErr != nil {
		aghhttp.Error(r, w, http.StatusBadGateway, "setting service_urls: %s", loadErr)
		return
	}

	func() {
		d.confMu.Lock()
		defer d.confMu.Unlock()
		d.conf.ServiceURLs = slices.Clone(urls)
	}()

	activateDynamicCatalog(urls, loader, services)
	d.normalizeBlockedServicesAfterCatalogChange()

	log.Debug("Updated service URLs: %d", len(data.ServiceURLs))
	if d.conf.ConfigModified != nil {
		d.conf.ConfigModified()
	}

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
