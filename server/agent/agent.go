package agent

import "context"

// EventType represents the type of event from the AI agent
type EventType string

const (
	EventTypeText              EventType = "text"
	EventTypeToolCall          EventType = "tool_call"
	EventTypeToolResult        EventType = "tool_result"
	EventTypeError             EventType = "error"
	EventTypeDone              EventType = "done"
	EventTypePermissionRequest EventType = "permission_request"
	EventTypeAskUserQuestion   EventType = "ask_user_question"
	EventTypeSystem            EventType = "system"
	EventTypeInterrupted       EventType = "interrupted"
)

// Event represents an event from the AI agent
type Event struct {
	Type         EventType              `json:"type"`
	Content      string                 `json:"content,omitempty"`
	ToolUseID    string                 `json:"tool_use_id,omitempty"`
	ToolName     string                 `json:"tool_name,omitempty"`
	ToolInput    map[string]interface{} `json:"tool_input,omitempty"`
	ToolOutput   string                 `json:"tool_output,omitempty"`
	PermissionID string                 `json:"permission_id,omitempty"`
	QuestionID   string                 `json:"question_id,omitempty"`
	Question     string                 `json:"question,omitempty"`
	Options      []QuestionOption       `json:"options,omitempty"`
	Error        string                 `json:"error,omitempty"`
}

// QuestionOption represents an option for user questions
type QuestionOption struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// Agent is the interface for AI agents
type Agent interface {
	// SendMessage sends a message to the agent and returns a channel of events
	SendMessage(ctx context.Context, message string) (<-chan Event, error)

	// Interrupt interrupts the current processing
	Interrupt(ctx context.Context) error

	// RespondToPermission responds to a permission request
	RespondToPermission(ctx context.Context, permissionID string, allowed bool) error

	// RespondToQuestion responds to a user question
	RespondToQuestion(ctx context.Context, questionID string, answer string) error

	// IsRunning returns true if the agent is currently processing
	IsRunning() bool

	// Close terminates the agent process
	Close() error
}
