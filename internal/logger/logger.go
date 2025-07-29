package logger

import "log"

// Logger provides structured logging with levels
type Logger struct {
	verbose bool
}

// New creates a new logger instance
func New(verbose bool) *Logger {
	return &Logger{verbose: verbose}
}

// Info logs an info level message
func (l *Logger) Info(msg string, args ...any) {
	log.Printf("[INFO] "+msg, args...)
}

// Error logs an error level message
func (l *Logger) Error(msg string, args ...any) {
	log.Printf("[ERROR] "+msg, args...)
}

// Debug logs a debug level message (only if verbose is enabled)
func (l *Logger) Debug(msg string, args ...any) {
	if l.verbose {
		log.Printf("[DEBUG] "+msg, args...)
	}
}