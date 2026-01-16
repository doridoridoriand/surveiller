package log

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// Level represents log level
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

var levelNames = map[Level]string{
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERROR",
}

// Logger provides structured logging
type Logger struct {
	level  Level
	output io.Writer
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// NewLogger creates a new logger with the specified level
func NewLogger(level Level) *Logger {
	return &Logger{
		level:  level,
		output: os.Stderr,
	}
}

// SetOutput sets the output writer for the logger
func (l *Logger) SetOutput(w io.Writer) {
	l.output = w
}

// SetLevel sets the log level
func (l *Logger) SetLevel(level Level) {
	l.level = level
}

// log writes a structured log entry
func (l *Logger) log(level Level, message string, fields map[string]interface{}) {
	if level < l.level {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     levelNames[level],
		Message:   message,
		Fields:    fields,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		// Fallback to plain text if JSON marshaling fails
		fmt.Fprintf(l.output, "[%s] %s: %s\n", entry.Timestamp, entry.Level, message)
		return
	}

	fmt.Fprintln(l.output, string(data))
}

// Debug logs a debug message
func (l *Logger) Debug(message string, fields map[string]interface{}) {
	l.log(LevelDebug, message, fields)
}

// Info logs an info message
func (l *Logger) Info(message string, fields map[string]interface{}) {
	l.log(LevelInfo, message, fields)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, fields map[string]interface{}) {
	l.log(LevelWarn, message, fields)
}

// Error logs an error message
func (l *Logger) Error(message string, fields map[string]interface{}) {
	l.log(LevelError, message, fields)
}

// LogPingResult logs a ping result
func (l *Logger) LogPingResult(targetName string, success bool, rtt time.Duration, err error) {
	fields := map[string]interface{}{
		"target":  targetName,
		"success": success,
		"rtt_ms":  rtt.Milliseconds(),
	}
	if err != nil {
		fields["error"] = err.Error()
	}

	if success {
		l.Info("ping result", fields)
	} else {
		l.Warn("ping failed", fields)
	}
}

// LogConfigLoad logs a config load event
func (l *Logger) LogConfigLoad(success bool, path string, err error) {
	fields := map[string]interface{}{
		"path": path,
	}
	if err != nil {
		fields["error"] = err.Error()
	}

	if success {
		l.Info("config loaded", fields)
	} else {
		l.Error("config load failed", fields)
	}
}

// LogError logs a general error
func (l *Logger) LogError(component string, err error, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["component"] = component
	if err != nil {
		fields["error"] = err.Error()
	}
	l.Error("error occurred", fields)
}

// ParseLevel parses a log level string
func ParseLevel(levelStr string) Level {
	switch levelStr {
	case "DEBUG", "debug":
		return LevelDebug
	case "INFO", "info":
		return LevelInfo
	case "WARN", "warn", "WARNING", "warning":
		return LevelWarn
	case "ERROR", "error":
		return LevelError
	default:
		return LevelInfo
	}
}
