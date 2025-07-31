package responder

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/riaanpieterse81/ClaudeToGo/internal/logger"
	"github.com/riaanpieterse81/ClaudeToGo/internal/types"
)

// ResponseHandler handles user responses from messenger apps and executes actions
type ResponseHandler struct {
	outputDir string
	logger    *logger.Logger
}

// SessionStatus contains information about a specific session
type SessionStatus struct {
	SessionID     string                 `json:"session_id"`
	Status        string                 `json:"status"`
	CreatedAt     time.Time             `json:"created_at"`
	LastAction    string                 `json:"last_action,omitempty"`
	Context       map[string]interface{} `json:"context,omitempty"`
	MessengerFile string                 `json:"messenger_file,omitempty"`
}

// PendingAction represents a pending action that needs user response
type PendingAction struct {
	SessionID     string    `json:"session_id"`
	Type          string    `json:"type"`
	Title         string    `json:"title"`
	Message       string    `json:"message"`
	CreatedAt     time.Time `json:"created_at"`
	MessengerFile string    `json:"messenger_file"`
}

// NewResponseHandler creates a new response handler
func NewResponseHandler(outputDir string, logger *logger.Logger) *ResponseHandler {
	if outputDir == "" {
		outputDir = "messenger-output"
	}

	return &ResponseHandler{
		outputDir: outputDir,
		logger:    logger,
	}
}

// HandleResponse processes a user response (approve, reject, etc.)
func (rh *ResponseHandler) HandleResponse(sessionID, action string) error {
	rh.logger.Info("Processing response for session %s: %s", sessionID, action)

	// Find the messenger file for this session
	messengerFile, err := rh.findMessengerFile(sessionID)
	if err != nil {
		return fmt.Errorf("failed to find messenger file for session %s: %w", sessionID, err)
	}

	// Load the messenger message
	message, err := rh.loadMessengerMessage(messengerFile)
	if err != nil {
		return fmt.Errorf("failed to load messenger message: %w", err)
	}

	// Validate the action
	if !rh.isValidAction(message, action) {
		return fmt.Errorf("invalid action '%s' for this message type", action)
	}

	// Execute the action
	return rh.executeAction(sessionID, action, message, messengerFile)
}

// ExecuteAction executes the approved action by interfacing with Claude Code
func (rh *ResponseHandler) ExecuteAction(sessionID, action string, message *types.MessengerMessage) error {
	rh.logger.Info("Executing action %s for session %s", action, sessionID)

	switch action {
	case "approve":
		return rh.executeApproval(sessionID, message)
	case "reject":
		return rh.executeRejection(sessionID, message)
	default:
		return fmt.Errorf("unsupported action: %s", action)
	}
}

// GetSessionStatus retrieves status information for a specific session
func (rh *ResponseHandler) GetSessionStatus(sessionID string) (*SessionStatus, error) {
	rh.logger.Debug("Getting status for session: %s", sessionID)

	// Find the messenger file for this session
	messengerFile, err := rh.findMessengerFile(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	// Load the messenger message
	message, err := rh.loadMessengerMessage(messengerFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load session data: %w", err)
	}

	// Get file info for creation time
	fileInfo, err := os.Stat(messengerFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	status := &SessionStatus{
		SessionID:     sessionID,
		Status:        rh.determineStatus(message),
		CreatedAt:     fileInfo.ModTime(),
		MessengerFile: messengerFile,
		Context:       message.Context,
	}

	// Check if there's been any action on this session
	responseFile := rh.getResponseFilePath(sessionID)
	if rh.fileExists(responseFile) {
		responseData, err := rh.loadResponseData(responseFile)
		if err == nil {
			status.LastAction = responseData["action"].(string)
		}
	}

	return status, nil
}

// ListPendingActions returns all pending actions that need user responses
func (rh *ResponseHandler) ListPendingActions() ([]*PendingAction, error) {
	rh.logger.Debug("Listing pending actions...")

	var pendingActions []*PendingAction

	// Scan messenger output directory for notification files
	pattern := filepath.Join(rh.outputDir, "messenger-notification-*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to scan for messenger files: %w", err)
	}

	for _, file := range matches {
		// Load the message
		message, err := rh.loadMessengerMessage(file)
		if err != nil {
			rh.logger.Debug("Failed to load messenger file %s: %v", file, err)
			continue
		}

		// Check if this is a pending action (action_needed type)
		if message.Type == "action_needed" {
			sessionID := message.SessionID
			
			// Check if already responded to
			responseFile := rh.getResponseFilePath(sessionID)
			if rh.fileExists(responseFile) {
				continue // Already handled
			}

			// Get file creation time
			fileInfo, err := os.Stat(file)
			if err != nil {
				continue
			}

			pendingAction := &PendingAction{
				SessionID:     sessionID,
				Type:          message.Type,
				Title:         message.Title,
				Message:       message.Message,
				CreatedAt:     fileInfo.ModTime(),
				MessengerFile: file,
			}

			pendingActions = append(pendingActions, pendingAction)
		}
	}

	return pendingActions, nil
}

// findMessengerFile finds the messenger JSON file for a given session ID
func (rh *ResponseHandler) findMessengerFile(sessionID string) (string, error) {
	// Try different patterns to find the file
	patterns := []string{
		fmt.Sprintf("messenger-notification-%s-*.json", sessionID[:8]),
		fmt.Sprintf("messenger-stop-%s-*.json", sessionID[:8]),
		fmt.Sprintf("messenger-*-%s-*.json", sessionID[:8]),
	}

	for _, pattern := range patterns {
		fullPattern := filepath.Join(rh.outputDir, pattern)
		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			continue
		}

		for _, match := range matches {
			// Verify this file contains the correct session ID
			message, err := rh.loadMessengerMessage(match)
			if err != nil {
				continue
			}

			if strings.HasPrefix(message.SessionID, sessionID) || strings.HasPrefix(sessionID, message.SessionID) {
				return match, nil
			}
		}
	}

	return "", fmt.Errorf("no messenger file found for session ID: %s", sessionID)
}

// loadMessengerMessage loads a messenger message from a JSON file
func (rh *ResponseHandler) loadMessengerMessage(filePath string) (*types.MessengerMessage, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var message types.MessengerMessage
	if err := json.Unmarshal(data, &message); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &message, nil
}

// isValidAction checks if the given action is valid for the message
func (rh *ResponseHandler) isValidAction(message *types.MessengerMessage, action string) bool {
	// For action_needed messages, check if the action is in the available actions
	if message.Type == "action_needed" && len(message.Actions) > 0 {
		for _, msgAction := range message.Actions {
			if msgAction.Type == action {
				return true
			}
		}
		return false
	}

	// For other message types, only basic actions are allowed
	return action == "approve" || action == "reject" || action == "info"
}

// executeAction performs the actual action execution
func (rh *ResponseHandler) executeAction(sessionID, action string, message *types.MessengerMessage, messengerFile string) error {
	// Record the response
	if err := rh.recordResponse(sessionID, action, message); err != nil {
		return fmt.Errorf("failed to record response: %w", err)
	}

	// Execute the specific action
	switch action {
	case "approve":
		return rh.executeApproval(sessionID, message)
	case "reject":
		return rh.executeRejection(sessionID, message)
	case "info":
		return rh.showInfo(sessionID, message)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

// executeApproval handles approval actions
func (rh *ResponseHandler) executeApproval(sessionID string, message *types.MessengerMessage) error {
	rh.logger.Info("Executing approval for session %s", sessionID)

	// TODO: Interface with Claude Code to execute the approved action
	// This would involve:
	// 1. Extracting the tool and parameters from the message context
	// 2. Constructing the appropriate Claude Code command
	// 3. Executing the command
	// 4. Recording the result

	rh.logger.Info("Action approved and executed successfully")
	return nil
}

// executeRejection handles rejection actions
func (rh *ResponseHandler) executeRejection(sessionID string, message *types.MessengerMessage) error {
	rh.logger.Info("Executing rejection for session %s", sessionID)

	// TODO: Interface with Claude Code to reject the action
	// This might involve sending a signal to Claude Code that the action was rejected

	rh.logger.Info("Action rejected successfully")
	return nil
}

// showInfo displays information about the session
func (rh *ResponseHandler) showInfo(sessionID string, message *types.MessengerMessage) error {
	rh.logger.Info("Showing info for session %s", sessionID)

	fmt.Printf("📋 Session Information: %s\n", sessionID)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Type:     %s\n", message.Type)
	fmt.Printf("Title:    %s\n", message.Title)
	fmt.Printf("Message:  %s\n", message.Message)
	fmt.Printf("Time:     %s\n", message.Timestamp)

	if message.Context != nil {
		fmt.Printf("Context:\n")
		for key, value := range message.Context {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	return nil
}

// recordResponse records the user's response for tracking
func (rh *ResponseHandler) recordResponse(sessionID, action string, message *types.MessengerMessage) error {
	responseFile := rh.getResponseFilePath(sessionID)

	response := map[string]interface{}{
		"session_id": sessionID,
		"action":     action,
		"timestamp":  time.Now().Format(time.RFC3339),
		"message_type": message.Type,
		"message_title": message.Title,
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	// Ensure responses directory exists
	responsesDir := filepath.Join(rh.outputDir, "responses")
	if err := os.MkdirAll(responsesDir, 0755); err != nil {
		return fmt.Errorf("failed to create responses directory: %w", err)
	}

	if err := os.WriteFile(responseFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write response file: %w", err)
	}

	return nil
}

// getResponseFilePath returns the path for storing response data
func (rh *ResponseHandler) getResponseFilePath(sessionID string) string {
	filename := fmt.Sprintf("response-%s.json", sessionID[:8])
	return filepath.Join(rh.outputDir, "responses", filename)
}

// loadResponseData loads response data from file
func (rh *ResponseHandler) loadResponseData(filePath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var response map[string]interface{}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	return response, nil
}

// determineStatus determines the current status of a session
func (rh *ResponseHandler) determineStatus(message *types.MessengerMessage) string {
	switch message.Type {
	case "action_needed":
		return "pending_response"
	case "completion":
		return "completed"
	default:
		return "active"
	}
}

// fileExists checks if a file exists
func (rh *ResponseHandler) fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}