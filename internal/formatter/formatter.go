package formatter

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/riaanpieterse81/ClaudeToGo/internal/types"
)

// MessengerFormatter handles formatting extracted data for messenger consumption
type MessengerFormatter struct{}

// NewMessengerFormatter creates a new messenger formatter
func NewMessengerFormatter() *MessengerFormatter {
	return &MessengerFormatter{}
}

// FormatForMessenger converts extracted data into a messenger-friendly format
func (mf *MessengerFormatter) FormatForMessenger(data *types.ExtractedData) (*types.MessengerMessage, error) {
	switch data.EventType {
	case "stop":
		return mf.formatStopEvent(data)
	case "notification":
		return mf.formatNotificationEvent(data)
	default:
		return nil, fmt.Errorf("unknown event type: %s", data.EventType)
	}
}

// formatStopEvent formats a Stop event for messenger
func (mf *MessengerFormatter) formatStopEvent(data *types.ExtractedData) (*types.MessengerMessage, error) {
	stopData, ok := data.Data.(*types.StopEventData)
	if !ok {
		return nil, fmt.Errorf("invalid stop event data type")
	}

	// Create base message
	message := &types.MessengerMessage{
		Type:      "completion",
		SessionID: data.SessionID,
		Timestamp: data.Timestamp,
		Context:   make(map[string]interface{}),
	}

	// Set title based on task status
	switch stopData.TaskStatus {
	case "completed":
		message.Title = "✅ Task Completed"
		message.Priority = "medium"
	case "error":
		message.Title = "❌ Task Failed"
		message.Priority = "high"
	case "cancelled":
		message.Title = "⏹️ Task Cancelled"
		message.Priority = "low"
	default:
		message.Title = "🔄 Task Finished"
		message.Priority = "medium"
	}

	// Set the main message
	message.Message = mf.formatStopMessage(stopData)

	// Add context information
	message.Context["cwd"] = data.CWD
	message.Context["task_status"] = stopData.TaskStatus
	message.Context["session_id"] = data.SessionID
	if stopData.Summary != "" {
		message.Context["summary"] = stopData.Summary
	}

	// Add suggested actions for completed tasks
	if stopData.TaskStatus == "completed" {
		message.Actions = mf.createStopEventActions(data)
	} else if stopData.TaskStatus == "error" {
		message.Actions = mf.createErrorEventActions(data)
	}

	return message, nil
}

// formatNotificationEvent formats a Notification event for messenger
func (mf *MessengerFormatter) formatNotificationEvent(data *types.ExtractedData) (*types.MessengerMessage, error) {
	notificationData, ok := data.Data.(*types.NotificationEventData)
	if !ok {
		return nil, fmt.Errorf("invalid notification event data type")
	}

	// Create base message
	message := &types.MessengerMessage{
		Type:      "action_needed",
		SessionID: data.SessionID,
		Timestamp: data.Timestamp,
		Priority:  "high",
		Context:   make(map[string]interface{}),
	}

	// Set title and message based on tool type
	message.Title = mf.getNotificationTitle(notificationData)
	message.Message = mf.formatNotificationMessage(notificationData)

	// Add context information
	message.Context["cwd"] = data.CWD
	message.Context["tool_name"] = notificationData.ToolName
	message.Context["action"] = notificationData.Action
	message.Context["session_id"] = data.SessionID

	// Copy tool details to context
	for key, value := range notificationData.Details {
		message.Context[key] = value
	}

	// Create suggested actions
	message.Actions = mf.createNotificationActions(notificationData, data)

	return message, nil
}

// formatStopMessage creates a user-friendly message for stop events
func (mf *MessengerFormatter) formatStopMessage(data *types.StopEventData) string {
	if data.FinalMessage == "" {
		return "Claude has completed the task."
	}

	// Clean up the message
	message := strings.TrimSpace(data.FinalMessage)
	
	// Add status context if needed
	switch data.TaskStatus {
	case "error":
		if !strings.Contains(strings.ToLower(message), "error") {
			message = "Error: " + message
		}
	case "cancelled":
		if !strings.Contains(strings.ToLower(message), "cancel") {
			message = "Cancelled: " + message
		}
	}

	return message
}

// formatNotificationMessage creates a user-friendly message for notification events
func (mf *MessengerFormatter) formatNotificationMessage(data *types.NotificationEventData) string {
	baseMessage := fmt.Sprintf("Claude wants to %s", mf.getActionDescription(data))
	
	// Add specific details based on tool type
	switch strings.ToLower(data.ToolName) {
	case "write":
		if filePath, exists := data.Details["target_file"]; exists {
			fileName := filepath.Base(fmt.Sprintf("%v", filePath))
			baseMessage = fmt.Sprintf("Claude wants to create file: %s", fileName)
			if preview, exists := data.Details["content_preview"]; exists {
				baseMessage += fmt.Sprintf("\n\nContent preview:\n%v", preview)
			}
		}
	case "edit":
		if filePath, exists := data.Details["target_file"]; exists {
			fileName := filepath.Base(fmt.Sprintf("%v", filePath))
			baseMessage = fmt.Sprintf("Claude wants to edit file: %s", fileName)
		}
	case "read":
		if filePath, exists := data.Details["target_file"]; exists {
			fileName := filepath.Base(fmt.Sprintf("%v", filePath))
			baseMessage = fmt.Sprintf("Claude wants to read file: %s", fileName)
		}
	case "webfetch", "fetch":
		if url, exists := data.Details["target_url"]; exists {
			baseMessage = fmt.Sprintf("Claude wants to fetch: %v", url)
		}
	case "bash":
		if command, exists := data.Details["command"]; exists {
			baseMessage = fmt.Sprintf("Claude wants to run: %v", command)
		}
	case "list", "ls":
		if path, exists := data.Details["target_path"]; exists {
			baseMessage = fmt.Sprintf("Claude wants to list directory: %v", path)
		}
	}

	return baseMessage
}

// getNotificationTitle creates a title for notification events
func (mf *MessengerFormatter) getNotificationTitle(data *types.NotificationEventData) string {
	switch strings.ToLower(data.ToolName) {
	case "write":
		return "📝 File Creation Request"
	case "edit":
		return "✏️ File Edit Request"
	case "read":
		return "👀 File Read Request"
	case "webfetch", "fetch":
		return "🌐 Web Fetch Request"
	case "bash":
		return "⚡ Command Execution Request"
	case "list", "ls":
		return "📂 Directory List Request"
	default:
		return fmt.Sprintf("🔧 %s Tool Request", data.ToolName)
	}
}

// getActionDescription returns a human-friendly description of the action
func (mf *MessengerFormatter) getActionDescription(data *types.NotificationEventData) string {
	switch data.Action {
	case "create_file":
		return "create a new file"
	case "edit_file":
		return "edit an existing file"
	case "read_file":
		return "read a file"
	case "fetch_url":
		return "fetch content from a URL"
	case "execute_command":
		return "execute a command"
	case "list_directory":
		return "list directory contents"
	default:
		return fmt.Sprintf("use the %s tool", data.ToolName)
	}
}

// createStopEventActions creates suggested actions for completed tasks
func (mf *MessengerFormatter) createStopEventActions(data *types.ExtractedData) []types.SuggestedAction {
	actions := []types.SuggestedAction{
		{
			Type:        "info",
			Label:       "ℹ️ View Details",
			Command:     fmt.Sprintf("claudetogo status --session %s", data.SessionID),
			Description: "View full session details",
			Icon:        "ℹ️",
		},
	}

	return actions
}

// createErrorEventActions creates suggested actions for failed tasks
func (mf *MessengerFormatter) createErrorEventActions(data *types.ExtractedData) []types.SuggestedAction {
	actions := []types.SuggestedAction{
		{
			Type:        "info",
			Label:       "🔍 Debug",
			Command:     fmt.Sprintf("claudetogo debug --session %s", data.SessionID),
			Description: "Get debug information about the error",
			Icon:        "🔍",
		},
		{
			Type:        "info",
			Label:       "📋 View Log",
			Command:     fmt.Sprintf("claudetogo log --session %s", data.SessionID),
			Description: "View the full session log",
			Icon:        "📋",
		},
	}

	return actions
}

// createNotificationActions creates suggested actions for notification events
func (mf *MessengerFormatter) createNotificationActions(notificationData *types.NotificationEventData, extractedData *types.ExtractedData) []types.SuggestedAction {
	sessionID := extractedData.SessionID
	
	baseActions := []types.SuggestedAction{
		{
			Type:        "approve",
			Label:       "✅ Approve",
			Command:     fmt.Sprintf("claudetogo respond --session %s --action approve", sessionID),
			Description: fmt.Sprintf("Allow Claude to %s", mf.getActionDescription(notificationData)),
			Icon:        "✅",
		},
		{
			Type:        "reject",
			Label:       "❌ Reject",
			Command:     fmt.Sprintf("claudetogo respond --session %s --action reject", sessionID),
			Description: fmt.Sprintf("Deny the %s request", mf.getActionDescription(notificationData)),
			Icon:        "❌",
		},
	}

	// Add tool-specific actions
	switch strings.ToLower(notificationData.ToolName) {
	case "write", "edit":
		baseActions = append(baseActions, types.SuggestedAction{
			Type:        "modify",
			Label:       "✏️ Review File",
			Command:     mf.getFileReviewCommand(notificationData),
			Description: "Review the file before approving",
			Icon:        "✏️",
		})
	case "bash":
		baseActions = append(baseActions, types.SuggestedAction{
			Type:        "info",
			Label:       "ℹ️ Command Info",
			Command:     mf.getCommandInfoCommand(notificationData),
			Description: "Get more information about this command",
			Icon:        "ℹ️",
		})
	}

	// Add general info action
	baseActions = append(baseActions, types.SuggestedAction{
		Type:        "info",
		Label:       "📖 More Info",
		Command:     fmt.Sprintf("claudetogo info --session %s", sessionID),
		Description: "Get more details about this request",
		Icon:        "📖",
	})

	return baseActions
}

// getFileReviewCommand creates a command to review a file
func (mf *MessengerFormatter) getFileReviewCommand(data *types.NotificationEventData) string {
	if filePath, exists := data.Details["target_file"]; exists {
		return fmt.Sprintf("cat \"%v\"", filePath)
	}
	return "echo 'No file specified'"
}

// getCommandInfoCommand creates a command to get info about a bash command
func (mf *MessengerFormatter) getCommandInfoCommand(data *types.NotificationEventData) string {
	if command, exists := data.Details["command"]; exists {
		// Extract the base command (first word)
		cmdStr := fmt.Sprintf("%v", command)
		parts := strings.Fields(cmdStr)
		if len(parts) > 0 {
			return fmt.Sprintf("man %s", parts[0])
		}
	}
	return "echo 'No command specified'"
}

// CreateActionableMessage creates a message with enhanced actionability
func (mf *MessengerFormatter) CreateActionableMessage(data *types.ExtractedData) (*types.MessengerMessage, error) {
	message, err := mf.FormatForMessenger(data)
	if err != nil {
		return nil, err
	}

	// Enhance with additional context
	message.Context["formatted_at"] = data.Timestamp
	message.Context["cwd_basename"] = filepath.Base(data.CWD)
	
	// Add quick action hints
	if data.EventType == "notification" {
		message.Context["quick_approve"] = fmt.Sprintf("claudetogo respond --session %s --action approve", data.SessionID)
		message.Context["quick_reject"] = fmt.Sprintf("claudetogo respond --session %s --action reject", data.SessionID)
	}

	return message, nil
}