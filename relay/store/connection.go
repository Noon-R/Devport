package store

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// RelayConnection represents a connection from a local PC
type RelayConnection struct {
	Subdomain   string
	RelayToken  string
	Conn        *websocket.Conn
	ConnectedAt time.Time
	LastPing    time.Time
	Clients     sync.Map // map[connectionID]*ClientConnection
	mu          sync.RWMutex
}

// ClientConnection represents a connection from a mobile client
type ClientConnection struct {
	ID         string
	Conn       *websocket.Conn
	RelayConn  *RelayConnection
	ConnectedAt time.Time
}

// Store manages all connections
type Store struct {
	relays    sync.Map // map[subdomain]*RelayConnection
	tokens    sync.Map // map[token]*RelayConnection
	subdomains sync.Map // map[subdomain]bool (used subdomains)
	mu        sync.RWMutex
}

// NewStore creates a new connection store
func NewStore() *Store {
	return &Store{}
}

// RegisterRelay registers a new relay connection
func (s *Store) RegisterRelay(subdomain, token string) *RelayConnection {
	relay := &RelayConnection{
		Subdomain:   subdomain,
		RelayToken:  token,
		ConnectedAt: time.Now(),
		LastPing:    time.Now(),
	}
	s.relays.Store(subdomain, relay)
	s.tokens.Store(token, relay)
	s.subdomains.Store(subdomain, true)
	return relay
}

// GetRelayBySubdomain gets a relay by subdomain
func (s *Store) GetRelayBySubdomain(subdomain string) *RelayConnection {
	if v, ok := s.relays.Load(subdomain); ok {
		return v.(*RelayConnection)
	}
	return nil
}

// GetRelayByToken gets a relay by token
func (s *Store) GetRelayByToken(token string) *RelayConnection {
	if v, ok := s.tokens.Load(token); ok {
		return v.(*RelayConnection)
	}
	return nil
}

// SetRelayConn sets the WebSocket connection for a relay
func (s *Store) SetRelayConn(subdomain string, conn *websocket.Conn) {
	if relay := s.GetRelayBySubdomain(subdomain); relay != nil {
		relay.mu.Lock()
		relay.Conn = conn
		relay.ConnectedAt = time.Now()
		relay.mu.Unlock()
	}
}

// RemoveRelay removes a relay connection
func (s *Store) RemoveRelay(subdomain string) {
	if relay := s.GetRelayBySubdomain(subdomain); relay != nil {
		s.relays.Delete(subdomain)
		s.tokens.Delete(relay.RelayToken)
		// Don't delete from subdomains to prevent reuse
	}
}

// AddClient adds a client connection to a relay
func (s *Store) AddClient(relay *RelayConnection, connID string, conn *websocket.Conn) *ClientConnection {
	client := &ClientConnection{
		ID:          connID,
		Conn:        conn,
		RelayConn:   relay,
		ConnectedAt: time.Now(),
	}
	relay.Clients.Store(connID, client)
	return client
}

// GetClient gets a client by connection ID
func (s *Store) GetClient(relay *RelayConnection, connID string) *ClientConnection {
	if v, ok := relay.Clients.Load(connID); ok {
		return v.(*ClientConnection)
	}
	return nil
}

// RemoveClient removes a client connection
func (s *Store) RemoveClient(relay *RelayConnection, connID string) {
	relay.Clients.Delete(connID)
}

// GenerateSubdomain generates a unique subdomain
func (s *Store) GenerateSubdomain() string {
	for {
		bytes := make([]byte, 4)
		rand.Read(bytes)
		subdomain := hex.EncodeToString(bytes)
		if _, exists := s.subdomains.Load(subdomain); !exists {
			return subdomain
		}
	}
}

// GenerateToken generates a secure token
func GenerateToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return "rt_" + hex.EncodeToString(bytes)
}

// IsRelayConnected checks if a relay is connected
func (r *RelayConnection) IsConnected() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.Conn != nil
}
