package ws

import (
	"context"
	"crypto/subtle"
	"encoding/json"

	"github.com/google/uuid"
)

// JSON-RPC 2.0 structures
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
	ID      interface{}   `json:"id,omitempty"`
}

type JSONRPCNotification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Error codes
const (
	ErrCodeParseError     = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603
	ErrCodeAuthFailed     = -32001
	ErrCodeUnauthorized   = -32002
	ErrCodeSessionNotFound = -32003
)

func (h *Handler) handleRequest(ctx context.Context, state *ConnState, req *JSONRPCRequest) *JSONRPCResponse {
	// Require authentication for all methods except "auth"
	if req.Method != "auth" && !state.authenticated {
		return errorResponse(req.ID, ErrCodeUnauthorized, "Not authenticated")
	}

	switch req.Method {
	case "auth":
		return h.handleAuth(ctx, state, req)
	case "session.list":
		return h.handleSessionList(ctx, state, req)
	case "session.create":
		return h.handleSessionCreate(ctx, state, req)
	case "chat.attach":
		return h.handleChatAttach(ctx, state, req)
	case "chat.message":
		return h.handleChatMessage(ctx, state, req)
	default:
		return errorResponse(req.ID, ErrCodeMethodNotFound, "Method not found: "+req.Method)
	}
}

// handleAuth authenticates the client with a token
func (h *Handler) handleAuth(ctx context.Context, state *ConnState, req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errorResponse(req.ID, ErrCodeInvalidParams, "Invalid params")
	}

	// Constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(params.Token), []byte(h.cfg.AuthToken)) != 1 {
		return errorResponse(req.ID, ErrCodeAuthFailed, "Invalid token")
	}

	state.authenticated = true
	return successResponse(req.ID, map[string]bool{"success": true})
}

// handleSessionList returns the list of sessions
func (h *Handler) handleSessionList(ctx context.Context, state *ConnState, req *JSONRPCRequest) *JSONRPCResponse {
	// TODO: Implement session store
	return successResponse(req.ID, map[string]interface{}{
		"sessions": []interface{}{},
	})
}

// handleSessionCreate creates a new session
func (h *Handler) handleSessionCreate(ctx context.Context, state *ConnState, req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		Title string `json:"title"`
	}
	json.Unmarshal(req.Params, &params)

	if params.Title == "" {
		params.Title = "New Chat"
	}

	// TODO: Implement session store
	sessionID := "session_" + generateID()
	return successResponse(req.ID, map[string]interface{}{
		"session": map[string]interface{}{
			"id":    sessionID,
			"title": params.Title,
		},
	})
}

// handleChatAttach attaches to a session
func (h *Handler) handleChatAttach(ctx context.Context, state *ConnState, req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errorResponse(req.ID, ErrCodeInvalidParams, "Invalid params")
	}

	state.sessionID = params.SessionID
	return successResponse(req.ID, map[string]interface{}{
		"session_id": params.SessionID,
		"status":     "attached",
	})
}

// handleChatMessage handles a chat message
func (h *Handler) handleChatMessage(ctx context.Context, state *ConnState, req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		SessionID string `json:"session_id"`
		Content   string `json:"content"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errorResponse(req.ID, ErrCodeInvalidParams, "Invalid params")
	}

	// TODO: Implement Claude CLI integration
	// For now, echo back the message
	go func() {
		h.SendNotification(ctx, state, "chat.text", map[string]interface{}{
			"session_id": params.SessionID,
			"content":    "Echo: " + params.Content,
		})
		h.SendNotification(ctx, state, "chat.done", map[string]interface{}{
			"session_id": params.SessionID,
		})
	}()

	return successResponse(req.ID, map[string]bool{"accepted": true})
}

// Helper functions
func successResponse(id interface{}, result interface{}) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}
}

func errorResponse(id interface{}, code int, message string) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
		ID: id,
	}
}

func generateID() string {
	return uuid.New().String()[:8]
}
