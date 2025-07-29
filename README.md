# ClaudeToGo

A Go-based tool for logging and monitoring Claude Code hook events. ClaudeToGo intercepts and logs all tool usage events from Claude Code, providing insights into AI assistant interactions and tool usage patterns.

## 🎯 Purpose

ClaudeToGo serves as a monitoring and logging solution for Claude Code hooks, currently implementing **Stage 1: Event Collection**. It captures all tool events for future analysis while allowing all operations to proceed normally.

## ✨ Features

- **Event Logging**: Captures all Claude Code tool events with detailed metadata
- **Real-time Monitoring**: Live monitoring of events as they occur
- **Interactive Setup**: Guided setup wizard for easy configuration
- **Automatic Hook Configuration**: Seamlessly integrates with Claude Code settings
- **Flexible Configuration**: JSON-based configuration with command-line overrides
- **Graceful Shutdown**: Proper signal handling for clean exits

## 🏗️ Architecture

The project follows Go best practices with a modular architecture:

```
claudetogo/
├── cmd/claudetogo/          # Application entry point
├── internal/
│   ├── types/              # Data structures and models
│   ├── logger/             # Structured logging
│   ├── config/             # Configuration management
│   ├── hooks/              # Hook processing logic
│   ├── monitor/            # Event monitoring
│   ├── setup/              # Setup wizard
│   └── claude/             # Claude Code settings management
├── go.mod                  # Go module definition
└── README.md               # This file
```

## 🚀 Getting Started

### Prerequisites

- Go 1.22.2 or later
- Claude Code installed and configured

### Installation

1. **Clone the repository:**
   ```bash
   git clone https://github.com/riaanpieterse81/ClaudeToGo.git
   cd ClaudeToGo
   ```

2. **Build the application:**
   ```bash
   go build -o claudetogo ./cmd/claudetogo
   ```

3. **Run the setup wizard (recommended for first-time users):**
   ```bash
   ./claudetogo --setup
   ```

### Quick Start

The setup wizard will guide you through:
- Configuring event log location
- Setting verbosity level  
- Automatically configuring Claude Code hooks
- Displaying usage instructions

## 📖 Usage

### Command Line Options

```bash
Usage: claudetogo [options]

Options:
  -config string
        Path to configuration file (JSON format)
  -help
        Show help information
  -hook
        Process hook event from stdin (for Claude Code hooks)
  -logfile string
        Path to log file (default "claude-events.jsonl")
  -monitor
        Monitor events in real-time
  -poll-interval duration
        Polling interval for monitoring (default 100ms)
  -setup
        Run interactive setup wizard
  -verbose
        Enable verbose debug output
```

### Common Commands

**Setup and Configuration:**
```bash
./claudetogo --setup                     # Run setup wizard
./claudetogo --help                      # Show help
```

**Hook Processing (used by Claude Code):**
```bash
./claudetogo --hook                      # Process hook event from stdin
./claudetogo --hook --verbose            # Process with debug output
```

**Monitoring:**
```bash
./claudetogo --monitor                   # Monitor events in real-time
./claudetogo --monitor --verbose         # Monitor with debug output
```

**Custom Configuration:**
```bash
./claudetogo --config myconfig.json      # Use custom config file
./claudetogo --logfile custom.log        # Use custom log file
```

## ⚙️ Configuration

### Configuration File

ClaudeToGo uses JSON configuration files. The setup wizard creates `claudetogo-config.json`:

```json
{
  "logFile": "claude-events.jsonl",
  "pollInterval": "100ms",
  "verbose": false
}
```

### Claude Code Integration

The tool integrates with Claude Code through hooks configured in Claude's `settings.json`:

**Configuration Locations:**
- **Global**: `~/.claude/settings.json`
- **Project**: `.claude/settings.json` 
- **Local**: `.claude/settings.local.json`

**Hook Configuration:**
```json
{
  "hooks": {
    "Stop": [
      {
        "matcher": "*",
        "hooks": [
          {
            "type": "command",
            "command": "./claudetogo --hook",
            "timeout": 30
          }
        ]
      }
    ],
    "Notification": [
      {
        "matcher": "*", 
        "hooks": [
          {
            "type": "command",
            "command": "./claudetogo --hook",
            "timeout": 30
          }
        ]
      }
    ]
  }
}
```

## 📊 Event Logging

Events are logged in JSON format to `claude-events.jsonl` (or your configured log file):

```json
{
  "session_id": "39b32221-a660-4c59-b515-b4be18909c3c",
  "transcript_path": "/path/to/transcript.jsonl",
  "cwd": "/current/working/directory",
  "hook_event_name": "Notification",
  "tool_name": "Bash",
  "timestamp": "2024-07-29T14:35:49Z",
  "message": "Claude needs your permission to use Bash"
}
```

## 🔧 Development

### Building from Source

```bash
# Build for current platform
go build -o claudetogo ./cmd/claudetogo

# Build for specific platform
GOOS=linux GOARCH=amd64 go build -o claudetogo-linux ./cmd/claudetogo
GOOS=windows GOARCH=amd64 go build -o claudetogo.exe ./cmd/claudetogo
GOOS=darwin GOARCH=amd64 go build -o claudetogo-mac ./cmd/claudetogo
```

### Testing

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...
```

### Project Structure

- **`cmd/claudetogo/`**: Main application entry point
- **`internal/types/`**: Core data structures and types
- **`internal/logger/`**: Structured logging utilities
- **`internal/config/`**: Configuration loading and management
- **`internal/hooks/`**: Hook event processing logic
- **`internal/monitor/`**: Real-time event monitoring
- **`internal/setup/`**: Interactive setup wizard
- **`internal/claude/`**: Claude Code settings management

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built for integration with [Claude Code](https://claude.ai/code)
- Follows [Go project layout standards](https://go.dev/doc/modules/layout)

## 📞 Support

If you encounter any issues or have questions:

1. Check the [Issues](https://github.com/riaanpieterse81/ClaudeToGo/issues) page
2. Create a new issue with detailed information
3. Include log files and configuration when reporting bugs

## 🚀 Roadmap

**Current Stage: Event Collection**
- ✅ Hook event logging
- ✅ Real-time monitoring
- ✅ Automatic Claude Code configuration

**Future Stages:**
- 🔄 Event analysis and reporting
- 🔄 Advanced filtering and rules
- 🔄 Web dashboard for visualization
- 🔄 Integration with other tools