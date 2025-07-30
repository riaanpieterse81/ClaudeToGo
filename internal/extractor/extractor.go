package extractor

import (
	"fmt"
	"strings"
	"time"

	"github.com/riaanpieterse81/ClaudeToGo/internal/transcript"
	"github.com/riaanpieterse81/ClaudeToGo/internal/types"
)

// DataExtractor handles extracting relevant data from Claude events and transcripts
type DataExtractor struct {
	transcriptReader *transcript.Reader
}

// NewDataExtractor creates a new data extractor
func NewDataExtractor() *DataExtractor {
	return &DataExtractor{
		transcriptReader: transcript.NewReader(),
	}
}

// ProcessEvent processes a Claude hook event and extracts relevant data
func (de *DataExtractor) ProcessEvent(event *types.ClaudeHookEvent) (*types.ExtractedData, error) {
	switch strings.ToLower(event.HookEventName) {
	case "stop":
		return de.ProcessStopEvent(event)
	case "notification":
		return de.ProcessNotificationEvent(event)
	default:
		return nil, fmt.Errorf("unknown hook event type: %s", event.HookEventName)
	}
}

// ProcessStopEvent processes a Stop event and extracts the final assistant message
func (de *DataExtractor) ProcessStopEvent(event *types.ClaudeHookEvent) (*types.ExtractedData, error) {
	// Get the last assistant message from the transcript
	lastAssistantMsg, err := de.transcriptReader.GetLastAssistantMessage(event.TranscriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get last assistant message: %w", err)
	}

	// Extract the text content
	finalMessage := de.transcriptReader.ExtractTextContent(lastAssistantMsg)
	if finalMessage == "" {
		finalMessage = "Task completed (no text response)"
	}

	// Determine task status based on content
	taskStatus := de.determineTaskStatus(finalMessage, lastAssistantMsg)

	// Create stop event data
	stopData := &types.StopEventData{
		FinalMessage: finalMessage,
		Summary:      de.generateSummary(finalMessage),
		TaskStatus:   taskStatus,
	}

	// Get timestamp (use event timestamp or current time)
	timestamp := event.Timestamp
	if timestamp == "" {
		timestamp = time.Now().Format(time.RFC3339)
	}

	return &types.ExtractedData{
		EventType: "stop",
		SessionID: event.SessionID,
		CWD:       event.CWD,
		Timestamp: timestamp,
		Data:      stopData,
	}, nil
}

// ProcessNotificationEvent processes a Notification event and extracts tool usage details
func (de *DataExtractor) ProcessNotificationEvent(event *types.ClaudeHookEvent) (*types.ExtractedData, error) {
	// Get the last tool use from the transcript
	lastToolUse, err := de.transcriptReader.GetLastToolUse(event.TranscriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get last tool use: %w", err)
	}

	// Extract tool use details
	toolUseContent, err := de.transcriptReader.ExtractToolUseDetails(lastToolUse)
	if err != nil {
		return nil, fmt.Errorf("failed to extract tool use details: %w", err)
	}

	// Process the tool use based on tool type
	notificationData, err := de.processToolUse(toolUseContent, event)
	if err != nil {
		return nil, fmt.Errorf("failed to process tool use: %w", err)
	}

	// Get timestamp (use event timestamp or current time)
	timestamp := event.Timestamp
	if timestamp == "" {
		timestamp = time.Now().Format(time.RFC3339)
	}

	return &types.ExtractedData{
		EventType: "notification",
		SessionID: event.SessionID,
		CWD:       event.CWD,
		Timestamp: timestamp,
		Data:      notificationData,
	}, nil
}

// processToolUse processes different types of tool usage
func (de *DataExtractor) processToolUse(toolUse *types.ContentItem, event *types.ClaudeHookEvent) (*types.NotificationEventData, error) {
	toolName := toolUse.Name
	if toolName == "" {
		toolName = "Unknown"
	}

	// Create base notification data
	notificationData := &types.NotificationEventData{
		ToolName:    toolName,
		Action:      de.getActionForTool(toolName),
		Details:     make(map[string]interface{}),
		RequestText: event.Message,
	}

	// Copy all input parameters to details
	if toolUse.Input != nil {
		for key, value := range toolUse.Input {
			notificationData.Details[key] = value
		}
	}

	// Add tool-specific processing
	switch strings.ToLower(toolName) {
	case "write":
		de.processWriteTool(toolUse, notificationData)
	case "read":
		de.processReadTool(toolUse, notificationData)
	case "webfetch", "fetch":
		de.processWebFetchTool(toolUse, notificationData)
	case "bash":
		de.processBashTool(toolUse, notificationData)
	case "edit":
		de.processEditTool(toolUse, notificationData)
	case "list", "ls":
		de.processListTool(toolUse, notificationData)
	default:
		// Generic tool processing
		de.processGenericTool(toolUse, notificationData)
	}

	return notificationData, nil
}

// processWriteTool handles Write tool specific processing
func (de *DataExtractor) processWriteTool(toolUse *types.ContentItem, data *types.NotificationEventData) {
	data.Action = "create_file"
	
	if filePath, exists := toolUse.Input["file_path"]; exists {
		data.Details["target_file"] = filePath
	}
	if content, exists := toolUse.Input["content"]; exists {
		// Truncate very long content for preview
		contentStr := fmt.Sprintf("%v", content)
		if len(contentStr) > 200 {
			data.Details["content_preview"] = contentStr[:200] + "..."
			data.Details["content_length"] = len(contentStr)
		} else {
			data.Details["content_preview"] = contentStr
		}
	}
}

// processReadTool handles Read tool specific processing
func (de *DataExtractor) processReadTool(toolUse *types.ContentItem, data *types.NotificationEventData) {
	data.Action = "read_file"
	
	if filePath, exists := toolUse.Input["file_path"]; exists {
		data.Details["target_file"] = filePath
	}
}

// processWebFetchTool handles WebFetch tool specific processing
func (de *DataExtractor) processWebFetchTool(toolUse *types.ContentItem, data *types.NotificationEventData) {
	data.Action = "fetch_url"
	
	if url, exists := toolUse.Input["url"]; exists {
		data.Details["target_url"] = url
	}
	if prompt, exists := toolUse.Input["prompt"]; exists {
		data.Details["fetch_prompt"] = prompt
	}
}

// processBashTool handles Bash tool specific processing
func (de *DataExtractor) processBashTool(toolUse *types.ContentItem, data *types.NotificationEventData) {
	data.Action = "execute_command"
	
	if command, exists := toolUse.Input["command"]; exists {
		data.Details["command"] = command
	}
	if description, exists := toolUse.Input["description"]; exists {
		data.Details["command_description"] = description
	}
}

// processEditTool handles Edit tool specific processing
func (de *DataExtractor) processEditTool(toolUse *types.ContentItem, data *types.NotificationEventData) {
	data.Action = "edit_file"
	
	if filePath, exists := toolUse.Input["file_path"]; exists {
		data.Details["target_file"] = filePath
	}
	if oldString, exists := toolUse.Input["old_string"]; exists {
		// Truncate for preview
		oldStr := fmt.Sprintf("%v", oldString)
		if len(oldStr) > 100 {
			data.Details["old_string_preview"] = oldStr[:100] + "..."
		} else {
			data.Details["old_string_preview"] = oldStr
		}
	}
	if newString, exists := toolUse.Input["new_string"]; exists {
		// Truncate for preview
		newStr := fmt.Sprintf("%v", newString)
		if len(newStr) > 100 {
			data.Details["new_string_preview"] = newStr[:100] + "..."
		} else {
			data.Details["new_string_preview"] = newStr
		}
	}
}

// processListTool handles List/LS tool specific processing
func (de *DataExtractor) processListTool(toolUse *types.ContentItem, data *types.NotificationEventData) {
	data.Action = "list_directory"
	
	if path, exists := toolUse.Input["path"]; exists {
		data.Details["target_path"] = path
	}
}

// processGenericTool handles unknown tools
func (de *DataExtractor) processGenericTool(toolUse *types.ContentItem, data *types.NotificationEventData) {
	data.Action = "use_tool"
	data.Details["tool_id"] = toolUse.ID
}

// getActionForTool returns a human-readable action for a tool
func (de *DataExtractor) getActionForTool(toolName string) string {
	switch strings.ToLower(toolName) {
	case "write":
		return "create_file"
	case "read":
		return "read_file"
	case "edit":
		return "edit_file"
	case "webfetch", "fetch":
		return "fetch_url"
	case "bash":
		return "execute_command"
	case "list", "ls":
		return "list_directory"
	default:
		return "use_tool"
	}
}

// determineTaskStatus determines the task status from the final message
func (de *DataExtractor) determineTaskStatus(finalMessage string, message *types.TranscriptMessage) string {
	lowerMsg := strings.ToLower(finalMessage)
	
	// Check for error indicators
	if strings.Contains(lowerMsg, "error") || 
	   strings.Contains(lowerMsg, "failed") || 
	   strings.Contains(lowerMsg, "cannot") ||
	   strings.Contains(lowerMsg, "unable") {
		return "error"
	}
	
	// Check for completion indicators
	if strings.Contains(lowerMsg, "completed") || 
	   strings.Contains(lowerMsg, "done") || 
	   strings.Contains(lowerMsg, "finished") ||
	   strings.Contains(lowerMsg, "created") ||
	   strings.Contains(lowerMsg, "updated") ||
	   strings.Contains(lowerMsg, "successfully") {
		return "completed"
	}
	
	// Check message usage for completion indicators
	if message.Message.Usage != nil && message.Message.Usage.OutputTokens > 0 {
		return "completed"
	}
	
	return "completed" // Default to completed for stop events
}

// generateSummary generates a brief summary of the final message
func (de *DataExtractor) generateSummary(finalMessage string) string {
	// Truncate long messages
	if len(finalMessage) <= 100 {
		return finalMessage
	}
	
	// Find a good break point (sentence end, period, etc.)
	truncated := finalMessage[:97]
	
	// Look for last period or sentence break
	lastPeriod := strings.LastIndex(truncated, ".")
	lastExclamation := strings.LastIndex(truncated, "!")
	lastQuestion := strings.LastIndex(truncated, "?")
	
	breakPoint := maxOfThree(lastPeriod, lastExclamation, lastQuestion)
	if breakPoint > 50 { // Only use break point if it's not too early
		return finalMessage[:breakPoint+1]
	}
	
	return truncated + "..."
}

// GetEventContext gets additional context for an event by analyzing recent transcript messages
func (de *DataExtractor) GetEventContext(event *types.ClaudeHookEvent, maxMessages int) (map[string]interface{}, error) {
	context := make(map[string]interface{})
	
	// Get recent messages for context
	recentMessages, err := de.transcriptReader.GetConversationContext(event.TranscriptPath, maxMessages)
	if err != nil {
		return context, err // Return empty context rather than error
	}
	
	// Count message types
	userMessages := de.transcriptReader.GetMessagesByType(recentMessages, "user")
	assistantMessages := de.transcriptReader.GetMessagesByType(recentMessages, "assistant")
	
	context["recent_user_messages"] = len(userMessages)
	context["recent_assistant_messages"] = len(assistantMessages)
	context["total_recent_messages"] = len(recentMessages)
	
	// Get session info
	sessionInfo, err := de.transcriptReader.GetSessionInfo(event.TranscriptPath)
	if err == nil {
		context["session_info"] = sessionInfo
	}
	
	return context, nil
}

// maxOfThree returns the maximum of three integers
func maxOfThree(a, b, c int) int {
	if a >= b && a >= c {
		return a
	}
	if b >= c {
		return b
	}
	return c
}