package processor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/riaanpieterse81/ClaudeToGo/internal/extractor"
	"github.com/riaanpieterse81/ClaudeToGo/internal/formatter"
	"github.com/riaanpieterse81/ClaudeToGo/internal/types"
)

// EventProcessor handles the complete pipeline from Claude events to messenger JSON files
type EventProcessor struct {
	extractor *extractor.DataExtractor
	formatter *formatter.MessengerFormatter
	outputDir string
}

// NewEventProcessor creates a new event processor
func NewEventProcessor(outputDir string) *EventProcessor {
	// Default output directory if not specified
	if outputDir == "" {
		outputDir = "messenger-output"
	}

	return &EventProcessor{
		extractor: extractor.NewDataExtractor(),
		formatter: formatter.NewMessengerFormatter(),
		outputDir: outputDir,
	}
}

// ProcessEvent processes a single Claude hook event and generates a messenger JSON file
func (ep *EventProcessor) ProcessEvent(event *types.ClaudeHookEvent) (*types.MessengerMessage, error) {
	// Extract data from the event
	extractedData, err := ep.extractor.ProcessEvent(event)
	if err != nil {
		return nil, fmt.Errorf("failed to extract data from event: %w", err)
	}

	// Format for messenger
	messengerMessage, err := ep.formatter.CreateActionableMessage(extractedData)
	if err != nil {
		return nil, fmt.Errorf("failed to format message for messenger: %w", err)
	}

	return messengerMessage, nil
}

// ProcessEventAndSave processes an event and saves the result to a JSON file
func (ep *EventProcessor) ProcessEventAndSave(event *types.ClaudeHookEvent) (string, error) {
	// Process the event
	messengerMessage, err := ep.ProcessEvent(event)
	if err != nil {
		return "", err
	}

	// Generate filename
	filename := ep.generateFileName(event)
	filepath := filepath.Join(ep.outputDir, filename)

	// Save to file
	err = ep.saveMessageToFile(messengerMessage, filepath)
	if err != nil {
		return "", fmt.Errorf("failed to save message to file: %w", err)
	}

	return filepath, nil
}

// ProcessEventsFromFile processes all events from a claude-events.jsonl file
func (ep *EventProcessor) ProcessEventsFromFile(eventsFilePath string) ([]string, error) {
	// Read events from file
	events, err := ep.readEventsFromFile(eventsFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read events from file: %w", err)
	}

	var outputFiles []string

	// Process each event
	for i, event := range events {
		outputFile, err := ep.ProcessEventAndSave(&event)
		if err != nil {
			fmt.Printf("Warning: Failed to process event %d: %v\n", i+1, err)
			continue
		}
		outputFiles = append(outputFiles, outputFile)
	}

	return outputFiles, nil
}

// ProcessLatestEvents processes only the most recent events (useful for monitoring)
func (ep *EventProcessor) ProcessLatestEvents(eventsFilePath string, maxEvents int) ([]string, error) {
	// Read all events
	events, err := ep.readEventsFromFile(eventsFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read events from file: %w", err)
	}

	// Take only the latest events
	start := 0
	if len(events) > maxEvents {
		start = len(events) - maxEvents
	}
	latestEvents := events[start:]

	var outputFiles []string

	// Process each latest event
	for i, event := range latestEvents {
		outputFile, err := ep.ProcessEventAndSave(&event)
		if err != nil {
			fmt.Printf("Warning: Failed to process latest event %d: %v\n", i+1, err)
			continue
		}
		outputFiles = append(outputFiles, outputFile)
	}

	return outputFiles, nil
}

// GenerateTestData creates sample JSON files using real event data
func (ep *EventProcessor) GenerateTestData(eventsFilePath string) error {
	events, err := ep.readEventsFromFile(eventsFilePath)
	if err != nil {
		return fmt.Errorf("failed to read events: %w", err)
	}

	// Create test output directory
	testDir := filepath.Join(ep.outputDir, "test-samples")
	if err := ep.ensureDirectoryExists(testDir); err != nil {
		return fmt.Errorf("failed to create test directory: %w", err)
	}

	// Process a few sample events of different types
	var stopEventProcessed, notificationEventProcessed bool
	
	for i, event := range events {
		// Skip if we've already processed both types
		if stopEventProcessed && notificationEventProcessed {
			break
		}

		// Skip if this event type is already processed
		if event.HookEventName == "Stop" && stopEventProcessed {
			continue
		}
		if event.HookEventName == "Notification" && notificationEventProcessed {
			continue
		}

		// Check if transcript file exists
		if !ep.fileExists(event.TranscriptPath) {
			fmt.Printf("Skipping event %d: transcript file not found: %s\n", i+1, event.TranscriptPath)
			continue
		}

		// Process the event
		messengerMessage, err := ep.ProcessEvent(&event)
		if err != nil {
			fmt.Printf("Warning: Failed to process test event %d: %v\n", i+1, err)
			continue
		}

		// Generate test filename
		eventType := strings.ToLower(event.HookEventName)
		filename := fmt.Sprintf("sample-%s-event.json", eventType)
		filepath := filepath.Join(testDir, filename)

		// Save the sample
		err = ep.saveMessageToFile(messengerMessage, filepath)
		if err != nil {
			fmt.Printf("Warning: Failed to save test sample %s: %v\n", filename, err)
			continue
		}

		// Mark as processed
		if event.HookEventName == "Stop" {
			stopEventProcessed = true
		} else if event.HookEventName == "Notification" {
			notificationEventProcessed = true
		}

		fmt.Printf("Created test sample: %s\n", filepath)
	}

	return nil
}

// readEventsFromFile reads claude hook events from a JSONL file
func (ep *EventProcessor) readEventsFromFile(filePath string) ([]types.ClaudeHookEvent, error) {
	if !ep.fileExists(filePath) {
		return nil, fmt.Errorf("events file does not exist: %s", filePath)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open events file: %w", err)
	}
	defer file.Close()

	var events []types.ClaudeHookEvent
	scanner := bufio.NewScanner(file)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines
		if line == "" {
			continue
		}

		var event types.ClaudeHookEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			fmt.Printf("Warning: Failed to parse line %d in events file: %v\n", lineNum, err)
			continue
		}

		events = append(events, event)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading events file: %w", err)
	}

	return events, nil
}

// saveMessageToFile saves a messenger message to a JSON file
func (ep *EventProcessor) saveMessageToFile(message *types.MessengerMessage, filePath string) error {
	// Ensure output directory exists
	dir := filepath.Dir(filePath)
	if err := ep.ensureDirectoryExists(dir); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Marshal to pretty JSON
	jsonData, err := json.MarshalIndent(message, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal message to JSON: %w", err)
	}

	// Write to file
	err = os.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	return nil
}

// generateFileName creates a filename for a messenger JSON file
func (ep *EventProcessor) generateFileName(event *types.ClaudeHookEvent) string {
	// Use current time if timestamp is empty
	timestamp := event.Timestamp
	if timestamp == "" {
		timestamp = time.Now().Format("2006-01-02T15-04-05")
	} else {
		// Clean up timestamp for filename (replace colons with dashes)
		timestamp = strings.ReplaceAll(timestamp, ":", "-")
	}

	eventType := strings.ToLower(event.HookEventName)
	sessionShort := event.SessionID
	if len(sessionShort) > 8 {
		sessionShort = sessionShort[:8]
	}

	return fmt.Sprintf("messenger-%s-%s-%s.json", eventType, sessionShort, timestamp)
}

// ensureDirectoryExists creates a directory if it doesn't exist
func (ep *EventProcessor) ensureDirectoryExists(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// fileExists checks if a file exists
func (ep *EventProcessor) fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// GetOutputDirectory returns the configured output directory
func (ep *EventProcessor) GetOutputDirectory() string {
	return ep.outputDir
}

// SetOutputDirectory changes the output directory
func (ep *EventProcessor) SetOutputDirectory(dir string) {
	ep.outputDir = dir
}

// GetProcessingStats returns statistics about processed events
func (ep *EventProcessor) GetProcessingStats(eventsFilePath string) (*ProcessingStats, error) {
	events, err := ep.readEventsFromFile(eventsFilePath)
	if err != nil {
		return nil, err
	}

	stats := &ProcessingStats{
		TotalEvents:         len(events),
		StopEvents:          0,
		NotificationEvents:  0,
		MissingTranscripts:  0,
		ProcessableEvents:   0,
	}

	for _, event := range events {
		switch event.HookEventName {
		case "Stop":
			stats.StopEvents++
		case "Notification":
			stats.NotificationEvents++
		}

		if ep.fileExists(event.TranscriptPath) {
			stats.ProcessableEvents++
		} else {
			stats.MissingTranscripts++
		}
	}

	return stats, nil
}

// ProcessingStats contains statistics about event processing
type ProcessingStats struct {
	TotalEvents        int `json:"total_events"`
	StopEvents         int `json:"stop_events"`
	NotificationEvents int `json:"notification_events"`
	ProcessableEvents  int `json:"processable_events"`
	MissingTranscripts int `json:"missing_transcripts"`
}