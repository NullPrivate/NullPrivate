package home

import (
	"net/http"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/golibs/log"
)

// serviceTypeJSON is the response format for GET/PUT service-type API.
type serviceTypeJSON struct {
	ServiceType string `json:"service_type"`
}

// handleServiceTypeGet is the handler for the GET /control/service-type HTTP API.
func (web *webAPI) handleServiceTypeGet(w http.ResponseWriter, r *http.Request) {
	resp := serviceTypeJSON{}

	func() {
		config.RLock()
		defer config.RUnlock()

		resp.ServiceType = config.ServiceType
	}()

	log.Debug("service-type: returning service_type=%s", resp.ServiceType)
	aghhttp.WriteJSONResponseOK(w, r, resp)
}
