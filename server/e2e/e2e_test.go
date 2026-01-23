package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Noon-R/Devport/server/api"
	"github.com/Noon-R/Devport/server/config"
	"github.com/Noon-R/Devport/server/ws"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const testToken = "test-token"

func setupTestServer(t *testing.T) *httptest.Server {
	cfg := &config.Config{
		AuthToken:    testToken,
		ServerPort:   "0",
		WorkDir:      t.TempDir(),
		DataDir:      t.TempDir(),
		DevMode:      true,
		RelayEnabled: false,
	}

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// WebSocket
	wsHandler := ws.NewHandler(cfg)
	mux.Handle("/ws", wsHandler)

	// APIs
	fsHandler := api.NewFSHandler(cfg.WorkDir, cfg.AuthToken)
	mux.Handle("/api/fs/", fsHandler)
	mux.Handle("/api/fs", fsHandler)

	gitHandler := api.NewGitHandler(cfg.WorkDir, cfg.AuthToken)
	mux.Handle("/api/git/", gitHandler)

	chatHandler := api.NewChatHandler(cfg.AuthToken, wsHandler.GetSessionStore(), wsHandler.GetProcessManager())
	mux.Handle("/api/sessions/", chatHandler)

	return httptest.NewServer(mux)
}

func TestHealthCheck(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("Health check request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %s", result["status"])
	}
}

func TestWebSocketAuth(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + server.URL[4:] + "/ws"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect WebSocket: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "test complete")

	// Send auth request
	authReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "auth",
		"params":  map[string]string{"token": testToken},
		"id":      1,
	}
	if err := wsjson.Write(ctx, conn, authReq); err != nil {
		t.Fatalf("Failed to send auth request: %v", err)
	}

	// Read auth response
	var authResp map[string]interface{}
	if err := wsjson.Read(ctx, conn, &authResp); err != nil {
		t.Fatalf("Failed to read auth response: %v", err)
	}

	// Check for error
	if authResp["error"] != nil {
		t.Fatalf("Auth failed: %v", authResp["error"])
	}

	// Check result
	result, ok := authResp["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("Invalid result type: %T", authResp["result"])
	}

	if result["status"] != "authenticated" {
		t.Errorf("Expected status 'authenticated', got %v", result["status"])
	}
}

func TestWebSocketAuthInvalidToken(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	wsURL := "ws" + server.URL[4:] + "/ws"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect WebSocket: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "test complete")

	// Send auth request with wrong token
	authReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "auth",
		"params":  map[string]string{"token": "wrong-token"},
		"id":      1,
	}
	if err := wsjson.Write(ctx, conn, authReq); err != nil {
		t.Fatalf("Failed to send auth request: %v", err)
	}

	// Read auth response
	var authResp map[string]interface{}
	if err := wsjson.Read(ctx, conn, &authResp); err != nil {
		t.Fatalf("Failed to read auth response: %v", err)
	}

	// Check for error
	if authResp["error"] == nil {
		t.Error("Expected auth error for invalid token")
	}
}

func TestSessionCreate(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	wsURL := "ws" + server.URL[4:] + "/ws"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect WebSocket: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "test complete")

	// Authenticate
	authReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "auth",
		"params":  map[string]string{"token": testToken},
		"id":      1,
	}
	if err := wsjson.Write(ctx, conn, authReq); err != nil {
		t.Fatalf("Failed to send auth request: %v", err)
	}

	var authResp map[string]interface{}
	if err := wsjson.Read(ctx, conn, &authResp); err != nil {
		t.Fatalf("Failed to read auth response: %v", err)
	}

	if authResp["error"] != nil {
		t.Fatalf("Auth failed: %v", authResp["error"])
	}

	// Create session
	createReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "session.create",
		"params":  map[string]string{"title": "Test Session"},
		"id":      2,
	}
	if err := wsjson.Write(ctx, conn, createReq); err != nil {
		t.Fatalf("Failed to send create request: %v", err)
	}

	var createResp map[string]interface{}
	if err := wsjson.Read(ctx, conn, &createResp); err != nil {
		t.Fatalf("Failed to read create response: %v", err)
	}

	if createResp["error"] != nil {
		t.Fatalf("Session create failed: %v", createResp["error"])
	}

	result, ok := createResp["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("Invalid result type: %T", createResp["result"])
	}

	session, ok := result["session"].(map[string]interface{})
	if !ok {
		t.Fatalf("Invalid session type: %T", result["session"])
	}

	if session["id"] == nil || session["id"] == "" {
		t.Error("Session ID should not be empty")
	}

	if session["title"] != "Test Session" {
		t.Errorf("Expected title 'Test Session', got %v", session["title"])
	}
}

func TestSessionList(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	wsURL := "ws" + server.URL[4:] + "/ws"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect WebSocket: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "test complete")

	// Authenticate
	authReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "auth",
		"params":  map[string]string{"token": testToken},
		"id":      1,
	}
	if err := wsjson.Write(ctx, conn, authReq); err != nil {
		t.Fatalf("Failed to send auth request: %v", err)
	}

	var authResp map[string]interface{}
	if err := wsjson.Read(ctx, conn, &authResp); err != nil {
		t.Fatalf("Failed to read auth response: %v", err)
	}

	// List sessions
	listReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "session.list",
		"params":  map[string]interface{}{},
		"id":      2,
	}
	if err := wsjson.Write(ctx, conn, listReq); err != nil {
		t.Fatalf("Failed to send list request: %v", err)
	}

	var listResp map[string]interface{}
	if err := wsjson.Read(ctx, conn, &listResp); err != nil {
		t.Fatalf("Failed to read list response: %v", err)
	}

	if listResp["error"] != nil {
		t.Fatalf("Session list failed: %v", listResp["error"])
	}

	result, ok := listResp["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("Invalid result type: %T", listResp["result"])
	}

	sessions, ok := result["sessions"].([]interface{})
	if !ok {
		t.Fatalf("Invalid sessions type: %T", result["sessions"])
	}

	// Should be empty initially
	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions, got %d", len(sessions))
	}
}
