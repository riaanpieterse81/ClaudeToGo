package transcript

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/riaanpieterse81/ClaudeToGo/internal/types"
)

// Reader handles reading and parsing Claude Code transcript files
type Reader struct{}

// NewReader creates a new transcript reader
func NewReader() *Reader {
	return &Reader{}
}

// ReadLatestMessage reads the last message from a transcript file
func (r *Reader) ReadLatestMessage(transcriptPath string) (*types.TranscriptMessage, error) {
	messages, err := r.ParseTranscriptFile(transcriptPath)
	if err != nil {
		return nil, err
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages found in transcript file: %s", transcriptPath)
	}

	// Return the last message
	return &messages[len(messages)-1], nil
}

// GetLastAssistantMessage finds the most recent assistant message in the transcript
func (r *Reader) GetLastAssistantMessage(transcriptPath string) (*types.TranscriptMessage, error) {
	messages, err := r.ParseTranscriptFile(transcriptPath)
	if err != nil {
		return nil, err
	}

	// Search backwards for the last assistant message
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Type == "assistant" {
			return &messages[i], nil
		}
	}

	return nil, fmt.Errorf("no assistant messages found in transcript")
}

// GetLastToolUse finds the most recent tool use message from assistant
func (r *Reader) GetLastToolUse(transcriptPath string) (*types.TranscriptMessage, error) {
	messages, err := r.ParseTranscriptFile(transcriptPath)
	if err != nil {
		return nil, err
	}

	// Search backwards for the last assistant message with tool_use content
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Type == "assistant" {
			if r.hasToolUse(&messages[i]) {
				return &messages[i], nil
			}
		}
	}

	return nil, fmt.Errorf("no tool use messages found in transcript")
}

// ParseTranscriptFile reads and parses an entire transcript JSONL file
func (r *Reader) ParseTranscriptFile(path string) ([]types.TranscriptMessage, error) {
	if !r.fileExists(path) {
		return nil, fmt.Errorf("transcript file does not exist: %s", path)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open transcript file: %w", err)
	}
	defer file.Close()

	var messages []types.TranscriptMessage
	scanner := bufio.NewScanner(file)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines
		if line == "" {
			continue
		}

		var message types.TranscriptMessage
		if err := json.Unmarshal([]byte(line), &message); err != nil {
			return nil, fmt.Errorf("failed to parse line %d in transcript file %s: %w", lineNum, path, err)
		}

		messages = append(messages, message)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading transcript file: %w", err)
	}

	return messages, nil
}

// ExtractTextContent extracts text content from a message
func (r *Reader) ExtractTextContent(message *types.TranscriptMessage) string {
	var textParts []string

	// Handle different content types (string or []ContentItem)
	switch content := message.Message.Content.(type) {
	case string:
		// For user messages, content is a string
		return content
	case []interface{}:
		// For assistant messages, content is an array
		for _, item := range content {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if itemType, exists := itemMap["type"]; exists && itemType == "text" {
					if text, exists := itemMap["text"]; exists {
						if textStr, ok := text.(string); ok {
							textParts = append(textParts, textStr)
						}
					}
				}
			}
		}
	}

	return strings.Join(textParts, " ")
}

// ExtractToolUseDetails extracts tool use details from a message
func (r *Reader) ExtractToolUseDetails(message *types.TranscriptMessage) (*types.ContentItem, error) {
	// Handle different content types (string or []ContentItem)
	switch content := message.Message.Content.(type) {
	case string:
		// User messages don't have tool use
		return nil, fmt.Errorf("user messages don't contain tool_use content")
	case []interface{}:
		// For assistant messages, content is an array
		for _, item := range content {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if itemType, exists := itemMap["type"]; exists && itemType == "tool_use" {
					// Convert map back to ContentItem
					contentItem := &types.ContentItem{
						Type: "tool_use",
					}
					if id, exists := itemMap["id"]; exists {
						if idStr, ok := id.(string); ok {
							contentItem.ID = idStr
						}
					}
					if name, exists := itemMap["name"]; exists {
						if nameStr, ok := name.(string); ok {
							contentItem.Name = nameStr
						}
					}
					if input, exists := itemMap["input"]; exists {
						if inputMap, ok := input.(map[string]interface{}); ok {
							contentItem.Input = inputMap
						}
					}
					return contentItem, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no tool_use content found in message")
}

// GetConversationContext gets the last few messages for context
func (r *Reader) GetConversationContext(transcriptPath string, maxMessages int) ([]types.TranscriptMessage, error) {
	messages, err := r.ParseTranscriptFile(transcriptPath)
	if err != nil {
		return nil, err
	}

	// Return the last N messages
	start := 0
	if len(messages) > maxMessages {
		start = len(messages) - maxMessages
	}

	return messages[start:], nil
}

// GetMessagesByType filters messages by type (user, assistant)
func (r *Reader) GetMessagesByType(messages []types.TranscriptMessage, messageType string) []types.TranscriptMessage {
	var filtered []types.TranscriptMessage
	
	for _, message := range messages {
		if message.Type == messageType {
			filtered = append(filtered, message)
		}
	}

	return filtered
}

// FindToolUseByName finds the last tool use of a specific tool name
func (r *Reader) FindToolUseByName(transcriptPath string, toolName string) (*types.TranscriptMessage, error) {
	messages, err := r.ParseTranscriptFile(transcriptPath)
	if err != nil {
		return nil, err
	}

	// Search backwards for tool use with specific name
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Type == "assistant" {
			if r.hasToolUseWithName(&messages[i], toolName) {
				return &messages[i], nil
			}
		}
	}

	return nil, fmt.Errorf("no tool use found for tool: %s", toolName)
}

// GetSessionInfo extracts session information from any message in the transcript
func (r *Reader) GetSessionInfo(transcriptPath string) (*SessionInfo, error) {
	messages, err := r.ParseTranscriptFile(transcriptPath)
	if err != nil {
		return nil, err
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages found in transcript")
	}

	// Use the first message to get session info
	firstMessage := messages[0]
	
	return &SessionInfo{
		SessionID: firstMessage.SessionID,
		CWD:       firstMessage.CWD,
		Version:   firstMessage.Version,
		GitBranch: firstMessage.GitBranch,
	}, nil
}

// SessionInfo contains session metadata
type SessionInfo struct {
	SessionID string `json:"session_id"`
	CWD       string `json:"cwd"`
	Version   string `json:"version"`
	GitBranch string `json:"git_branch"`
}

// fileExists checks if a file exists
func (r *Reader) fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// GetMessageChain gets a chain of related messages by following parent UUIDs
func (r *Reader) GetMessageChain(transcriptPath string, startUUID string) ([]types.TranscriptMessage, error) {
	messages, err := r.ParseTranscriptFile(transcriptPath)
	if err != nil {
		return nil, err
	}

	// Create a map for quick UUID lookup
	messageMap := make(map[string]types.TranscriptMessage)
	for _, msg := range messages {
		messageMap[msg.UUID] = msg
	}

	var chain []types.TranscriptMessage
	currentUUID := startUUID

	// Follow the chain backwards via ParentUUID
	for currentUUID != "" {
		if msg, exists := messageMap[currentUUID]; exists {
			chain = append([]types.TranscriptMessage{msg}, chain...) // Prepend to maintain order
			currentUUID = msg.ParentUUID
		} else {
			break
		}
	}

	return chain, nil
}

// hasToolUse checks if a message contains any tool_use content
func (r *Reader) hasToolUse(message *types.TranscriptMessage) bool {
	switch content := message.Message.Content.(type) {
	case []interface{}:
		for _, item := range content {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if itemType, exists := itemMap["type"]; exists && itemType == "tool_use" {
					return true
				}
			}
		}
	}
	return false
}

// hasToolUseWithName checks if a message contains tool_use content with a specific name
func (r *Reader) hasToolUseWithName(message *types.TranscriptMessage, toolName string) bool {
	switch content := message.Message.Content.(type) {
	case []interface{}:
		for _, item := range content {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if itemType, exists := itemMap["type"]; exists && itemType == "tool_use" {
					if name, exists := itemMap["name"]; exists {
						if nameStr, ok := name.(string); ok && nameStr == toolName {
							return true
						}
					}
				}
			}
		}
	}
	return false
}