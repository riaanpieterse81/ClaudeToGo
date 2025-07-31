package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/riaanpieterse81/ClaudeToGo/internal/logger"
	"github.com/riaanpieterse81/ClaudeToGo/internal/processor"
)

// EventWatcher monitors claude-events.jsonl for new events and processes them automatically
type EventWatcher struct {
	eventsFile     string
	outputDir      string
	processor      *processor.EventProcessor
	lastProcessed  time.Time
	pollInterval   time.Duration
	logger         *logger.Logger
	lastFileSize   int64
	lastEventCount int
}

// WatcherConfig contains configuration for the event watcher
type WatcherConfig struct {
	EventsFile   string
	OutputDir    string
	PollInterval time.Duration
	Logger       *logger.Logger
}

// NewEventWatcher creates a new event watcher
func NewEventWatcher(config WatcherConfig) *EventWatcher {
	// Default values
	if config.EventsFile == "" {
		config.EventsFile = "claude-events.jsonl"
	}
	if config.OutputDir == "" {
		config.OutputDir = "messenger-output"
	}
	if config.PollInterval == 0 {
		config.PollInterval = 2 * time.Second
	}

	return &EventWatcher{
		eventsFile:   config.EventsFile,
		outputDir:    config.OutputDir,
		processor:    processor.NewEventProcessor(config.OutputDir),
		pollInterval: config.PollInterval,
		logger:       config.Logger,
	}
}

// Start begins monitoring the events file for changes
func (ew *EventWatcher) Start(ctx context.Context) error {
	ew.logger.Info("Starting event watcher service...")
	ew.logger.Info("Watching: %s", ew.eventsFile)
	ew.logger.Info("Output: %s", ew.outputDir)
	ew.logger.Info("Poll interval: %v", ew.pollInterval)

	// Initialize baseline
	if err := ew.initializeBaseline(); err != nil {
		return fmt.Errorf("failed to initialize baseline: %w", err)
	}

	ticker := time.NewTicker(ew.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			ew.logger.Info("Event watcher service stopped")
			return nil
		case <-ticker.C:
			if err := ew.checkForNewEvents(); err != nil {
				ew.logger.Error("Error checking for new events: %v", err)
				// Continue running despite errors
			}
		}
	}
}

// initializeBaseline establishes the starting point for monitoring
func (ew *EventWatcher) initializeBaseline() error {
	// Check if events file exists
	if !ew.fileExists(ew.eventsFile) {
		ew.logger.Info("Events file does not exist yet: %s", ew.eventsFile)
		ew.lastFileSize = 0
		ew.lastEventCount = 0
		ew.lastProcessed = time.Now()
		return nil
	}

	// Get initial file size
	fileInfo, err := os.Stat(ew.eventsFile)
	if err != nil {
		return fmt.Errorf("failed to stat events file: %w", err)
	}
	ew.lastFileSize = fileInfo.Size()

	// Get initial event count
	stats, err := ew.processor.GetProcessingStats(ew.eventsFile)
	if err != nil {
		ew.logger.Debug("Could not get initial stats: %v", err)
		ew.lastEventCount = 0
	} else {
		ew.lastEventCount = stats.TotalEvents
		ew.logger.Info("Baseline established: %d events, %d bytes", ew.lastEventCount, ew.lastFileSize)
	}

	ew.lastProcessed = time.Now()
	return nil
}

// checkForNewEvents checks if there are new events to process
func (ew *EventWatcher) checkForNewEvents() error {
	// Check if file exists
	if !ew.fileExists(ew.eventsFile) {
		return nil // File doesn't exist yet, that's OK
	}

	// Check file size first (quick check)
	fileInfo, err := os.Stat(ew.eventsFile)
	if err != nil {
		return fmt.Errorf("failed to stat events file: %w", err)
	}

	currentFileSize := fileInfo.Size()
	if currentFileSize == ew.lastFileSize {
		// No change in file size, skip processing
		return nil
	}

	// File has changed, check event count
	stats, err := ew.processor.GetProcessingStats(ew.eventsFile)
	if err != nil {
		return fmt.Errorf("failed to get processing stats: %w", err)
	}

	if stats.TotalEvents > ew.lastEventCount {
		newEvents := stats.TotalEvents - ew.lastEventCount
		ew.logger.Info("Detected %d new event(s), processing...", newEvents)

		// Process the new events
		outputFiles, err := ew.processNewEvents(newEvents)
		if err != nil {
			return fmt.Errorf("failed to process new events: %w", err)
		}

		// Log results
		for _, file := range outputFiles {
			ew.logger.Info("Generated: %s", file)
		}

		// Update tracking variables
		ew.lastEventCount = stats.TotalEvents
		ew.lastFileSize = currentFileSize
		ew.lastProcessed = time.Now()

		ew.logger.Info("Successfully processed %d new events", len(outputFiles))
	}

	return nil
}

// processNewEvents processes the most recent events
func (ew *EventWatcher) processNewEvents(count int) ([]string, error) {
	return ew.processor.ProcessLatestEvents(ew.eventsFile, count)
}

// GetStats returns current watcher statistics
func (ew *EventWatcher) GetStats() (*WatcherStats, error) {
	stats, err := ew.processor.GetProcessingStats(ew.eventsFile)
	if err != nil {
		return nil, err
	}

	return &WatcherStats{
		EventsFile:        ew.eventsFile,
		OutputDir:         ew.outputDir,
		PollInterval:      ew.pollInterval,
		LastProcessed:     ew.lastProcessed,
		TotalEvents:       stats.TotalEvents,
		ProcessableEvents: stats.ProcessableEvents,
		IsRunning:         true,
	}, nil
}

// fileExists checks if a file exists
func (ew *EventWatcher) fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// Stop gracefully stops the watcher (called via context cancellation)
func (ew *EventWatcher) Stop() {
	ew.logger.Info("Stopping event watcher service...")
}

// WatcherStats contains statistics about the watcher service
type WatcherStats struct {
	EventsFile        string        `json:"events_file"`
	OutputDir         string        `json:"output_dir"`
	PollInterval      time.Duration `json:"poll_interval"`
	LastProcessed     time.Time     `json:"last_processed"`
	TotalEvents       int           `json:"total_events"`
	ProcessableEvents int           `json:"processable_events"`
	IsRunning         bool          `json:"is_running"`
}

// ServiceMode runs the watcher as a background service
func ServiceMode(ctx context.Context, config WatcherConfig) error {
	watcher := NewEventWatcher(config)

	// Ensure output directory exists
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create a status file to indicate the service is running
	statusFile := filepath.Join(config.OutputDir, ".watcher-status")
	if err := watcher.createStatusFile(statusFile); err != nil {
		config.Logger.Debug("Could not create status file: %v", err)
	}

	// Clean up status file when done
	defer func() {
		if err := os.Remove(statusFile); err != nil {
			config.Logger.Debug("Could not remove status file: %v", err)
		}
	}()

	return watcher.Start(ctx)
}

// createStatusFile creates a status file indicating the service is running
func (ew *EventWatcher) createStatusFile(statusFile string) error {
	status := fmt.Sprintf(`{
  "service": "claudetogo-watcher",
  "status": "running",
  "started": "%s",
  "events_file": "%s",
  "output_dir": "%s",
  "poll_interval": "%s",
  "pid": %d
}`, time.Now().Format(time.RFC3339), ew.eventsFile, ew.outputDir, ew.pollInterval, os.Getpid())

	return os.WriteFile(statusFile, []byte(status), 0644)
}