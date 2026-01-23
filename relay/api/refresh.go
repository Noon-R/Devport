package api

import (
	"encoding/json"
	"net/http"

	"github.com/Noon-R/Devport/relay/config"
	"github.com/Noon-R/Devport/relay/store"
)

// RefreshHandler handles token refresh
type RefreshHandler struct {
	cfg   *config.Config
	store *store.Store
}

// RefreshRequest is the request body for refresh
type RefreshRequest struct {
	RelayToken string `json:"relay_token"`
}

// RefreshResponse is the response for refresh
type RefreshResponse struct {
	Subdomain   string `json:"subdomain"`
	RelayServer string `json:"relay_server"`
	Status      string `json:"status"`
}

// NewRefreshHandler creates a new refresh handler
func NewRefreshHandler(cfg *config.Config, store *store.Store) *RefreshHandler {
	return &RefreshHandler{
		cfg:   cfg,
		store: store,
	}
}

// ServeHTTP handles POST /api/relay/refresh
func (h *RefreshHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.RelayToken == "" {
		http.Error(w, "relay_token is required", http.StatusBadRequest)
		return
	}

	// Find relay by token
	relay := h.store.GetRelayByToken(req.RelayToken)
	if relay == nil {
		http.Error(w, "Invalid relay token", http.StatusUnauthorized)
		return
	}

	// Build response
	resp := RefreshResponse{
		Subdomain:   relay.Subdomain,
		RelayServer: h.cfg.Domain,
		Status:      "ok",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
