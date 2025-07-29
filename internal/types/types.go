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