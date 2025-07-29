package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/riaanpieterse81/ClaudeToGo/internal/claude"
	"github.com/riaanpieterse81/ClaudeToGo/internal/config"
	"github.com/riaanpieterse81/ClaudeToGo/internal/types"
)

// RunWizard guides the user through interactive setup
func RunWizard() error {
	fmt.Println("🎯 Welcome to ClaudeToGo Setup Wizard!")
	fmt.Println("=====================================")
	fmt.Println()
	fmt.Println("This wizard will help you configure ClaudeToGo for monitoring")
	fmt.Println("Claude Code's tool usage through hooks.")
	fmt.Println()

	configFile := types.ConfigFile{
		LogFile:      "claude-events.log",
		PollInterval: "100ms",
		Verbose:      false,
	}

	fmt.Println("📋 Configuration Questions:")
	fmt.Println()
	fmt.Println("ClaudeToGo will log all Claude Code tool events for future analysis.")
	fmt.Println()

	// Ask about log file location
	fmt.Print("1. Where should events be logged? [claude-events.log]: ")
	var logFileInput string
	fmt.Scanln(&logFileInput)
	if logFileInput != "" {
		configFile.LogFile = logFileInput
	}
	fmt.Printf("✓ Events will be logged to: %s\n", configFile.LogFile)
	fmt.Println()

	// Ask about verbose logging
	fmt.Print("2. Enable verbose debug logging? [y/N]: ")
	var verboseInput string
	fmt.Scanln(&verboseInput)
	configFile.Verbose = strings.ToLower(verboseInput) == "y" || strings.ToLower(verboseInput) == "yes"
	if configFile.Verbose {
		fmt.Println("✓ Verbose logging enabled")
	} else {
		fmt.Println("✓ Normal logging level")
	}
	fmt.Println()

	// Save configuration
	configPath := "claudetogo-config.json"
	if err := config.Save(configFile, configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}
	fmt.Printf("✅ Configuration saved to: %s\n", configPath)
	fmt.Println()

	// Ask about Claude Code settings.json configuration
	fmt.Print("3. Would you like to automatically configure Claude Code hooks? [y/N]: ")
	var configureHooksInput string
	fmt.Scanln(&configureHooksInput)
	if strings.ToLower(configureHooksInput) == "y" || strings.ToLower(configureHooksInput) == "yes" {
		if err := configureHooks(configFile); err != nil {
			fmt.Printf("⚠️  Could not configure Claude Code hooks automatically: %v\n", err)
			fmt.Println("   You can configure them manually using the instructions below.")
		} else {
			fmt.Println("✅ Claude Code hooks configured successfully!")
		}
	} else {
		fmt.Println("✓ You can configure Claude Code hooks manually later")
	}
	fmt.Println()

	// Show usage examples
	ShowResults(configFile)

	return nil
}

// configureHooks automatically configures Claude Code settings.json
func configureHooks(config types.ConfigFile) error {
	// Ask user to choose configuration location
	location, err := chooseConfigLocation()
	if err != nil {
		return fmt.Errorf("failed to choose configuration location: %w", err)
	}

	return claude.ConfigureHooksAtLocation(config, location)
}

// chooseConfigLocation lets user choose between global and project configuration
func chooseConfigLocation() (*types.ConfigLocation, error) {
	fmt.Println("\n📁 Choose Claude Code Configuration Location:")
	fmt.Println("============================================")

	// Detect current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("could not get current directory: %w", err)
	}

	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not get user home directory: %w", err)
	}

	locations := []types.ConfigLocation{
		{
			Path:        filepath.Join(homeDir, ".claude", "settings.json"),
			Description: "Global configuration (affects all projects)",
			Scope:       "global",
		},
		{
			Path:        filepath.Join(cwd, ".claude", "settings.json"),
			Description: "Project configuration (shared with team, committed to repo)",
			Scope:       "project",
		},
		{
			Path:        filepath.Join(cwd, ".claude", "settings.local.json"),
			Description: "Local project configuration (personal, not committed)",
			Scope:       "local",
		},
	}

	// Show options
	for i, loc := range locations {
		existsMarker := ""
		if _, err := os.Stat(loc.Path); err == nil {
			existsMarker = " ✅ (exists)"
		}
		fmt.Printf("  [%d] %s%s\n", i+1, loc.Description, existsMarker)
		fmt.Printf("      Path: %s\n", loc.Path)
		fmt.Println()
	}

	fmt.Print("Choose location [1-3]: ")
	var choice string
	fmt.Scanln(&choice)

	switch choice {
	case "1":
		return &locations[0], nil
	case "2":
		return &locations[1], nil
	case "3":
		return &locations[2], nil
	default:
		fmt.Println("✓ Defaulting to global configuration")
		return &locations[0], nil
	}
}

// ShowResults displays the setup results and usage instructions
func ShowResults(config types.ConfigFile) {
	fmt.Println("🚀 Setup Complete! Here's how to use ClaudeToGo:")
	fmt.Println("================================================")
	fmt.Println()
	fmt.Println("ClaudeToGo is now configured to log all Claude Code tool events.")
	fmt.Println("This is Stage 1: Event collection for future analysis.")
	fmt.Println()

	// Show the command to run based on configuration
	var cmd strings.Builder
	cmd.WriteString("./claudetogo --hook")

	if config.LogFile != "claude-events.log" {
		cmd.WriteString(fmt.Sprintf(` --logfile "%s"`, config.LogFile))
	}

	if config.Verbose {
		cmd.WriteString(" --verbose")
	}

	fmt.Println("📝 To use as a Claude Code hook:")
	fmt.Printf("   %s\n", cmd.String())
	fmt.Println()

	fmt.Println("📊 To monitor events in real-time:")
	monitorCmd := "./claudetogo --monitor"
	if config.Verbose {
		monitorCmd += " --verbose"
	}
	if config.LogFile != "claude-events.log" {
		monitorCmd += fmt.Sprintf(` --logfile "%s"`, config.LogFile)
	}
	fmt.Printf("   %s\n", monitorCmd)
	fmt.Println()

	fmt.Println("⚙️ To configure Claude Code hooks manually:")
	fmt.Println("   1. Choose configuration location:")
	fmt.Println("      - Global: ~/.claude/settings.json")
	fmt.Println("      - Project: .claude/settings.json")
	fmt.Println("      - Local: .claude/settings.local.json")
	fmt.Println("   2. Add this hook configuration:")
	fmt.Println("   {")
	fmt.Println(`     "hooks": {`)
	fmt.Println(`       "Stop": [`)
	fmt.Println("         {")
	fmt.Println(`           "matcher": "*",`)
	fmt.Println(`           "hooks": [`)
	fmt.Println("             {")
	fmt.Println(`               "type": "command",`)
	fmt.Printf(`               "command": "%s",`+"\n", cmd.String())
	fmt.Println(`               "timeout": 30`)
	fmt.Println("             }")
	fmt.Println("           ]")
	fmt.Println("         }")
	fmt.Println("       ],")
	fmt.Println(`       "Notification": [`)
	fmt.Println("         {")
	fmt.Println(`           "matcher": "*",`)
	fmt.Println(`           "hooks": [`)
	fmt.Println("             {")
	fmt.Println(`               "type": "command",`)
	fmt.Printf(`               "command": "%s",`+"\n", cmd.String())
	fmt.Println(`               "timeout": 30`)
	fmt.Println("             }")
	fmt.Println("           ]")
	fmt.Println("         }")
	fmt.Println("       ]")
	fmt.Println("     }")
	fmt.Println("   }")
	fmt.Println()

	fmt.Println("💡 Tips:")
	fmt.Println("   - All tool events are logged and allowed (Stage 1: Event collection)")
	fmt.Println("   - Run with --help to see all available options")
	fmt.Println("   - Edit claudetogo-config.json to modify settings")
	fmt.Println("   - Use --setup again to reconfigure")
}