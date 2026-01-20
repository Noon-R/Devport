package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"
	"time"

	"github.com/Noon-R/Devport/server/agent"
)

// Claude implements the Agent interface for Claude CLI
type Claude struct {
	sessionID string
	workDir   string

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	running    bool
	mu         sync.Mutex
	cancelFunc context.CancelFunc

	// For permission/question responses
	pendingResponses chan pendingResponse
}

type pendingResponse struct {
	Type string
	Data interface{}
}

// New creates a new Claude agent
func New(sessionID, workDir string) *Claude {
	return &Claude{
		sessionID:        sessionID,
		workDir:          workDir,
		pendingResponses: make(chan pendingResponse, 10),
	}
}

// Start starts the Claude CLI process
func (c *Claude) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cmd != nil {
		return fmt.Errorf("process already started")
	}

	ctx, cancel := context.WithCancel(ctx)
	c.cancelFunc = cancel

	c.cmd = exec.CommandContext(ctx, "claude",
		"--output-format", "stream-json",
		"--input-format", "stream-json",
		"--permission-prompt-tool", "stdio",
		"--session-id", c.sessionID,
	)
	c.cmd.Dir = c.workDir

	var err error
	c.stdin, err = c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	c.stdout, err = c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	c.stderr, err = c.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	// Log stderr
	go func() {
		scanner := bufio.NewScanner(c.stderr)
		for scanner.Scan() {
			log.Printf("[Claude stderr] %s", scanner.Text())
		}
	}()

	log.Printf("Claude CLI started for session %s", c.sessionID)
	return nil
}

// SendMessage sends a message to the Claude CLI
func (c *Claude) SendMessage(ctx context.Context, message string) (<-chan agent.Event, error) {
	c.mu.Lock()
	if c.cmd == nil {
		c.mu.Unlock()
		if err := c.Start(ctx); err != nil {
			return nil, err
		}
		c.mu.Lock()
	}
	c.running = true
	c.mu.Unlock()

	events := make(chan agent.Event, 100)

	// Send message to stdin
	input := map[string]interface{}{
		"type":    "user_message",
		"content": message,
	}
	data, _ := json.Marshal(input)

	c.mu.Lock()
	_, err := c.stdin.Write(append(data, '\n'))
	c.mu.Unlock()

	if err != nil {
		close(events)
		return nil, fmt.Errorf("write stdin: %w", err)
	}

	// Read events from stdout
	go c.readEvents(ctx, events)

	return events, nil
}

func (c *Claude) readEvents(ctx context.Context, events chan<- agent.Event) {
	defer func() {
		c.mu.Lock()
		c.running = false
		c.mu.Unlock()
		close(events)
	}()

	scanner := bufio.NewScanner(c.stdout)
	// Increase buffer size for large outputs
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		event := c.parseEvent(line)
		if event != nil {
			events <- *event

			// Handle permission requests and questions
			if event.Type == agent.EventTypePermissionRequest || event.Type == agent.EventTypeAskUserQuestion {
				// Wait for response
				c.waitForResponse(ctx, event)
			}

			if event.Type == agent.EventTypeDone || event.Type == agent.EventTypeError {
				return
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Scanner error: %v", err)
		events <- agent.Event{
			Type:  agent.EventTypeError,
			Error: err.Error(),
		}
	}
}

func (c *Claude) waitForResponse(ctx context.Context, event *agent.Event) {
	select {
	case resp := <-c.pendingResponses:
		var input map[string]interface{}
		switch resp.Type {
		case "permission":
			data := resp.Data.(map[string]interface{})
			input = map[string]interface{}{
				"type":          "permission_response",
				"permission_id": data["permission_id"],
				"allowed":       data["allowed"],
			}
		case "question":
			data := resp.Data.(map[string]interface{})
			input = map[string]interface{}{
				"type":        "question_response",
				"question_id": data["question_id"],
				"answer":      data["answer"],
			}
		}
		if input != nil {
			data, _ := json.Marshal(input)
			c.mu.Lock()
			c.stdin.Write(append(data, '\n'))
			c.mu.Unlock()
		}
	case <-ctx.Done():
		return
	case <-time.After(5 * time.Minute):
		// Timeout waiting for response
		log.Printf("Timeout waiting for response")
		return
	}
}

func (c *Claude) parseEvent(data []byte) *agent.Event {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		log.Printf("Failed to parse event: %v", err)
		return nil
	}

	eventType, _ := raw["type"].(string)

	switch eventType {
	case "assistant":
		// Text output
		if msg, ok := raw["message"].(map[string]interface{}); ok {
			if content, ok := msg["content"].([]interface{}); ok {
				for _, c := range content {
					if block, ok := c.(map[string]interface{}); ok {
						if block["type"] == "text" {
							return &agent.Event{
								Type:    agent.EventTypeText,
								Content: block["text"].(string),
							}
						}
					}
				}
			}
		}

	case "content_block_start":
		if cb, ok := raw["content_block"].(map[string]interface{}); ok {
			if cb["type"] == "tool_use" {
				return &agent.Event{
					Type:      agent.EventTypeToolCall,
					ToolUseID: getString(cb, "id"),
					ToolName:  getString(cb, "name"),
				}
			}
		}

	case "content_block_delta":
		if delta, ok := raw["delta"].(map[string]interface{}); ok {
			if delta["type"] == "text_delta" {
				return &agent.Event{
					Type:    agent.EventTypeText,
					Content: getString(delta, "text"),
				}
			}
		}

	case "tool_result":
		return &agent.Event{
			Type:       agent.EventTypeToolResult,
			ToolUseID:  getString(raw, "tool_use_id"),
			ToolOutput: getString(raw, "content"),
		}

	case "result":
		return &agent.Event{Type: agent.EventTypeDone}

	case "permission_request":
		return &agent.Event{
			Type:         agent.EventTypePermissionRequest,
			PermissionID: getString(raw, "permission_id"),
			ToolName:     getString(raw, "tool_name"),
			Content:      getString(raw, "description"),
		}

	case "ask_user_question":
		options := []agent.QuestionOption{}
		if opts, ok := raw["options"].([]interface{}); ok {
			for _, o := range opts {
				if opt, ok := o.(map[string]interface{}); ok {
					options = append(options, agent.QuestionOption{
						Label:       getString(opt, "label"),
						Description: getString(opt, "description"),
					})
				}
			}
		}
		return &agent.Event{
			Type:       agent.EventTypeAskUserQuestion,
			QuestionID: getString(raw, "question_id"),
			Question:   getString(raw, "question"),
			Options:    options,
		}

	case "system":
		return &agent.Event{
			Type:    agent.EventTypeSystem,
			Content: getString(raw, "message"),
		}
	}

	return nil
}

// Interrupt interrupts the current processing
func (c *Claude) Interrupt(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stdin == nil {
		return nil
	}

	input := map[string]string{"type": "interrupt"}
	data, _ := json.Marshal(input)
	_, err := c.stdin.Write(append(data, '\n'))
	return err
}

// RespondToPermission responds to a permission request
func (c *Claude) RespondToPermission(ctx context.Context, permissionID string, allowed bool) error {
	c.pendingResponses <- pendingResponse{
		Type: "permission",
		Data: map[string]interface{}{
			"permission_id": permissionID,
			"allowed":       allowed,
		},
	}
	return nil
}

// RespondToQuestion responds to a user question
func (c *Claude) RespondToQuestion(ctx context.Context, questionID string, answer string) error {
	c.pendingResponses <- pendingResponse{
		Type: "question",
		Data: map[string]interface{}{
			"question_id": questionID,
			"answer":      answer,
		},
	}
	return nil
}

// IsRunning returns true if the agent is currently processing
func (c *Claude) IsRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running
}

// Close terminates the Claude CLI process
func (c *Claude) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancelFunc != nil {
		c.cancelFunc()
	}

	if c.cmd != nil && c.cmd.Process != nil {
		return c.cmd.Process.Kill()
	}
	return nil
}

// Helper function
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
