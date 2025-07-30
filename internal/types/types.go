package types

import (
	"encoding/json"
	"time"
)

// ClaudeHookEvent represents the JSON data received from Claude Code hooks
type ClaudeHookEvent struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	CWD            string `json:"cwd"`
	HookEventName  string `json:"hook_event_name"`
	ToolName       string `json:"tool_name,omitempty"`
	Timestamp      string `json:"timestamp"`
	Message        string `json:"message,omitempty"`
}

// ClaudeHookResponse represents the response sent back to Claude Code
type ClaudeHookResponse struct {
	Continue *bool  `json:"continue,omitempty"`
	Decision string `json:"decision,omitempty"`
}

// Config holds application configuration
type Config struct {
	LogFile      string
	PollInterval time.Duration
	Verbose      bool
}

// ConfigFile represents the configuration file structure
type ConfigFile struct {
	LogFile      string `json:"logFile"`
	PollInterval string `json:"pollInterval"`
	Verbose      bool   `json:"verbose"`
}

// ClaudeSettingsConfig represents the Claude Code settings.json structure
type ClaudeSettingsConfig struct {
	Hooks map[string][]HookMatcher `json:"hooks,omitempty"`
	// Preserve all other unknown fields in the settings.json
	Extra map[string]json.RawMessage `json:"-"`
}

// HookMatcher represents a hook matcher configuration
type HookMatcher struct {
	Matcher string       `json:"matcher"`
	Hooks   []HookConfig `json:"hooks"`
}

// HookConfig represents a hook configuration
type HookConfig struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout *int   `json:"timeout,omitempty"`
}

// ConfigLocation represents a configuration location choice
type ConfigLocation struct {
	Path        string
	Description string
	Scope       string // "global", "project", "local"
}

// Transcript processing types

// TranscriptMessage represents a single message in the Claude Code transcript JSONL file
type TranscriptMessage struct {
	ParentUUID    string        `json:"parentUuid"`
	IsSidechain   bool          `json:"isSidechain"`
	UserType      string        `json:"userType"`
	CWD           string        `json:"cwd"`
	SessionID     string        `json:"sessionId"`
	Version       string        `json:"version"`
	GitBranch     string        `json:"gitBranch"`
	Type          string        `json:"type"` // "user" or "assistant"
	Message       ClaudeMessage `json:"message"`
	UUID          string        `json:"uuid"`
	Timestamp     string        `json:"timestamp"`
	RequestID     string        `json:"requestId,omitempty"`
	IsMeta        bool          `json:"isMeta,omitempty"`
	ToolUseResult interface{}   `json:"toolUseResult,omitempty"`
}

// ClaudeMessage represents the message content within a transcript message
type ClaudeMessage struct {
	ID           string      `json:"id,omitempty"`
	Type         string      `json:"type,omitempty"`
	Role         string      `json:"role"`
	Model        string      `json:"model,omitempty"`
	Content      interface{} `json:"content"` // Can be string or []ContentItem
	StopReason   string      `json:"stop_reason,omitempty"`
	StopSequence string      `json:"stop_sequence,omitempty"`
	Usage        *Usage      `json:"usage,omitempty"`
}

// ContentItem represents individual content items (text, tool_use, tool_result, etc.)
type ContentItem struct {
	Type      string                 `json:"type"` // "text", "tool_use", "tool_result"
	Text      string                 `json:"text,omitempty"`
	ID        string                 `json:"id,omitempty"`
	Name      string                 `json:"name,omitempty"`
	Input     map[string]interface{} `json:"input,omitempty"`
	Content   interface{}            `json:"content,omitempty"`
	ToolUseID string                 `json:"tool_use_id,omitempty"`
	IsError   bool                   `json:"is_error,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	InputTokens              int    `json:"input_tokens"`
	CacheCreationInputTokens int    `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int    `json:"cache_read_input_tokens"`
	OutputTokens             int    `json:"output_tokens"`
	ServiceTier              string `json:"service_tier"`
}

// Data extraction types

// ExtractedData represents the output of the data extraction process
type ExtractedData struct {
	EventType string      `json:"event_type"` // "stop" or "notification"
	SessionID string      `json:"session_id"`
	CWD       string      `json:"cwd"`
	Timestamp string      `json:"timestamp"`
	Data      interface{} `json:"data"` // StopEventData or NotificationEventData
}

// StopEventData represents data extracted from Stop events
type StopEventData struct {
	FinalMessage string `json:"final_message"`
	Summary      string `json:"summary,omitempty"`
	TaskStatus   string `json:"task_status"` // "completed", "error", "cancelled"
}

// NotificationEventData represents data extracted from Notification events
type NotificationEventData struct {
	ToolName    string                 `json:"tool_name"`
	Action      string                 `json:"action"`
	Details     map[string]interface{} `json:"details"`
	RequestText string                 `json:"request_text,omitempty"`
}

// Messenger formatting types

// MessengerMessage represents the final formatted message for messenger apps
type MessengerMessage struct {
	Type        string                 `json:"type"`          // "completion" or "action_needed"
	SessionID   string                 `json:"session_id"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Actions     []SuggestedAction      `json:"actions,omitempty"`
	Context     map[string]interface{} `json:"context"`
	Timestamp   string                 `json:"timestamp"`
	Priority    string                 `json:"priority,omitempty"` // "high", "medium", "low"
}

// SuggestedAction represents actions a user can take via messenger
type SuggestedAction struct {
	Type        string `json:"type"`        // "approve", "modify", "reject", "info"
	Label       string `json:"label"`       // User-friendly text
	Command     string `json:"command"`     // Command to execute
	Description string `json:"description"` // What this action does
	Icon        string `json:"icon,omitempty"` // Emoji or icon identifier
}