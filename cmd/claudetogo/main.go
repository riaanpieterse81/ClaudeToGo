package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/riaanpieterse81/ClaudeToGo/internal/config"
	"github.com/riaanpieterse81/ClaudeToGo/internal/hooks"
	"github.com/riaanpieterse81/ClaudeToGo/internal/logger"
	"github.com/riaanpieterse81/ClaudeToGo/internal/monitor"
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
	fmt.Println("  claudetogo --help                    Show this help")
	fmt.Println("  claudetogo --setup                   Run interactive setup wizard (recommended for first use)")
	fmt.Println("  claudetogo --hook                    Process hook event from stdin (logs and allows all events)")
	fmt.Println("  claudetogo --config myconfig.json    Use custom configuration file")
	fmt.Println("  claudetogo --monitor                 Monitor events in real-time")
	fmt.Println("  claudetogo --monitor --verbose       Monitor with debug output")
	fmt.Println("  claudetogo --logfile custom.log      Use custom log file")
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
	logFileFlag := flag.String("logfile", "claude-events.log", "Path to log file")
	verboseFlag := flag.Bool("verbose", false, "Enable verbose debug output")
	pollIntervalFlag := flag.Duration("poll-interval", 100*time.Millisecond, "Polling interval for monitoring")

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
		LogFile:      "claude-events.log",
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