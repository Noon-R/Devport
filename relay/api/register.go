package api

import (
	"encoding/json"
	"net/http"

	"github.com/Noon-R/Devport/relay/config"
	"github.com/Noon-R/Devport/relay/store"
)

// RegisterHandler handles relay registration
type RegisterHandler struct {
	cfg   *config.Config
	store *store.Store
}

// RegisterRequest is the request body for registration
type RegisterRequest struct {
	ClientVersion string `json:"client_version"`
}

// RegisterResponse is the response for registration
type RegisterResponse struct {
	Subdomain   string `json:"subdomain"`
	RelayToken  string `json:"relay_token"`
	RelayServer string `json:"relay_server"`
}

// NewRegisterHandler creates a new register handler
func NewRegisterHandler(cfg *config.Config, store *store.Store) *RegisterHandler {
	return &RegisterHandler{
		cfg:   cfg,
		store: store,
	}
}

// ServeHTTP handles POST /api/relay/register
func (h *RegisterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body for simple registration
		req = RegisterRequest{}
	}

	// Generate unique subdomain and token
	subdomain := h.store.GenerateSubdomain()
	token := store.GenerateToken()

	// Register the relay
	h.store.RegisterRelay(subdomain, token)

	// Build response
	resp := RegisterResponse{
		Subdomain:   subdomain,
		RelayToken:  token,
		RelayServer: h.cfg.Domain,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
