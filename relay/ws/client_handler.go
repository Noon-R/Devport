package ws

import (
	"log"
	"net/http"
	"strings"

	"github.com/Noon-R/Devport/relay/config"
	"github.com/Noon-R/Devport/relay/store"
	"github.com/coder/websocket"
	"github.com/google/uuid"
)

// ClientHandler handles WebSocket connections from mobile clients
type ClientHandler struct {
	cfg          *config.Config
	store        *store.Store
	relayHandler *RelayHandler
}

// NewClientHandler creates a new client handler
func NewClientHandler(cfg *config.Config, store *store.Store, relayHandler *RelayHandler) *ClientHandler {
	return &ClientHandler{
		cfg:          cfg,
		store:        store,
		relayHandler: relayHandler,
	}
}

// ServeHTTP handles WebSocket connections at /ws
func (h *ClientHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract subdomain from host
	subdomain := h.extractSubdomain(r.Host)
	if subdomain == "" {
		http.Error(w, "Invalid subdomain", http.StatusBadRequest)
		return
	}

	// Check if relay is connected
	relay := h.store.GetRelayBySubdomain(subdomain)
	if relay == nil || !relay.IsConnected() {
		http.Error(w, "Relay not connected", http.StatusServiceUnavailable)
		return
	}

	// Get multiplexer
	mux := h.relayHandler.GetMultiplexer(subdomain)
	if mux == nil {
		http.Error(w, "Relay not ready", http.StatusServiceUnavailable)
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

	// Generate connection ID
	connID := uuid.New().String()

	// Register client
	h.store.AddClient(relay, connID, conn)
	defer h.store.RemoveClient(relay, connID)

	// Notify relay of new client
	if err := mux.NotifyClientConnected(ctx, connID); err != nil {
		log.Printf("Failed to notify client connected: %v", err)
	}

	log.Printf("Client connected: %s (subdomain: %s)", connID, subdomain)

	// Message loop
	for {
		msgType, data, err := conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) != -1 {
				log.Printf("Client WebSocket closed: %s (status: %v)", connID, websocket.CloseStatus(err))
			} else {
				log.Printf("Client read error: %s - %v", connID, err)
			}
			break
		}

		if msgType != websocket.MessageText {
			continue
		}

		// Forward message to relay
		if err := mux.HandleClientMessage(ctx, connID, data); err != nil {
			log.Printf("Failed to forward client message: %v", err)
		}
	}

	// Notify relay of client disconnection
	if err := mux.NotifyClientDisconnected(ctx, connID); err != nil {
		log.Printf("Failed to notify client disconnected: %v", err)
	}

	log.Printf("Client disconnected: %s (subdomain: %s)", connID, subdomain)
}

// extractSubdomain extracts the subdomain from the host
func (h *ClientHandler) extractSubdomain(host string) string {
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
