package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// MessengerConfig represents the configuration for messenger integration
type MessengerConfig struct {
	Messenger   MessengerSettings   `yaml:"messenger"`
	Processing  ProcessingSettings  `yaml:"processing"`
	Service     ServiceSettings     `yaml:"service"`
	Formatting  FormattingSettings  `yaml:"formatting"`
	Integration IntegrationSettings `yaml:"integrations"`
}

// MessengerSettings contains messenger-specific configuration
type MessengerSettings struct {
	OutputDir     string `yaml:"output_dir"`
	FileFormat    string `yaml:"file_format"`
	IncludeSamples bool  `yaml:"include_samples"`
}

// ProcessingSettings contains event processing configuration
type ProcessingSettings struct {
	WatchMode          bool          `yaml:"watch_mode"`
	PollInterval       time.Duration `yaml:"poll_interval"`
	MaxEventsPerBatch  int           `yaml:"max_events_per_batch"`
	AutoProcess        bool          `yaml:"auto_process"`
	ProcessLatestOnly  int           `yaml:"process_latest_only"`
}

// ServiceSettings contains background service configuration
type ServiceSettings struct {
	Enabled        bool          `yaml:"enabled"`
	DaemonMode     bool          `yaml:"daemon_mode"`
	PidFile        string        `yaml:"pid_file"`
	LogLevel       string        `yaml:"log_level"`
	ServiceInterval time.Duration `yaml:"service_interval"`
	StatusFile     string        `yaml:"status_file"`
	AutoRestart    bool          `yaml:"auto_restart"`
}

// FormattingSettings contains message formatting configuration
type FormattingSettings struct {
	IncludeEmojis      bool `yaml:"include_emojis"`
	MaxMessageLength   int  `yaml:"max_message_length"`
	MaxContentPreview  int  `yaml:"max_content_preview"`
	TimestampFormat    string `yaml:"timestamp_format"`
	UseRelativeTime    bool `yaml:"use_relative_time"`
}

// IntegrationSettings contains external integration configuration
type IntegrationSettings struct {
	WebhookURL      string            `yaml:"webhook_url"`
	SlackToken      string            `yaml:"slack_token"`
	TelegramToken   string            `yaml:"telegram_token"`
	CustomHeaders   map[string]string `yaml:"custom_headers"`
	RetryAttempts   int               `yaml:"retry_attempts"`
	RetryInterval   time.Duration     `yaml:"retry_interval"`
	TimeoutDuration time.Duration     `yaml:"timeout_duration"`
}

// DefaultMessengerConfig returns a configuration with sensible defaults
func DefaultMessengerConfig() *MessengerConfig {
	return &MessengerConfig{
		Messenger: MessengerSettings{
			OutputDir:      "messenger-output",
			FileFormat:     "json",
			IncludeSamples: true,
		},
		Processing: ProcessingSettings{
			WatchMode:         false,
			PollInterval:      2 * time.Second,
			MaxEventsPerBatch: 10,
			AutoProcess:       false,
			ProcessLatestOnly: 0,
		},
		Service: ServiceSettings{
			Enabled:         false,
			DaemonMode:      false,
			PidFile:         "",
			LogLevel:        "info",
			ServiceInterval: 2 * time.Second,
			StatusFile:      "",
			AutoRestart:     false,
		},
		Formatting: FormattingSettings{
			IncludeEmojis:     true,
			MaxMessageLength:  1000,
			MaxContentPreview: 200,
			TimestampFormat:   "2006-01-02 15:04:05",
			UseRelativeTime:   false,
		},
		Integration: IntegrationSettings{
			WebhookURL:      "",
			SlackToken:      "",
			TelegramToken:   "",
			CustomHeaders:   make(map[string]string),
			RetryAttempts:   3,
			RetryInterval:   1 * time.Second,
			TimeoutDuration: 30 * time.Second,
		},
	}
}

// LoadMessengerConfig loads messenger configuration from a YAML file
func LoadMessengerConfig(configPath string) (*MessengerConfig, error) {
	// Check if file exists
	if !fileExists(configPath) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	// Read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Start with defaults
	config := DefaultMessengerConfig()

	// Parse YAML and merge with defaults
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// SaveMessengerConfig saves messenger configuration to a YAML file
func SaveMessengerConfig(config *MessengerConfig, configPath string) error {
	// Validate configuration before saving
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate checks if the configuration is valid
func (mc *MessengerConfig) Validate() error {
	// Validate messenger settings
	if mc.Messenger.OutputDir == "" {
		return fmt.Errorf("messenger.output_dir cannot be empty")
	}

	if mc.Messenger.FileFormat != "json" && mc.Messenger.FileFormat != "jsonl" {
		return fmt.Errorf("messenger.file_format must be 'json' or 'jsonl'")
	}

	// Validate processing settings
	if mc.Processing.PollInterval < 100*time.Millisecond {
		return fmt.Errorf("processing.poll_interval must be at least 100ms")
	}

	if mc.Processing.MaxEventsPerBatch < 1 {
		return fmt.Errorf("processing.max_events_per_batch must be at least 1")
	}

	// Validate service settings
	if mc.Service.ServiceInterval < 100*time.Millisecond {
		return fmt.Errorf("service.service_interval must be at least 100ms")
	}

	validLogLevels := []string{"debug", "info", "warn", "error"}
	validLogLevel := false
	for _, level := range validLogLevels {
		if mc.Service.LogLevel == level {
			validLogLevel = true
			break
		}
	}
	if !validLogLevel {
		return fmt.Errorf("service.log_level must be one of: debug, info, warn, error")
	}

	// Validate formatting settings
	if mc.Formatting.MaxMessageLength < 100 {
		return fmt.Errorf("formatting.max_message_length must be at least 100")
	}

	if mc.Formatting.MaxContentPreview < 50 {
		return fmt.Errorf("formatting.max_content_preview must be at least 50")
	}

	// Validate integration settings
	if mc.Integration.RetryAttempts < 0 {
		return fmt.Errorf("integrations.retry_attempts must be non-negative")
	}

	if mc.Integration.RetryInterval < 100*time.Millisecond {
		return fmt.Errorf("integrations.retry_interval must be at least 100ms")
	}

	if mc.Integration.TimeoutDuration < 1*time.Second {
		return fmt.Errorf("integrations.timeout_duration must be at least 1 second")
	}

	return nil
}

// GenerateExampleConfig creates an example configuration file with comments
func GenerateExampleConfig(configPath string) error {
	exampleYAML := `# ClaudeToGo Messenger Configuration
# This file configures the messenger integration features of ClaudeToGo

# Messenger output settings
messenger:
  output_dir: "messenger-output"     # Directory for generated JSON files
  file_format: "json"                # Output format: "json" or "jsonl"
  include_samples: true              # Generate sample files for testing

# Event processing settings
processing:
  watch_mode: false                  # Enable automatic file watching
  poll_interval: "2s"                # How often to check for new events
  max_events_per_batch: 10           # Maximum events to process at once
  auto_process: false                # Automatically process new events
  process_latest_only: 0             # Only process N latest events (0 = all)

# Background service settings
service:
  enabled: false                     # Enable background service mode
  daemon_mode: false                 # Run as daemon (background process)
  pid_file: ""                       # PID file location (empty = auto)
  log_level: "info"                  # Log level: debug, info, warn, error
  service_interval: "2s"             # Service check interval
  status_file: ""                    # Status file location (empty = auto)
  auto_restart: false                # Automatically restart on failure

# Message formatting settings
formatting:
  include_emojis: true               # Include emojis in messages
  max_message_length: 1000           # Maximum message length
  max_content_preview: 200           # Maximum content preview length
  timestamp_format: "2006-01-02 15:04:05"  # Timestamp format
  use_relative_time: false           # Use relative timestamps (e.g., "2 hours ago")

# External integration settings
integrations:
  webhook_url: ""                    # HTTP webhook URL for notifications
  slack_token: ""                    # Slack bot token
  telegram_token: ""                 # Telegram bot token
  custom_headers: {}                 # Custom HTTP headers for webhooks
  retry_attempts: 3                  # Number of retry attempts
  retry_interval: "1s"               # Interval between retries
  timeout_duration: "30s"            # Request timeout duration
`

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write example config
	if err := os.WriteFile(configPath, []byte(exampleYAML), 0644); err != nil {
		return fmt.Errorf("failed to write example config: %w", err)
	}

	return nil
}

// FindMessengerConfig searches for messenger config files in common locations
func FindMessengerConfig() string {
	commonPaths := []string{
		"claudetogo-messenger.yaml",
		"claudetogo-messenger.yml",
		".claudetogo/messenger.yaml",
		".claudetogo/messenger.yml",
		"config/claudetogo-messenger.yaml",
		"config/claudetogo-messenger.yml",
	}

	for _, path := range commonPaths {
		if fileExists(path) {
			return path
		}
	}

	return ""
}

// GetMessengerConfigWithDefaults loads config from file or returns defaults
func GetMessengerConfigWithDefaults(configPath string) *MessengerConfig {
	if configPath != "" && fileExists(configPath) {
		if config, err := LoadMessengerConfig(configPath); err == nil {
			return config
		}
	}

	// Try to find config automatically
	if foundConfig := FindMessengerConfig(); foundConfig != "" {
		if config, err := LoadMessengerConfig(foundConfig); err == nil {
			return config
		}
	}

	// Return defaults if no config found
	return DefaultMessengerConfig()
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// ApplyEnvironmentOverrides applies environment variable overrides to config
func (mc *MessengerConfig) ApplyEnvironmentOverrides() {
	// Apply environment variable overrides
	if outputDir := os.Getenv("CLAUDETOGO_OUTPUT_DIR"); outputDir != "" {
		mc.Messenger.OutputDir = outputDir
	}

	if logLevel := os.Getenv("CLAUDETOGO_LOG_LEVEL"); logLevel != "" {
		mc.Service.LogLevel = logLevel
	}

	if webhookURL := os.Getenv("CLAUDETOGO_WEBHOOK_URL"); webhookURL != "" {
		mc.Integration.WebhookURL = webhookURL
	}

	if slackToken := os.Getenv("CLAUDETOGO_SLACK_TOKEN"); slackToken != "" {
		mc.Integration.SlackToken = slackToken
	}

	if telegramToken := os.Getenv("CLAUDETOGO_TELEGRAM_TOKEN"); telegramToken != "" {
		mc.Integration.TelegramToken = telegramToken
	}
}

// Summary returns a human-readable summary of the configuration
func (mc *MessengerConfig) Summary() string {
	summary := fmt.Sprintf(`ClaudeToGo Messenger Configuration Summary:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📁 Output Directory:    %s
📋 File Format:         %s
⏱️  Poll Interval:       %v
🔄 Watch Mode:          %t
🚀 Service Mode:        %t
📊 Max Events/Batch:    %d
🎨 Include Emojis:      %t
📏 Max Message Length:  %d
🔗 Webhook URL:         %s
🤖 Slack Integration:   %t
📱 Telegram Integration: %t
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━`,
		mc.Messenger.OutputDir,
		mc.Messenger.FileFormat,
		mc.Processing.PollInterval,
		mc.Processing.WatchMode,
		mc.Service.Enabled,
		mc.Processing.MaxEventsPerBatch,
		mc.Formatting.IncludeEmojis,
		mc.Formatting.MaxMessageLength,
		mc.Integration.WebhookURL,
		mc.Integration.SlackToken != "",
		mc.Integration.TelegramToken != "",
	)

	return summary
}