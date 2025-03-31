package control

import (
	"encoding/json"
	"net/http"
)

// respondJSON returns data to the client in JSON format
func respondJSON(w http.ResponseWriter, _ *http.Request, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// BlockedServicesController handles API requests related to blocked services
type BlockedServicesController struct {
	// Dependencies can be added here
}

// NewBlockedServicesController creates a new blocked services controller
func NewBlockedServicesController() *BlockedServicesController {
	return &BlockedServicesController{}
}

// Get handles requests to retrieve the list of blocked services
// GET /control/blocked_services/get
func (c *BlockedServicesController) Get(w http.ResponseWriter, r *http.Request) {
	// Implement logic to get the list of blocked services
	// For example:
	data := map[string]interface{}{
		"services": []string{}, // Should return the actual list of blocked services
		"enabled":  false,      // Whether the blocked services feature is enabled
	}

	respondJSON(w, r, data)
}
