package hooks

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/riaanpieterse81/ClaudeToGo/internal/logger"
	"github.com/riaanpieterse81/ClaudeToGo/internal/types"
)

// Validate validates the required fields of a hook event
func Validate(event *types.ClaudeHookEvent) error {
	if event == nil {
		return errors.New("hook event is nil")
	}
	if strings.TrimSpace(event.SessionID) == "" {
		return errors.New("session_id is required")
	}
	if strings.TrimSpace(event.HookEventName) == "" {
		return errors.New("hook_event_name is required")
	}
	return nil
}

// ensureLogDirectory creates the log directory if it doesn't exist
func ensureLogDirectory(logFile string) error {
	dir := filepath.Dir(logFile)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}
	}
	return nil
}

// SaveEvent safely saves a hook event to the log file
func SaveEvent(event types.ClaudeHookEvent, config types.Config, logger *logger.Logger) error {
	if err := Validate(&event); err != nil {
		return fmt.Errorf("invalid hook event: %w", err)
	}

	if err := ensureLogDirectory(config.LogFile); err != nil {
		return err
	}

	file, err := os.OpenFile(config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(event); err != nil {
		return fmt.Errorf("failed to encode event: %w", err)
	}

	logger.Debug("Saved event: %s (Session: %s)", event.HookEventName, event.SessionID)
	return nil
}

// ProcessEvent processes a hook event and returns appropriate response
func ProcessEvent(event types.ClaudeHookEvent, logger *logger.Logger) types.ClaudeHookResponse {
	logger.Debug("Processing hook event: %s", event.HookEventName)

	// Always allow - this is stage 1: log everything
	continueVal := true
	return types.ClaudeHookResponse{
		Continue: &continueVal,
		Decision: "approve",
	}
}

// SendResponse sends the response back to Claude Code via stdout
func SendResponse(response types.ClaudeHookResponse, logger *logger.Logger) error {
	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(response); err != nil {
		return fmt.Errorf("failed to encode response: %w", err)
	}

	logger.Debug("Sent response: %s", response.Decision)
	return nil
}

// ProcessFromStdin reads and processes a hook event from stdin
func ProcessFromStdin(config types.Config, logger *logger.Logger) error {
	var event types.ClaudeHookEvent
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&event); err != nil {
		return fmt.Errorf("failed to decode hook event from stdin: %w", err)
	}

	if err := SaveEvent(event, config, logger); err != nil {
		return fmt.Errorf("failed to save hook event: %w", err)
	}

	// Process the event and generate response
	response := ProcessEvent(event, logger)

	// Send response back to Claude
	if err := SendResponse(response, logger); err != nil {
		return fmt.Errorf("failed to send hook response: %w", err)
	}

	logger.Info("Hook event processed successfully")
	return nil
}