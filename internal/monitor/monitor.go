package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/riaanpieterse81/ClaudeToGo/internal/logger"
	"github.com/riaanpieterse81/ClaudeToGo/internal/types"
)

// formatEventOutput formats an event for display
func formatEventOutput(event types.ClaudeHookEvent) string {
	timestamp := time.Now().Format("15:04:05")
	sessionID := event.SessionID
	if len(sessionID) > 8 {
		sessionID = sessionID[:8]
	}

	toolInfo := ""
	if event.ToolName != "" {
		toolInfo = fmt.Sprintf(" | Tool: %s", event.ToolName)
	}

	return fmt.Sprintf("[%s] 🎯 %s | Session: %s%s",
		timestamp, event.HookEventName, sessionID, toolInfo)
}

// checkForNewEvents checks for and processes new events in the log file
func checkForNewEvents(logFile string, lastSize *int64, logger *logger.Logger) error {
	info, err := os.Stat(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug("Log file does not exist yet: %s", logFile)
			return nil
		}
		return fmt.Errorf("failed to stat log file: %w", err)
	}

	currentSize := info.Size()
	if currentSize <= *lastSize {
		return nil
	}

	logger.Debug("File size changed: %d -> %d", *lastSize, currentSize)

	file, err := os.Open(logFile)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	if _, err := file.Seek(*lastSize, 0); err != nil {
		return fmt.Errorf("failed to seek in log file: %w", err)
	}

	decoder := json.NewDecoder(file)
	for decoder.More() {
		var event types.ClaudeHookEvent
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			logger.Error("Failed to decode event: %v", err)
			continue
		}

		fmt.Println(formatEventOutput(event))
	}

	*lastSize = currentSize
	return nil
}

// Start monitors the log file for new events with graceful shutdown
func Start(ctx context.Context, config types.Config, logger *logger.Logger) error {
	logger.Info("Starting event monitor (Poll interval: %v)", config.PollInterval)

	var lastSize int64 = 0
	if info, err := os.Stat(config.LogFile); err == nil {
		lastSize = info.Size()
		logger.Debug("Initial file size: %d bytes", lastSize)
	}

	ticker := time.NewTicker(config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Monitor stopping...")
			return ctx.Err()
		case <-ticker.C:
			if err := checkForNewEvents(config.LogFile, &lastSize, logger); err != nil {
				logger.Error("Error checking for events: %v", err)
			}
		}
	}
}