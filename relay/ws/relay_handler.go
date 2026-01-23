package ws

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/Noon-R/Devport/relay/config"
	"github.com/Noon-R/Devport/relay/store"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// RelayHandler handles WebSocket connections from local PCs
type RelayHandler struct {
	cfg          *config.Config
	store        *store.Store
	multiplexers sync.Map // map[subdomain]*Multiplexer
}

// RelayAuthRequest is the authentication request from local PC
type RelayAuthRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params"`
	ID      interface{}            `json:"id"`
}

// RelayAuthResponse is the authentication response
type RelayAuthResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewRelayHandler creates a new relay handler
func NewRelayHandler(cfg *config.Config, store *store.Store) *RelayHandler {
	return &RelayHandler{
		cfg:   cfg,
		store: store,
	}
}

// GetMultiplexer returns the multiplexer for a subdomain
func (h *RelayHandler) GetMultiplexer(subdomain string) *Multiplexer {
	if v, ok := h.multiplexers.Load(subdomain); ok {
		return v.(*Multiplexer)
	}
	return nil
}

// ServeHTTP handles WebSocket connections at /relay
func (h *RelayHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract subdomain from host
	subdomain := h.extractSubdomain(r.Host)
	if subdomain == "" {
		http.Error(w, "Invalid subdomain", http.StatusBadRequest)
		return
	}

	// Accept WebSocket connection
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		log.Printf("WebSocket accept error: %v", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "connection closed")

	ctx := r.Context()

	// Wait for authentication
	if !h.authenticate(ctx, conn, subdomain) {
		return
	}

	// Get relay and set connection
	relay := h.store.GetRelayBySubdomain(subdomain)
	if relay == nil {
		log.Printf("Relay not found for subdomain: %s", subdomain)
		return
	}

	h.store.SetRelayConn(subdomain, conn)

	// Create multiplexer
	mux := NewMultiplexer(relay)
	h.multiplexers.Store(subdomain, mux)
	defer h.multiplexers.Delete(subdomain)

	log.Printf("Relay connected: %s", subdomain)

	// Message loop
	for {
		msgType, data, err := conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) != -1 {
				log.Printf("Relay WebSocket closed: %s (status: %v)", subdomain, websocket.CloseStatus(err))
			} else {
				log.Printf("Relay read error: %s - %v", subdomain, err)
			}
			break
		}

		if msgType != websocket.MessageText {
			continue
		}

		// Handle envelope from relay
		if err := mux.HandleRelayMessage(ctx, data); err != nil {
			log.Printf("Failed to handle relay message: %v", err)
		}
	}

	// Cleanup
	h.store.SetRelayConn(subdomain, nil)
	log.Printf("Relay disconnected: %s", subdomain)
}

// authenticate handles the authentication handshake
func (h *RelayHandler) authenticate(ctx context.Context, conn *websocket.Conn, subdomain string) bool {
	var req RelayAuthRequest
	if err := wsjson.Read(ctx, conn, &req); err != nil {
		log.Printf("Failed to read auth request: %v", err)
		return false
	}

	if req.Method != "register" {
		h.sendError(ctx, conn, req.ID, -32600, "Expected 'register' method")
		return false
	}

	token, ok := req.Params["relay_token"].(string)
	if !ok || token == "" {
		h.sendError(ctx, conn, req.ID, -32602, "relay_token is required")
		return false
	}

	// Verify token
	relay := h.store.GetRelayByToken(token)
	if relay == nil || relay.Subdomain != subdomain {
		h.sendError(ctx, conn, req.ID, -32001, "Invalid relay token")
		return false
	}

	// Send success response
	resp := RelayAuthResponse{
		JSONRPC: "2.0",
		Result:  map[string]string{"status": "ok"},
		ID:      req.ID,
	}
	if err := wsjson.Write(ctx, conn, resp); err != nil {
		log.Printf("Failed to send auth response: %v", err)
		return false
	}

	return true
}

// sendError sends an error response
func (h *RelayHandler) sendError(ctx context.Context, conn *websocket.Conn, id interface{}, code int, message string) {
	resp := RelayAuthResponse{
		JSONRPC: "2.0",
		Error:   &RPCError{Code: code, Message: message},
		ID:      id,
	}
	wsjson.Write(ctx, conn, resp)
}

// extractSubdomain extracts the subdomain from the host
func (h *RelayHandler) extractSubdomain(host string) string {
	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Expected format: {subdomain}.{domain}
	domain := h.cfg.Domain
	if strings.HasSuffix(host, "."+domain) {
		return strings.TrimSuffix(host, "."+domain)
	}

	// For local development, use query param or first segment
	parts := strings.Split(host, ".")
	if len(parts) > 1 {
		return parts[0]
	}

	return ""
}

// SendToRelay sends data to the relay (used for HTTP proxying)
func (h *RelayHandler) SendToRelay(subdomain string, envelope *Envelope) error {
	mux := h.GetMultiplexer(subdomain)
	if mux == nil {
		return nil
	}
	
	data, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	
	relay := h.store.GetRelayBySubdomain(subdomain)
	if relay == nil || relay.Conn == nil {
		return nil
	}
	
	return relay.Conn.Write(context.Background(), websocket.MessageText, data)
}
