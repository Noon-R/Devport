package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Noon-R/Devport/server/config"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// EnvelopeType defines the type of envelope message
type EnvelopeType string

const (
	EnvelopeTypeMessage      EnvelopeType = "message"
	EnvelopeTypeDisconnected EnvelopeType = "disconnected"
	EnvelopeTypeConnected    EnvelopeType = "connected"
)

// Envelope wraps messages for multiplexing
type Envelope struct {
	ConnectionID string          `json:"connection_id"`
	Type         EnvelopeType    `json:"type"`
	Payload      json.RawMessage `json:"payload,omitempty"`
}

// RelayConfig holds the relay configuration
type RelayConfig struct {
	Subdomain   string `json:"subdomain"`
	RelayToken  string `json:"relay_token"`
	RelayServer string `json:"relay_server"`
}

// MessageHandler handles messages from clients
type MessageHandler func(ctx context.Context, connID string, data []byte) ([]byte, error)

// Client manages connection to the relay server
type Client struct {
	cfg         *config.Config
	relayConfig *RelayConfig
	conn        *websocket.Conn
	handler     MessageHandler
	mu          sync.RWMutex
	done        chan struct{}
	connections sync.Map
}

// NewClient creates a new relay client
func NewClient(cfg *config.Config) *Client {
	return &Client{
		cfg:  cfg,
		done: make(chan struct{}),
	}
}

// SetHandler sets the message handler for client messages
func (c *Client) SetHandler(handler MessageHandler) {
	c.handler = handler
}

// Start starts the relay client
func (c *Client) Start(ctx context.Context) error {
	if !c.cfg.RelayEnabled {
		log.Println("Relay is disabled")
		return nil
	}
	if err := c.loadOrRegister(ctx); err != nil {
		return fmt.Errorf("failed to load or register relay: %w", err)
	}
	go c.connectLoop(ctx)
	return nil
}

// Stop stops the relay client
func (c *Client) Stop() {
	close(c.done)
	c.mu.Lock()
	if c.conn != nil {
		c.conn.Close(websocket.StatusNormalClosure, "shutting down")
	}
	c.mu.Unlock()
}

// GetRemoteURL returns the remote URL for clients
func (c *Client) GetRemoteURL() string {
	if c.relayConfig == nil {
		return ""
	}
	return fmt.Sprintf("https://%s.%s", c.relayConfig.Subdomain, c.relayConfig.RelayServer)
}

// SendToClient sends a message to a specific client
func (c *Client) SendToClient(ctx context.Context, connID string, data []byte) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	if conn == nil {
		return fmt.Errorf("not connected to relay")
	}
	envelope := &Envelope{
		ConnectionID: connID,
		Type:         EnvelopeTypeMessage,
		Payload:      data,
	}
	return wsjson.Write(ctx, conn, envelope)
}

func (c *Client) loadOrRegister(ctx context.Context) error {
	configPath := c.getConfigPath()
	if data, err := os.ReadFile(configPath); err == nil {
		var relayConfig RelayConfig
		if err := json.Unmarshal(data, &relayConfig); err == nil {
			if err := c.refresh(ctx, &relayConfig); err == nil {
				c.relayConfig = &relayConfig
				log.Printf("Loaded relay config: subdomain=%s", relayConfig.Subdomain)
				return nil
			}
			log.Printf("Refresh failed, re-registering...")
		}
	}
	return c.register(ctx)
}

func (c *Client) register(ctx context.Context) error {
	url := c.cfg.RelayURL + "/api/relay/register"
	reqBody := map[string]string{"client_version": "1.0.0"}
	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registration failed: %s", string(body))
	}
	var relayConfig RelayConfig
	if err := json.NewDecoder(resp.Body).Decode(&relayConfig); err != nil {
		return err
	}
	c.relayConfig = &relayConfig
	if err := c.saveConfig(&relayConfig); err != nil {
		log.Printf("Warning: failed to save relay config: %v", err)
	}
	log.Printf("Registered with relay: subdomain=%s", relayConfig.Subdomain)
	return nil
}

func (c *Client) refresh(ctx context.Context, relayConfig *RelayConfig) error {
	url := c.cfg.RelayURL + "/api/relay/refresh"
	reqBody := map[string]string{"relay_token": relayConfig.RelayToken}
	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("refresh failed: %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) connectLoop(ctx context.Context) {
	backoff := time.Second
	maxBackoff := 30 * time.Second
	for {
		select {
		case <-c.done:
			return
		case <-ctx.Done():
			return
		default:
		}
		if err := c.connect(ctx); err != nil {
			log.Printf("Relay connection error: %v", err)
			select {
			case <-time.After(backoff):
				backoff = min(backoff*2, maxBackoff)
			case <-c.done:
				return
			case <-ctx.Done():
				return
			}
			continue
		}
		backoff = time.Second
	}
}

func (c *Client) connect(ctx context.Context) error {
	if c.relayConfig == nil {
		return fmt.Errorf("no relay config")
	}
	wsURL := strings.Replace(c.cfg.RelayURL, "https://", "wss://", 1)
	wsURL = strings.Replace(wsURL, "http://", "ws://", 1)
	wsURL = fmt.Sprintf("%s/relay", strings.Replace(wsURL, "://", "://"+c.relayConfig.Subdomain+".", 1))
	log.Printf("Connecting to relay: %s", wsURL)
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return err
	}
	authReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "register",
		"params":  map[string]string{"relay_token": c.relayConfig.RelayToken},
		"id":      1,
	}
	if err := wsjson.Write(ctx, conn, authReq); err != nil {
		conn.Close(websocket.StatusNormalClosure, "auth failed")
		return err
	}
	var authResp map[string]interface{}
	if err := wsjson.Read(ctx, conn, &authResp); err != nil {
		conn.Close(websocket.StatusNormalClosure, "auth failed")
		return err
	}
	if authResp["error"] != nil {
		conn.Close(websocket.StatusNormalClosure, "auth failed")
		return fmt.Errorf("auth failed: %v", authResp["error"])
	}
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	log.Printf("Connected to relay server")
	return c.messageLoop(ctx, conn)
}

func (c *Client) messageLoop(ctx context.Context, conn *websocket.Conn) error {
	defer func() {
		c.mu.Lock()
		c.conn = nil
		c.mu.Unlock()
	}()
	for {
		msgType, data, err := conn.Read(ctx)
		if err != nil {
			return err
		}
		if msgType != websocket.MessageText {
			continue
		}
		var envelope Envelope
		if err := json.Unmarshal(data, &envelope); err != nil {
			log.Printf("Failed to unmarshal envelope: %v", err)
			continue
		}
		switch envelope.Type {
		case EnvelopeTypeConnected:
			log.Printf("Client connected: %s", envelope.ConnectionID)
		case EnvelopeTypeDisconnected:
			log.Printf("Client disconnected: %s", envelope.ConnectionID)
		case EnvelopeTypeMessage:
			go c.handleClientMessage(ctx, envelope.ConnectionID, envelope.Payload)
		}
	}
}

func (c *Client) handleClientMessage(ctx context.Context, connID string, data []byte) {
	if c.handler == nil {
		return
	}
	resp, err := c.handler(ctx, connID, data)
	if err != nil {
		log.Printf("Handler error for %s: %v", connID, err)
		return
	}
	if resp != nil {
		if err := c.SendToClient(ctx, connID, resp); err != nil {
			log.Printf("Failed to send response to %s: %v", connID, err)
		}
	}
}

func (c *Client) getConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".devport", "relay", "config.json")
}

func (c *Client) saveConfig(relayConfig *RelayConfig) error {
	configPath := c.getConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(relayConfig, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0600)
}
