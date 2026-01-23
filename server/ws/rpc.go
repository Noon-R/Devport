package ws

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log"
	"time"

	"github.com/Noon-R/Devport/server/agent"
	"github.com/Noon-R/Devport/server/session"
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
	ErrCodeParseError      = -32700
	ErrCodeInvalidRequest  = -32600
	ErrCodeMethodNotFound  = -32601
	ErrCodeInvalidParams   = -32602
	ErrCodeInternal        = -32603
	ErrCodeAuthFailed      = -32001
	ErrCodeUnauthorized    = -32002
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
	case "chat.interrupt":
		return h.handleChatInterrupt(ctx, state, req)
	case "chat.permission_response":
		return h.handlePermissionResponse(ctx, state, req)
	case "chat.question_response":
		return h.handleQuestionResponse(ctx, state, req)
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
	sessions := h.sessionStore.List()
	return successResponse(req.ID, map[string]interface{}{
		"sessions": sessions,
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

	session := h.sessionStore.Create(params.Title)
	return successResponse(req.ID, map[string]interface{}{
		"session": session,
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

	// Check if session exists
	sess := h.sessionStore.Get(params.SessionID)
	if sess == nil {
		return errorResponse(req.ID, ErrCodeSessionNotFound, "Session not found")
	}

	// Get or create agent for this session
	_, err := h.processManager.GetOrCreate(ctx, params.SessionID)
	if err != nil {
		return errorResponse(req.ID, ErrCodeInternal, err.Error())
	}

	state.sessionID = params.SessionID

	// Get history for this session
	history := h.sessionStore.GetHistory(params.SessionID)

	return successResponse(req.ID, map[string]interface{}{
		"session_id": params.SessionID,
		"status":     "attached",
		"history":    history,
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

	// Get agent for this session
	ag, err := h.processManager.GetOrCreate(ctx, params.SessionID)
	if err != nil {
		return errorResponse(req.ID, ErrCodeInternal, err.Error())
	}

	// Save user message to history
	userMsg := session.HistoryMessage{
		ID:        uuid.New().String(),
		Role:      "user",
		Content:   params.Content,
		Timestamp: time.Now(),
	}
	h.sessionStore.AddMessage(params.SessionID, userMsg)

	// Initialize assistant message tracking
	state.mu.Lock()
	state.currentAssistantMsgID = uuid.New().String()
	state.currentAssistantContent = ""
	state.currentAssistantTools = nil
	state.mu.Unlock()

	// Send message and stream events
	go func() {
		events, err := ag.SendMessage(ctx, params.Content)
		if err != nil {
			log.Printf("SendMessage error: %v", err)
			h.SendNotification(ctx, state, "chat.error", map[string]interface{}{
				"session_id": params.SessionID,
				"error":      err.Error(),
			})
			return
		}

		for event := range events {
			h.sendEventNotification(ctx, state, params.SessionID, &event)
		}
	}()

	return successResponse(req.ID, map[string]bool{"accepted": true})
}

// handleChatInterrupt handles interrupt request
func (h *Handler) handleChatInterrupt(ctx context.Context, state *ConnState, req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		SessionID string `json:"session_id"`
	}
	json.Unmarshal(req.Params, &params)

	ag, err := h.processManager.GetOrCreate(ctx, params.SessionID)
	if err != nil {
		return errorResponse(req.ID, ErrCodeInternal, err.Error())
	}

	if err := ag.Interrupt(ctx); err != nil {
		return errorResponse(req.ID, ErrCodeInternal, err.Error())
	}

	return successResponse(req.ID, map[string]bool{"success": true})
}

// handlePermissionResponse handles permission response from user
func (h *Handler) handlePermissionResponse(ctx context.Context, state *ConnState, req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		SessionID    string `json:"session_id"`
		PermissionID string `json:"permission_id"`
		Allowed      bool   `json:"allowed"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errorResponse(req.ID, ErrCodeInvalidParams, "Invalid params")
	}

	ag, err := h.processManager.GetOrCreate(ctx, params.SessionID)
	if err != nil {
		return errorResponse(req.ID, ErrCodeInternal, err.Error())
	}

	if err := ag.RespondToPermission(ctx, params.PermissionID, params.Allowed); err != nil {
		return errorResponse(req.ID, ErrCodeInternal, err.Error())
	}

	return successResponse(req.ID, map[string]bool{"success": true})
}

// handleQuestionResponse handles question response from user
func (h *Handler) handleQuestionResponse(ctx context.Context, state *ConnState, req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		SessionID  string `json:"session_id"`
		QuestionID string `json:"question_id"`
		Answer     string `json:"answer"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errorResponse(req.ID, ErrCodeInvalidParams, "Invalid params")
	}

	ag, err := h.processManager.GetOrCreate(ctx, params.SessionID)
	if err != nil {
		return errorResponse(req.ID, ErrCodeInternal, err.Error())
	}

	if err := ag.RespondToQuestion(ctx, params.QuestionID, params.Answer); err != nil {
		return errorResponse(req.ID, ErrCodeInternal, err.Error())
	}

	return successResponse(req.ID, map[string]bool{"success": true})
}

// sendEventNotification sends an event as a JSON-RPC notification
func (h *Handler) sendEventNotification(ctx context.Context, state *ConnState, sessionID string, event *agent.Event) {
	var method string
	params := map[string]interface{}{
		"session_id": sessionID,
	}

	switch event.Type {
	case agent.EventTypeText:
		method = "chat.text"
		params["content"] = event.Content
		// Track assistant content
		state.mu.Lock()
		state.currentAssistantContent += event.Content
		state.mu.Unlock()

	case agent.EventTypeToolCall:
		method = "chat.tool_call"
		params["tool_use_id"] = event.ToolUseID
		params["tool_name"] = event.ToolName
		params["input"] = event.ToolInput
		// Track tool call
		state.mu.Lock()
		state.currentAssistantTools = append(state.currentAssistantTools, ToolCallState{
			ID:     event.ToolUseID,
			Name:   event.ToolName,
			Input:  event.ToolInput,
			Status: "pending",
		})
		state.mu.Unlock()

	case agent.EventTypeToolResult:
		method = "chat.tool_result"
		params["tool_use_id"] = event.ToolUseID
		params["output"] = event.ToolOutput
		// Update tool call status
		state.mu.Lock()
		for i := range state.currentAssistantTools {
			if state.currentAssistantTools[i].ID == event.ToolUseID {
				state.currentAssistantTools[i].Output = event.ToolOutput
				state.currentAssistantTools[i].Status = "completed"
				break
			}
		}
		state.mu.Unlock()

	case agent.EventTypePermissionRequest:
		method = "chat.permission_request"
		params["permission_id"] = event.PermissionID
		params["tool_name"] = event.ToolName
		params["description"] = event.Content

	case agent.EventTypeAskUserQuestion:
		method = "chat.ask_user_question"
		params["question_id"] = event.QuestionID
		params["question"] = event.Question
		params["options"] = event.Options

	case agent.EventTypeDone:
		method = "chat.done"
		// Save assistant message to history
		state.mu.Lock()
		if state.currentAssistantMsgID != "" {
			toolCalls := make([]session.ToolCallInfo, len(state.currentAssistantTools))
			for i, tc := range state.currentAssistantTools {
				toolCalls[i] = session.ToolCallInfo{
					ID:     tc.ID,
					Name:   tc.Name,
					Input:  tc.Input,
					Output: tc.Output,
					Status: tc.Status,
				}
			}
			assistantMsg := session.HistoryMessage{
				ID:        state.currentAssistantMsgID,
				Role:      "assistant",
				Content:   state.currentAssistantContent,
				ToolCalls: toolCalls,
				Timestamp: time.Now(),
			}
			h.sessionStore.AddMessage(sessionID, assistantMsg)
			// Reset tracking
			state.currentAssistantMsgID = ""
			state.currentAssistantContent = ""
			state.currentAssistantTools = nil
		}
		state.mu.Unlock()

	case agent.EventTypeError:
		method = "chat.error"
		params["error"] = event.Error

	case agent.EventTypeSystem:
		method = "chat.system"
		params["message"] = event.Content
		// Save system message to history
		sysMsg := session.HistoryMessage{
			ID:        uuid.New().String(),
			Role:      "system",
			Content:   event.Content,
			Timestamp: time.Now(),
		}
		h.sessionStore.AddMessage(sessionID, sysMsg)

	case agent.EventTypeInterrupted:
		method = "chat.interrupted"
		// Save partial assistant message if any
		state.mu.Lock()
		if state.currentAssistantMsgID != "" && (state.currentAssistantContent != "" || len(state.currentAssistantTools) > 0) {
			toolCalls := make([]session.ToolCallInfo, len(state.currentAssistantTools))
			for i, tc := range state.currentAssistantTools {
				toolCalls[i] = session.ToolCallInfo{
					ID:     tc.ID,
					Name:   tc.Name,
					Input:  tc.Input,
					Output: tc.Output,
					Status: tc.Status,
				}
			}
			assistantMsg := session.HistoryMessage{
				ID:        state.currentAssistantMsgID,
				Role:      "assistant",
				Content:   state.currentAssistantContent,
				ToolCalls: toolCalls,
				Timestamp: time.Now(),
			}
			h.sessionStore.AddMessage(sessionID, assistantMsg)
		}
		// Reset tracking
		state.currentAssistantMsgID = ""
		state.currentAssistantContent = ""
		state.currentAssistantTools = nil
		state.mu.Unlock()

	default:
		return
	}

	h.SendNotification(ctx, state, method, params)
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
