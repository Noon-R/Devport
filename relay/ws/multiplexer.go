package ws

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/Noon-R/Devport/relay/store"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// EnvelopeType defines the type of envelope message
type EnvelopeType string

const (
	EnvelopeTypeMessage      EnvelopeType = "message"
	EnvelopeTypeDisconnected EnvelopeType = "disconnected"
	EnvelopeTypeConnected    EnvelopeType = "connected"
	EnvelopeTypeHTTPRequest  EnvelopeType = "http_request"
	EnvelopeTypeHTTPResponse EnvelopeType = "http_response"
)

// Envelope wraps messages for multiplexing
type Envelope struct {
	ConnectionID string          `json:"connection_id"`
	Type         EnvelopeType    `json:"type"`
	Payload      json.RawMessage `json:"payload,omitempty"`
	HTTPRequest  *HTTPRequest    `json:"http_request,omitempty"`
	HTTPResponse *HTTPResponse   `json:"http_response,omitempty"`
}

// HTTPRequest represents an HTTP request forwarded through the relay
type HTTPRequest struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

// HTTPResponse represents an HTTP response forwarded through the relay
type HTTPResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string            `json:"body,omitempty"`
}

// Multiplexer handles message routing between relay and clients
type Multiplexer struct {
	relay *store.RelayConnection
	mu    sync.RWMutex
}

// NewMultiplexer creates a new multiplexer for a relay connection
func NewMultiplexer(relay *store.RelayConnection) *Multiplexer {
	return &Multiplexer{
		relay: relay,
	}
}

// ForwardToRelay sends an envelope to the local PC relay
func (m *Multiplexer) ForwardToRelay(ctx context.Context, envelope *Envelope) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.relay.Conn == nil {
		return nil
	}

	return wsjson.Write(ctx, m.relay.Conn, envelope)
}

// ForwardToClient sends a message to a specific client
func (m *Multiplexer) ForwardToClient(ctx context.Context, connID string, payload json.RawMessage) error {
	clientVal, ok := m.relay.Clients.Load(connID)
	if !ok {
		return nil
	}

	client := clientVal.(*store.ClientConnection)
	if client.Conn == nil {
		return nil
	}

	return client.Conn.Write(ctx, websocket.MessageText, payload)
}

// BroadcastToClients sends a message to all clients
func (m *Multiplexer) BroadcastToClients(ctx context.Context, payload json.RawMessage) {
	m.relay.Clients.Range(func(key, value interface{}) bool {
		client := value.(*store.ClientConnection)
		if client.Conn != nil {
			client.Conn.Write(ctx, websocket.MessageText, payload)
		}
		return true
	})
}

// NotifyClientConnected notifies the relay that a client connected
func (m *Multiplexer) NotifyClientConnected(ctx context.Context, connID string) error {
	envelope := &Envelope{
		ConnectionID: connID,
		Type:         EnvelopeTypeConnected,
	}
	return m.ForwardToRelay(ctx, envelope)
}

// NotifyClientDisconnected notifies the relay that a client disconnected
func (m *Multiplexer) NotifyClientDisconnected(ctx context.Context, connID string) error {
	envelope := &Envelope{
		ConnectionID: connID,
		Type:         EnvelopeTypeDisconnected,
	}
	return m.ForwardToRelay(ctx, envelope)
}

// HandleRelayMessage handles a message from the relay and routes it to clients
func (m *Multiplexer) HandleRelayMessage(ctx context.Context, data []byte) error {
	var envelope Envelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		log.Printf("Failed to unmarshal envelope: %v", err)
		return err
	}

	switch envelope.Type {
	case EnvelopeTypeMessage:
		return m.ForwardToClient(ctx, envelope.ConnectionID, envelope.Payload)

	case EnvelopeTypeHTTPResponse:
		// HTTP responses are handled separately
		return nil

	default:
		log.Printf("Unknown envelope type: %s", envelope.Type)
	}

	return nil
}

// HandleClientMessage handles a message from a client and forwards to relay
func (m *Multiplexer) HandleClientMessage(ctx context.Context, connID string, data []byte) error {
	envelope := &Envelope{
		ConnectionID: connID,
		Type:         EnvelopeTypeMessage,
		Payload:      data,
	}
	return m.ForwardToRelay(ctx, envelope)
}
