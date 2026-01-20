package process

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/Noon-R/Devport/server/agent"
	"github.com/Noon-R/Devport/server/agent/claude"
)

// Manager manages Claude CLI processes for sessions
type Manager struct {
	processes   sync.Map // map[sessionID]*processEntry
	workDir     string
	idleTimeout time.Duration
}

type processEntry struct {
	agent     agent.Agent
	refCount  int
	lastUsed  time.Time
	mu        sync.Mutex
	cancelCtx context.CancelFunc
}

// NewManager creates a new process manager
func NewManager(workDir string, idleTimeout time.Duration) *Manager {
	m := &Manager{
		workDir:     workDir,
		idleTimeout: idleTimeout,
	}

	// Start cleanup goroutine
	go m.cleanupLoop()

	return m
}

// GetOrCreate returns an existing agent or creates a new one
func (m *Manager) GetOrCreate(ctx context.Context, sessionID string) (agent.Agent, error) {
	// Try to get existing
	if val, ok := m.processes.Load(sessionID); ok {
		entry := val.(*processEntry)
		entry.mu.Lock()
		entry.refCount++
		entry.lastUsed = time.Now()
		entry.mu.Unlock()
		return entry.agent, nil
	}

	// Create new
	ctx, cancel := context.WithCancel(ctx)
	ag := claude.New(sessionID, m.workDir)

	entry := &processEntry{
		agent:     ag,
		refCount:  1,
		lastUsed:  time.Now(),
		cancelCtx: cancel,
	}

	m.processes.Store(sessionID, entry)
	log.Printf("Created new Claude process for session %s", sessionID)

	return ag, nil
}

// Release decrements the reference count for a session
func (m *Manager) Release(sessionID string) {
	if val, ok := m.processes.Load(sessionID); ok {
		entry := val.(*processEntry)
		entry.mu.Lock()
		entry.refCount--
		entry.lastUsed = time.Now()
		entry.mu.Unlock()
	}
}

// Close terminates a specific session's process
func (m *Manager) Close(sessionID string) {
	if val, ok := m.processes.LoadAndDelete(sessionID); ok {
		entry := val.(*processEntry)
		entry.cancelCtx()
		entry.agent.Close()
		log.Printf("Closed Claude process for session %s", sessionID)
	}
}

// CloseAll terminates all processes
func (m *Manager) CloseAll() {
	m.processes.Range(func(key, value interface{}) bool {
		sessionID := key.(string)
		m.Close(sessionID)
		return true
	})
}

func (m *Manager) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.cleanup()
	}
}

func (m *Manager) cleanup() {
	now := time.Now()

	m.processes.Range(func(key, value interface{}) bool {
		sessionID := key.(string)
		entry := value.(*processEntry)

		entry.mu.Lock()
		idle := entry.refCount == 0 && now.Sub(entry.lastUsed) > m.idleTimeout
		entry.mu.Unlock()

		if idle {
			m.Close(sessionID)
			log.Printf("Cleaned up idle process for session %s", sessionID)
		}

		return true
	})
}
