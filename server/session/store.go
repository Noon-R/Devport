package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Session represents a chat session
type Session struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	WorkDir   string    `json:"work_dir"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// HistoryMessage represents a message in the session history
type HistoryMessage struct {
	ID        string                 `json:"id"`
	Role      string                 `json:"role"` // "user", "assistant", "system"
	Content   string                 `json:"content"`
	ToolCalls []ToolCallInfo         `json:"tool_calls,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// ToolCallInfo represents a tool call in a message
type ToolCallInfo struct {
	ID     string                 `json:"id"`
	Name   string                 `json:"name"`
	Input  map[string]interface{} `json:"input,omitempty"`
	Output string                 `json:"output,omitempty"`
	Status string                 `json:"status"` // "pending", "completed", "error"
}

// Store manages sessions
type Store struct {
	sessions    sync.Map
	histories   sync.Map // map[sessionID][]HistoryMessage
	workDir     string
	sessionsDir string
}

// NewStore creates a new session store
func NewStore(workDir string) *Store {
	sessionsDir := filepath.Join(workDir, ".devport", "sessions")
	os.MkdirAll(sessionsDir, 0755)

	store := &Store{
		workDir:     workDir,
		sessionsDir: sessionsDir,
	}

	// Load existing sessions from disk
	store.loadFromDisk()

	return store
}

// Create creates a new session
func (s *Store) Create(title string) *Session {
	session := &Session{
		ID:        uuid.New().String(),
		Title:     title,
		WorkDir:   s.workDir,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.sessions.Store(session.ID, session)
	s.histories.Store(session.ID, []HistoryMessage{})

	// Save to disk
	s.saveSessionToDisk(session)

	return session
}

// Get returns a session by ID
func (s *Store) Get(id string) *Session {
	if val, ok := s.sessions.Load(id); ok {
		return val.(*Session)
	}
	return nil
}

// List returns all sessions sorted by UpdatedAt descending
func (s *Store) List() []*Session {
	var sessions []*Session
	s.sessions.Range(func(key, value interface{}) bool {
		sessions = append(sessions, value.(*Session))
		return true
	})
	// Sort by UpdatedAt descending (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})
	return sessions
}

// Delete removes a session
func (s *Store) Delete(id string) {
	s.sessions.Delete(id)
}

// UpdateTitle updates the session title
func (s *Store) UpdateTitle(id, title string) {
	if val, ok := s.sessions.Load(id); ok {
		session := val.(*Session)
		session.Title = title
		session.UpdatedAt = time.Now()
		s.saveSessionToDisk(session)
	}
}

// AddMessage adds a message to the session history
func (s *Store) AddMessage(sessionID string, msg HistoryMessage) {
	val, ok := s.histories.Load(sessionID)
	if !ok {
		val = []HistoryMessage{}
	}
	history := val.([]HistoryMessage)
	history = append(history, msg)
	s.histories.Store(sessionID, history)

	// Update session timestamp
	if sessionVal, ok := s.sessions.Load(sessionID); ok {
		session := sessionVal.(*Session)
		session.UpdatedAt = time.Now()
		s.saveSessionToDisk(session)
	}

	// Save history to disk
	s.saveHistoryToDisk(sessionID, history)
}

// UpdateLastAssistantMessage updates the last assistant message in history
func (s *Store) UpdateLastAssistantMessage(sessionID string, content string, toolCalls []ToolCallInfo) {
	val, ok := s.histories.Load(sessionID)
	if !ok {
		return
	}
	history := val.([]HistoryMessage)

	// Find last assistant message
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "assistant" {
			history[i].Content = content
			history[i].ToolCalls = toolCalls
			s.histories.Store(sessionID, history)
			s.saveHistoryToDisk(sessionID, history)
			return
		}
	}
}

// GetHistory returns the message history for a session
func (s *Store) GetHistory(sessionID string) []HistoryMessage {
	val, ok := s.histories.Load(sessionID)
	if !ok {
		return []HistoryMessage{}
	}
	return val.([]HistoryMessage)
}

// loadFromDisk loads all sessions from disk on startup
func (s *Store) loadFromDisk() {
	entries, err := os.ReadDir(s.sessionsDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		sessionID := entry.Name()
		sessionDir := filepath.Join(s.sessionsDir, sessionID)

		// Load metadata
		metaPath := filepath.Join(sessionDir, "meta.json")
		metaData, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}

		var session Session
		if err := json.Unmarshal(metaData, &session); err != nil {
			continue
		}
		s.sessions.Store(session.ID, &session)

		// Load history
		historyPath := filepath.Join(sessionDir, "history.json")
		historyData, err := os.ReadFile(historyPath)
		if err != nil {
			s.histories.Store(sessionID, []HistoryMessage{})
			continue
		}

		var history []HistoryMessage
		if err := json.Unmarshal(historyData, &history); err != nil {
			s.histories.Store(sessionID, []HistoryMessage{})
			continue
		}
		s.histories.Store(sessionID, history)
	}
}

// saveSessionToDisk saves session metadata to disk
func (s *Store) saveSessionToDisk(session *Session) error {
	sessionDir := filepath.Join(s.sessionsDir, session.ID)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return err
	}

	metaPath := filepath.Join(sessionDir, "meta.json")
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(metaPath, data, 0644)
}

// saveHistoryToDisk saves message history to disk
func (s *Store) saveHistoryToDisk(sessionID string, history []HistoryMessage) error {
	sessionDir := filepath.Join(s.sessionsDir, sessionID)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return err
	}

	historyPath := filepath.Join(sessionDir, "history.json")
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(historyPath, data, 0644)
}
