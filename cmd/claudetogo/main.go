package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/riaanpieterse81/ClaudeToGo/internal/config"
	messengerConfig "github.com/riaanpieterse81/ClaudeToGo/internal/config"
	"github.com/riaanpieterse81/ClaudeToGo/internal/hooks"
	"github.com/riaanpieterse81/ClaudeToGo/internal/logger"
	"github.com/riaanpieterse81/ClaudeToGo/internal/monitor"
	"github.com/riaanpieterse81/ClaudeToGo/internal/processor"
	"github.com/riaanpieterse81/ClaudeToGo/internal/responder"
	"github.com/riaanpieterse81/ClaudeToGo/internal/service"
	"github.com/riaanpieterse81/ClaudeToGo/internal/setup"
	"github.com/riaanpieterse81/ClaudeToGo/internal/types"
)

func showHelp() {
	fmt.Printf("Usage: %s [options]\n\n", os.Args[0])
	fmt.Println("Description:")
	fmt.Println("  A tool for logging and monitoring Claude Code hook events")
	fmt.Println()
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  claudetogo --help                           Show this help")
	fmt.Println("  claudetogo --setup                          Run interactive setup wizard (recommended for first use)")
	fmt.Println("  claudetogo --hook                           Process hook event from stdin (logs and allows all events)")
	fmt.Println("  claudetogo --config myconfig.json           Use custom configuration file")
	fmt.Println("  claudetogo --monitor                        Monitor events in real-time")
	fmt.Println("  claudetogo --monitor --verbose              Monitor with debug output")
	fmt.Println()
	fmt.Println("Processing Commands:")
	fmt.Println("  claudetogo --process                        Process all events and generate messenger JSON files")
	fmt.Println("  claudetogo --process --latest 5             Process latest 5 events only")
	fmt.Println("  claudetogo --process --generate-samples     Generate test samples from real data")
	fmt.Println("  claudetogo --process --stats               Get processing statistics")
	fmt.Println("  claudetogo --process --watch --interval 5s  Watch for new events and process them")
	fmt.Println("  claudetogo --process --output-dir custom/   Use custom output directory")
	fmt.Println()
	fmt.Println("Response Commands:")
	fmt.Println("  claudetogo --respond --session 1fa8811f --action approve   Approve a pending action")
	fmt.Println("  claudetogo --respond --session 1fa8811f --action reject    Reject a pending action")
	fmt.Println("  claudetogo --status --session 1fa8811f                     Get session status")
	fmt.Println("  claudetogo --pending                                       List pending actions")
	fmt.Println()
	fmt.Println("Service Commands:")
	fmt.Println("  claudetogo --service                                       Run as background service")
	fmt.Println("  claudetogo --service --daemon                              Run as daemon (background)")
	fmt.Println("  claudetogo --service --interval 10s                       Custom service poll interval")
	fmt.Println()
	fmt.Println("Configuration Commands:")
	fmt.Println("  claudetogo --config-init                                   Create example messenger config file")
	fmt.Println("  claudetogo --config-show                                   Show current configuration")
	fmt.Println("  claudetogo --config-validate claudetogo-messenger.yaml    Validate configuration file")
	fmt.Println("  claudetogo --messenger-config myconfig.yaml               Use custom messenger config")
	fmt.Println()
	fmt.Println("Getting Started:")
	fmt.Println("  For first-time users, run 'claudetogo --setup' to configure the application")
}

// setupGracefulShutdown sets up graceful shutdown handling
func setupGracefulShutdown() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("\nReceived shutdown signal, stopping gracefully...")
		cancel()
	}()

	return ctx, cancel
}

func main() {
	// Command line flags
	helpFlag := flag.Bool("help", false, "Show help information")
	setupFlag := flag.Bool("setup", false, "Run interactive setup wizard to configure the application")
	configFlag := flag.String("config", "", "Path to configuration file (JSON format)")
	hookFlag := flag.Bool("hook", false, "Process hook event from stdin (for Claude Code hooks)")
	monitorFlag := flag.Bool("monitor", false, "Monitor events in real-time")
	logFileFlag := flag.String("logfile", "claude-events.jsonl", "Path to log file")
	verboseFlag := flag.Bool("verbose", false, "Enable verbose debug output")
	pollIntervalFlag := flag.Duration("poll-interval", 100*time.Millisecond, "Polling interval for monitoring")

	// Processing command flags
	processFlag := flag.Bool("process", false, "Process Claude events and generate messenger JSON files")
	eventsFileFlag := flag.String("events-file", "claude-events.jsonl", "Path to events file for processing")
	outputDirFlag := flag.String("output-dir", "messenger-output", "Output directory for messenger JSON files")
	latestFlag := flag.Int("latest", 0, "Process only the latest N events (0 = all events)")
	generateSamplesFlag := flag.Bool("generate-samples", false, "Generate test samples from real data")
	statsFlag := flag.Bool("stats", false, "Show processing statistics")
	processWatchFlag := flag.Bool("watch", false, "Watch for new events and process them continuously")
	intervalFlag := flag.Duration("interval", 5*time.Second, "Interval for watch mode processing")

	// Response command flags
	respondFlag := flag.Bool("respond", false, "Respond to a notification event")
	sessionFlag := flag.String("session", "", "Session ID for response or status commands")
	actionFlag := flag.String("action", "", "Action to take (approve, reject)")
	statusFlag := flag.Bool("status", false, "Get session status")
	pendingFlag := flag.Bool("pending", false, "List pending actions")

	// Service command flags
	serviceFlag := flag.Bool("service", false, "Run as background service")
	daemonFlag := flag.Bool("daemon", false, "Run service in daemon mode (background)")
	serviceIntervalFlag := flag.Duration("service-interval", 2*time.Second, "Service mode poll interval")

	// Configuration command flags
	configInitFlag := flag.Bool("config-init", false, "Create example messenger configuration file")
	configShowFlag := flag.Bool("config-show", false, "Show current configuration")
	configValidateFlag := flag.String("config-validate", "", "Validate messenger configuration file")
	messengerConfigFlag := flag.String("messenger-config", "", "Path to messenger configuration file")

	flag.Parse()

	// Show help and exit
	if *helpFlag {
		showHelp()
		return
	}

	// Run setup wizard
	if *setupFlag {
		if err := setup.RunWizard(); err != nil {
			log.Printf("[ERROR] Setup failed: %v", err)
			os.Exit(1)
		}
		return
	}

	// Initialize configuration with defaults
	runtimeConfig := types.Config{
		LogFile:      "claude-events.jsonl",
		PollInterval: 100 * time.Millisecond,
		Verbose:      false,
	}

	// Load configuration file if specified or default exists
	var configPath string
	if *configFlag != "" {
		configPath = *configFlag
	} else {
		// Check for default config file
		if _, err := os.Stat("claudetogo-config.json"); err == nil {
			configPath = "claudetogo-config.json"
		}
	}

	if configPath != "" {
		configFile, err := config.Load(configPath)
		if err != nil {
			log.Printf("[ERROR] Failed to load config file '%s': %v", configPath, err)
			os.Exit(1)
		}

		if err := config.Apply(configFile, &runtimeConfig); err != nil {
			log.Printf("[ERROR] Failed to apply config file: %v", err)
			os.Exit(1)
		}

		log.Printf("[INFO] Loaded configuration from: %s", configPath)
	}

	// Command line flags override config file settings
	if flag.Lookup("logfile").Value.String() != flag.Lookup("logfile").DefValue {
		runtimeConfig.LogFile = *logFileFlag
	}
	if flag.Lookup("poll-interval").Value.String() != flag.Lookup("poll-interval").DefValue {
		runtimeConfig.PollInterval = *pollIntervalFlag
	}
	if *verboseFlag {
		runtimeConfig.Verbose = true
	}

	// Initialize logger
	appLogger := logger.New(runtimeConfig.Verbose)

	// Set up graceful shutdown
	ctx, cancel := setupGracefulShutdown()
	defer cancel()

	// Handle different modes
	if *configInitFlag {
		if err := handleConfigInitCommand(appLogger); err != nil {
			appLogger.Error("Config init command error: %v", err)
			os.Exit(1)
		}
		return
	}

	if *configShowFlag {
		if err := handleConfigShowCommand(*messengerConfigFlag, appLogger); err != nil {
			appLogger.Error("Config show command error: %v", err)
			os.Exit(1)
		}
		return
	}

	if *configValidateFlag != "" {
		if err := handleConfigValidateCommand(*configValidateFlag, appLogger); err != nil {
			appLogger.Error("Config validate command error: %v", err)
			os.Exit(1)
		}
		return
	}

	if *serviceFlag {
		if err := handleServiceCommand(ctx, *eventsFileFlag, *outputDirFlag, *daemonFlag, *serviceIntervalFlag, appLogger); err != nil {
			appLogger.Error("Service command error: %v", err)
			os.Exit(1)
		}
		return
	}

	if *processFlag {
		if err := handleProcessCommand(ctx, *eventsFileFlag, *outputDirFlag, *latestFlag, *generateSamplesFlag, *statsFlag, *processWatchFlag, *intervalFlag, appLogger); err != nil {
			appLogger.Error("Process command error: %v", err)
			os.Exit(1)
		}
		return
	}

	if *respondFlag {
		if err := handleRespondCommand(*sessionFlag, *actionFlag, appLogger); err != nil {
			appLogger.Error("Respond command error: %v", err)
			os.Exit(1)
		}
		return
	}

	if *statusFlag {
		if err := handleStatusCommand(*sessionFlag, appLogger); err != nil {
			appLogger.Error("Status command error: %v", err)
			os.Exit(1)
		}
		return
	}

	if *pendingFlag {
		if err := handlePendingCommand(appLogger); err != nil {
			appLogger.Error("Pending command error: %v", err)
			os.Exit(1)
		}
		return
	}

	if *monitorFlag {
		appLogger.Info("Monitoring Claude events... (Press Ctrl+C to stop)")
		if err := monitor.Start(ctx, runtimeConfig, appLogger); err != nil && err != context.Canceled {
			appLogger.Error("Monitor error: %v", err)
			os.Exit(1)
		}
		return
	}

	if *hookFlag {
		if err := hooks.ProcessFromStdin(runtimeConfig, appLogger); err != nil {
			appLogger.Error("Hook processing error: %v", err)
			os.Exit(1)
		}
		return
	}

	// No flags specified, show help
	showHelp()
}

// handleProcessCommand handles the --process command with all its sub-options
func handleProcessCommand(ctx context.Context, eventsFile, outputDir string, latest int, generateSamples, stats, watch bool, interval time.Duration, logger *logger.Logger) error {
	// Create processor
	eventProcessor := processor.NewEventProcessor(outputDir)

	// Handle stats command
	if stats {
		return handleStatsCommand(eventsFile, eventProcessor, logger)
	}

	// Handle generate samples command
	if generateSamples {
		return handleGenerateSamplesCommand(eventsFile, eventProcessor, logger)
	}

	// Handle watch mode
	if watch {
		return handleWatchCommand(ctx, eventsFile, eventProcessor, interval, logger)
	}

	// Handle regular processing (all events or latest N)
	return handleRegularProcessing(eventsFile, eventProcessor, latest, logger)
}

// handleStatsCommand shows processing statistics
func handleStatsCommand(eventsFile string, eventProcessor *processor.EventProcessor, logger *logger.Logger) error {
	logger.Info("Getting processing statistics...")
	
	stats, err := eventProcessor.GetProcessingStats(eventsFile)
	if err != nil {
		return fmt.Errorf("failed to get processing stats: %w", err)
	}

	fmt.Printf("\n📊 Processing Statistics for %s\n", eventsFile)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Total Events:         %d\n", stats.TotalEvents)
	fmt.Printf("Stop Events:          %d\n", stats.StopEvents)
	fmt.Printf("Notification Events:  %d\n", stats.NotificationEvents)
	fmt.Printf("Processable Events:   %d\n", stats.ProcessableEvents)
	fmt.Printf("Missing Transcripts:  %d\n", stats.MissingTranscripts)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	if stats.ProcessableEvents > 0 {
		fmt.Printf("✅ Ready to process %d events\n", stats.ProcessableEvents)
	} else {
		fmt.Printf("⚠️  No processable events found\n")
	}

	return nil
}

// handleGenerateSamplesCommand generates test samples
func handleGenerateSamplesCommand(eventsFile string, eventProcessor *processor.EventProcessor, logger *logger.Logger) error {
	logger.Info("Generating test samples from real data...")
	
	if err := eventProcessor.GenerateTestData(eventsFile); err != nil {
		return fmt.Errorf("failed to generate test samples: %w", err)
	}

	fmt.Printf("✅ Test samples generated successfully\n")
	fmt.Printf("📁 Check %s/test-samples/ for sample files\n", eventProcessor.GetOutputDirectory())
	
	return nil
}

// handleWatchCommand handles continuous monitoring and processing
func handleWatchCommand(ctx context.Context, eventsFile string, eventProcessor *processor.EventProcessor, interval time.Duration, logger *logger.Logger) error {
	logger.Info("Starting watch mode for new events... (Press Ctrl+C to stop)")
	fmt.Printf("📁 Watching: %s\n", eventsFile)
	fmt.Printf("📂 Output:   %s\n", eventProcessor.GetOutputDirectory())
	fmt.Printf("⏱️  Interval: %v\n", interval)
	fmt.Println()

	// Keep track of last processed event count
	lastEventCount := 0
	
	// Get initial event count
	if stats, err := eventProcessor.GetProcessingStats(eventsFile); err == nil {
		lastEventCount = stats.TotalEvents
		logger.Debug("Initial event count: %d", lastEventCount)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\n🛑 Watch mode stopped")
			return nil
		case <-ticker.C:
			// Check for new events
			stats, err := eventProcessor.GetProcessingStats(eventsFile)
			if err != nil {
				logger.Debug("Failed to get stats during watch: %v", err)
				continue
			}

			if stats.TotalEvents > lastEventCount {
				newEvents := stats.TotalEvents - lastEventCount
				logger.Info("Found %d new event(s), processing...", newEvents)
				
				// Process the latest new events
				outputFiles, err := eventProcessor.ProcessLatestEvents(eventsFile, newEvents)
				if err != nil {
					logger.Error("Failed to process new events: %v", err)
					continue
				}

				for _, file := range outputFiles {
					fmt.Printf("📝 Generated: %s\n", file)
				}
				
				lastEventCount = stats.TotalEvents
			}
		}
	}
}

// handleRegularProcessing handles regular event processing (all or latest N)
func handleRegularProcessing(eventsFile string, eventProcessor *processor.EventProcessor, latest int, logger *logger.Logger) error {
	var outputFiles []string
	var err error

	if latest > 0 {
		logger.Info("Processing latest %d events...", latest)
		outputFiles, err = eventProcessor.ProcessLatestEvents(eventsFile, latest)
	} else {
		logger.Info("Processing all events...")
		outputFiles, err = eventProcessor.ProcessEventsFromFile(eventsFile)
	}

	if err != nil {
		return fmt.Errorf("failed to process events: %w", err)
	}

	fmt.Printf("\n✅ Processing completed successfully\n")
	fmt.Printf("📁 Output directory: %s\n", eventProcessor.GetOutputDirectory())
	fmt.Printf("📊 Files generated: %d\n", len(outputFiles))
	
	if len(outputFiles) > 0 {
		fmt.Println("\n📝 Generated files:")
		for _, file := range outputFiles {
			fmt.Printf("  - %s\n", file)
		}
	}

	return nil
}

// handleRespondCommand handles user responses to notification events
func handleRespondCommand(sessionID, action string, logger *logger.Logger) error {
	if sessionID == "" {
		return fmt.Errorf("session ID is required for respond command")
	}
	if action == "" {
		return fmt.Errorf("action is required for respond command (approve, reject)")
	}

	logger.Info("Processing response for session %s with action: %s", sessionID, action)
	
	// Create response handler
	responseHandler := responder.NewResponseHandler("messenger-output", logger)
	
	// Process the response
	fmt.Printf("🔄 Processing response...\n")
	fmt.Printf("📋 Session:  %s\n", sessionID)
	fmt.Printf("⚡ Action:   %s\n", action)
	
	if err := responseHandler.HandleResponse(sessionID, action); err != nil {
		return fmt.Errorf("failed to handle response: %w", err)
	}

	switch action {
	case "approve":
		fmt.Printf("✅ Action approved and executed\n")
	case "reject":
		fmt.Printf("❌ Action rejected\n")
	case "info":
		fmt.Printf("ℹ️  Information displayed\n")
	default:
		fmt.Printf("✅ Action '%s' processed\n", action)
	}

	fmt.Printf("✅ Response processed successfully\n")
	return nil
}

// handleStatusCommand shows status for a specific session
func handleStatusCommand(sessionID string, logger *logger.Logger) error {
	if sessionID == "" {
		return fmt.Errorf("session ID is required for status command")
	}

	logger.Info("Getting status for session: %s", sessionID)
	
	// Create response handler
	responseHandler := responder.NewResponseHandler("messenger-output", logger)
	
	// Get session status
	status, err := responseHandler.GetSessionStatus(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session status: %w", err)
	}

	fmt.Printf("📋 Session Status: %s\n", sessionID)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("🔍 Status:      %s\n", status.Status)
	fmt.Printf("📅 Created:     %s\n", status.CreatedAt.Format("2006-01-02 15:04:05"))
	
	if status.LastAction != "" {
		fmt.Printf("⚡ Last Action: %s\n", status.LastAction)
	}
	
	if status.Context != nil && len(status.Context) > 0 {
		fmt.Printf("📝 Context:\n")
		for key, value := range status.Context {
			fmt.Printf("   %s: %v\n", key, value)
		}
	}
	
	fmt.Printf("📁 File:       %s\n", status.MessengerFile)
	
	return nil
}

// handlePendingCommand lists all pending actions
func handlePendingCommand(logger *logger.Logger) error {
	logger.Info("Listing pending actions...")
	
	// Create response handler
	responseHandler := responder.NewResponseHandler("messenger-output", logger)
	
	// Get pending actions
	pendingActions, err := responseHandler.ListPendingActions()
	if err != nil {
		return fmt.Errorf("failed to get pending actions: %w", err)
	}

	fmt.Printf("📋 Pending Actions\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	
	if len(pendingActions) == 0 {
		fmt.Printf("✅ No pending actions found\n")
		return nil
	}

	for i, action := range pendingActions {
		fmt.Printf("%d. 📝 %s\n", i+1, action.Title)
		fmt.Printf("   Session: %s\n", action.SessionID)
		fmt.Printf("   Created: %s\n", action.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("   Message: %s\n", action.Message)
		fmt.Printf("   Commands:\n")
		fmt.Printf("     Approve: claudetogo --respond --session %s --action approve\n", action.SessionID)
		fmt.Printf("     Reject:  claudetogo --respond --session %s --action reject\n", action.SessionID)
		fmt.Printf("     Info:    claudetogo --status --session %s\n", action.SessionID)
		
		if i < len(pendingActions)-1 {
			fmt.Println()
		}
	}
	
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📊 Total pending actions: %d\n", len(pendingActions))
	
	return nil
}

// handleServiceCommand runs the background service mode
func handleServiceCommand(ctx context.Context, eventsFile, outputDir string, daemon bool, interval time.Duration, logger *logger.Logger) error {
	logger.Info("Starting ClaudeToGo service mode...")
	
	if daemon {
		logger.Info("Running in daemon mode")
		fmt.Printf("🚀 ClaudeToGo service starting in daemon mode...\n")
	} else {
		fmt.Printf("🚀 ClaudeToGo service starting...\n")
	}

	fmt.Printf("📁 Events file: %s\n", eventsFile)
	fmt.Printf("📂 Output dir:  %s\n", outputDir)
	fmt.Printf("⏱️  Interval:   %v\n", interval)
	fmt.Printf("🔄 Press Ctrl+C to stop\n")
	fmt.Println()

	// Create service config
	serviceConfig := service.WatcherConfig{
		EventsFile:   eventsFile,
		OutputDir:    outputDir,
		PollInterval: interval,
		Logger:       logger,
	}

	// Run the service
	return service.ServiceMode(ctx, serviceConfig)
}

// handleConfigInitCommand creates an example messenger configuration file
func handleConfigInitCommand(logger *logger.Logger) error {
	configPath := "claudetogo-messenger.yaml"
	
	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("⚠️  Configuration file already exists: %s\n", configPath)
		fmt.Printf("🔄 Overwrite? (y/N): ")
		
		var response string
		fmt.Scanln(&response)
		
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Printf("❌ Configuration file creation cancelled\n")
			return nil
		}
	}

	logger.Info("Creating example messenger configuration file...")
	
	if err := messengerConfig.GenerateExampleConfig(configPath); err != nil {
		return fmt.Errorf("failed to create example config: %w", err)
	}

	fmt.Printf("✅ Example configuration file created: %s\n", configPath)
	fmt.Printf("📝 Edit the file to customize your settings\n")
	fmt.Printf("🔍 Validate with: claudetogo --config-validate %s\n", configPath)
	
	return nil
}

// handleConfigShowCommand shows the current configuration
func handleConfigShowCommand(messengerConfigPath string, logger *logger.Logger) error {
	logger.Info("Loading and displaying current configuration...")

	var config *messengerConfig.MessengerConfig
	
	if messengerConfigPath != "" {
		// Load specific config file
		var err error
		config, err = messengerConfig.LoadMessengerConfig(messengerConfigPath)
		if err != nil {
			return fmt.Errorf("failed to load config from %s: %w", messengerConfigPath, err)
		}
		fmt.Printf("📁 Loaded configuration from: %s\n\n", messengerConfigPath)
	} else {
		// Load with defaults and auto-discovery
		config = messengerConfig.GetMessengerConfigWithDefaults("")
		
		foundConfig := messengerConfig.FindMessengerConfig()
		if foundConfig != "" {
			fmt.Printf("📁 Using configuration from: %s\n\n", foundConfig)
		} else {
			fmt.Printf("📁 Using default configuration (no config file found)\n\n")
		}
	}

	// Apply environment overrides
	config.ApplyEnvironmentOverrides()

	// Show configuration summary
	fmt.Println(config.Summary())
	
	return nil
}

// handleConfigValidateCommand validates a messenger configuration file
func handleConfigValidateCommand(configPath string, logger *logger.Logger) error {
	logger.Info("Validating configuration file: %s", configPath)

	fmt.Printf("🔍 Validating configuration file: %s\n", configPath)
	
	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("❌ Configuration file not found: %s\n", configPath)
		return fmt.Errorf("config file not found: %s", configPath)
	}

	// Load and validate the configuration
	config, err := messengerConfig.LoadMessengerConfig(configPath)
	if err != nil {
		fmt.Printf("❌ Configuration validation failed:\n")
		fmt.Printf("   %v\n", err)
		return err
	}

	fmt.Printf("✅ Configuration file is valid!\n\n")
	
	// Show summary of loaded config
	fmt.Println(config.Summary())
	
	return nil
}
