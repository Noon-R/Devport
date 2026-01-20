package session

import (
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

// Store manages sessions
type Store struct {
	sessions sync.Map
	workDir  string
}

// NewStore creates a new session store
func NewStore(workDir string) *Store {
	return &Store{
		workDir: workDir,
	}
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
	return session
}

// Get returns a session by ID
func (s *Store) Get(id string) *Session {
	if val, ok := s.sessions.Load(id); ok {
		return val.(*Session)
	}
	return nil
}

// List returns all sessions
func (s *Store) List() []*Session {
	var sessions []*Session
	s.sessions.Range(func(key, value interface{}) bool {
		sessions = append(sessions, value.(*Session))
		return true
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
	}
}
