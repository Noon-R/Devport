package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Noon-R/Devport/server/agent"
	"github.com/Noon-R/Devport/server/process"
	"github.com/Noon-R/Devport/server/session"
	"github.com/google/uuid"
)

// ChatHandler handles chat REST API operations
type ChatHandler struct {
	authToken      string
	sessionStore   *session.Store
	processManager *process.Manager

	// Pending responses for async operations
	pendingResponses sync.Map // map[requestID]chan *ResponseEvent
}

// ResponseEvent represents an event to be sent back to the client
type ResponseEvent struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// NewChatHandler creates a new chat handler
func NewChatHandler(authToken string, sessionStore *session.Store, processManager *process.Manager) *ChatHandler {
	return &ChatHandler{
		authToken:      authToken,
		sessionStore:   sessionStore,
		processManager: processManager,
	}
}

// ServeHTTP implements http.Handler
func (h *ChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	token := r.Header.Get("Authorization")
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	if strings.TrimPrefix(token, "Bearer ") != h.authToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Route based on path and method
	path := strings.TrimPrefix(r.URL.Path, "/api")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	switch {
	// GET /api/sessions/:id/messages - Get message history
	case len(parts) == 3 && parts[0] == "sessions" && parts[2] == "messages" && r.Method == http.MethodGet:
		h.handleGetHistory(w, r, parts[1])

	// POST /api/sessions/:id/messages - Send message
	case len(parts) == 3 && parts[0] == "sessions" && parts[2] == "messages" && r.Method == http.MethodPost:
		h.handleSendMessage(w, r, parts[1])

	// POST /api/sessions/:id/cancel - Cancel generation
	case len(parts) == 3 && parts[0] == "sessions" && parts[2] == "cancel" && r.Method == http.MethodPost:
		h.handleCancel(w, r, parts[1])

	// POST /api/permissions/:id - Respond to permission
	case len(parts) == 2 && parts[0] == "permissions" && r.Method == http.MethodPost:
		h.handlePermissionResponse(w, r, parts[1])

	// POST /api/questions/:id - Respond to question
	case len(parts) == 2 && parts[0] == "questions" && r.Method == http.MethodPost:
		h.handleQuestionResponse(w, r, parts[1])

	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// handleGetHistory returns message history for a session
func (h *ChatHandler) handleGetHistory(w http.ResponseWriter, r *http.Request, sessionID string) {
	// Check if session exists
	sess := h.sessionStore.Get(sessionID)
	if sess == nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	history := h.sessionStore.GetHistory(sessionID)

	// Support pagination via query params
	afterSeq := r.URL.Query().Get("after")
	if afterSeq != "" {
		// Find messages after the given sequence (using message ID as sequence)
		var filtered []session.HistoryMessage
		found := false
		for _, msg := range history {
			if found {
				filtered = append(filtered, msg)
			}
			if msg.ID == afterSeq {
				found = true
			}
		}
		history = filtered
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"session_id": sessionID,
		"messages":   history,
	})
}

// handleSendMessage handles sending a message to the session
func (h *ChatHandler) handleSendMessage(w http.ResponseWriter, r *http.Request, sessionID string) {
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check if session exists
	sess := h.sessionStore.Get(sessionID)
	if sess == nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Get or create agent for this session
	ctx := r.Context()
	ag, err := h.processManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Save user message to history
	userMsg := session.HistoryMessage{
		ID:        uuid.New().String(),
		Role:      "user",
		Content:   req.Content,
		Timestamp: time.Now(),
	}
	h.sessionStore.AddMessage(sessionID, userMsg)

	// Generate request ID for tracking
	requestID := uuid.New().String()

	// Return immediately with request ID
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"request_id":  requestID,
		"message_id":  userMsg.ID,
		"session_id":  sessionID,
		"status":      "accepted",
	})

	// Process message asynchronously
	go h.processMessage(ctx, sessionID, req.Content, requestID, ag)
}

// processMessage processes the message and saves the assistant response
func (h *ChatHandler) processMessage(ctx context.Context, sessionID, content, requestID string, ag agent.Agent) {
	events, err := ag.SendMessage(ctx, content)
	if err != nil {
		return
	}

	var assistantContent strings.Builder
	var toolCalls []session.ToolCallInfo
	assistantMsgID := uuid.New().String()

	for event := range events {
		switch event.Type {
		case agent.EventTypeText:
			assistantContent.WriteString(event.Content)

		case agent.EventTypeToolCall:
			toolCalls = append(toolCalls, session.ToolCallInfo{
				ID:     event.ToolUseID,
				Name:   event.ToolName,
				Input:  event.ToolInput,
				Status: "pending",
			})

		case agent.EventTypeToolResult:
			for i := range toolCalls {
				if toolCalls[i].ID == event.ToolUseID {
					toolCalls[i].Output = event.ToolOutput
					toolCalls[i].Status = "completed"
					break
				}
			}

		case agent.EventTypeDone, agent.EventTypeInterrupted:
			// Save assistant message
			if assistantContent.Len() > 0 || len(toolCalls) > 0 {
				assistantMsg := session.HistoryMessage{
					ID:        assistantMsgID,
					Role:      "assistant",
					Content:   assistantContent.String(),
					ToolCalls: toolCalls,
					Timestamp: time.Now(),
				}
				h.sessionStore.AddMessage(sessionID, assistantMsg)
			}

		case agent.EventTypeSystem:
			// Save system message
			sysMsg := session.HistoryMessage{
				ID:        uuid.New().String(),
				Role:      "system",
				Content:   event.Content,
				Timestamp: time.Now(),
			}
			h.sessionStore.AddMessage(sessionID, sysMsg)
		}
	}
}

// handleCancel handles canceling the current generation
func (h *ChatHandler) handleCancel(w http.ResponseWriter, r *http.Request, sessionID string) {
	// Check if session exists
	sess := h.sessionStore.Get(sessionID)
	if sess == nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	ctx := r.Context()
	ag, err := h.processManager.GetOrCreate(ctx, sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := ag.Interrupt(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// handlePermissionResponse handles responding to a permission request
func (h *ChatHandler) handlePermissionResponse(w http.ResponseWriter, r *http.Request, permissionID string) {
	var req struct {
		SessionID string `json:"session_id"`
		Allowed   bool   `json:"allowed"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	ag, err := h.processManager.GetOrCreate(ctx, req.SessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := ag.RespondToPermission(ctx, permissionID, req.Allowed); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// handleQuestionResponse handles responding to a user question
func (h *ChatHandler) handleQuestionResponse(w http.ResponseWriter, r *http.Request, questionID string) {
	var req struct {
		SessionID string `json:"session_id"`
		Answer    string `json:"answer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	ag, err := h.processManager.GetOrCreate(ctx, req.SessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := ag.RespondToQuestion(ctx, questionID, req.Answer); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
