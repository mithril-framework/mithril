package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// LogLevel represents the log level
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarning
	LogLevelError
	LogLevelFatal
)

// String returns the string representation of the log level
func (ll LogLevel) String() string {
	switch ll {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarning:
		return "WARNING"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Color returns the color code for the log level
func (ll LogLevel) Color() string {
	switch ll {
	case LogLevelDebug:
		return "\033[36m" // Cyan
	case LogLevelInfo:
		return "\033[34m" // Blue
	case LogLevelWarning:
		return "\033[33m" // Yellow
	case LogLevelError:
		return "\033[31m" // Red
	case LogLevelFatal:
		return "\033[35m" // Magenta
	default:
		return "\033[0m" // Reset
	}
}

// Logger provides logging functionality for CLI commands
type Logger struct {
	level    LogLevel
	writer   io.Writer
	colors   bool
	verbose  bool
	logFile  string
	file     *os.File
}

// NewLogger creates a new logger
func NewLogger() *Logger {
	return &Logger{
		level:   LogLevelInfo,
		writer:  os.Stdout,
		colors:  true,
		verbose: false,
	}
}

// SetLevel sets the log level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// SetWriter sets the output writer
func (l *Logger) SetWriter(writer io.Writer) {
	l.writer = writer
}

// SetColors enables or disables colors
func (l *Logger) SetColors(colors bool) {
	l.colors = colors
}

// SetVerbose enables or disables verbose output
func (l *Logger) SetVerbose(verbose bool) {
	l.verbose = verbose
}

// SetLogFile sets the log file
func (l *Logger) SetLogFile(logFile string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(logFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}
	
	// Open log file
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	
	// Close previous file if open
	if l.file != nil {
		l.file.Close()
	}
	
	l.file = file
	l.logFile = logFile
	return nil
}

// Close closes the logger
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// log logs a message
func (l *Logger) log(level LogLevel, message string) {
	if level < l.level {
		return
	}
	
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	
	var logMessage string
	if l.colors {
		logMessage = fmt.Sprintf("%s[%s] %s[%s]\033[0m %s\n", 
			l.level.Color(), timestamp, l.level.Color(), level.String(), message)
	} else {
		logMessage = fmt.Sprintf("[%s] [%s] %s\n", timestamp, level.String(), message)
	}
	
	// Write to console
	_, _ = l.writer.Write([]byte(logMessage))
	
	// Write to log file if set
	if l.file != nil {
		fileMessage := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level.String(), message)
		_, _ = l.file.WriteString(fileMessage)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(message string) {
	l.log(LogLevelDebug, message)
}

// Info logs an info message
func (l *Logger) Info(message string) {
	l.log(LogLevelInfo, message)
}

// Warning logs a warning message
func (l *Logger) Warning(message string) {
	l.log(LogLevelWarning, message)
}

// Error logs an error message
func (l *Logger) Error(message string) {
	l.log(LogLevelError, message)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(message string) {
	l.log(LogLevelFatal, message)
	os.Exit(1)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Debug(fmt.Sprintf(format, args...))
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Info(fmt.Sprintf(format, args...))
}

// Warningf logs a formatted warning message
func (l *Logger) Warningf(format string, args ...interface{}) {
	l.Warning(fmt.Sprintf(format, args...))
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Error(fmt.Sprintf(format, args...))
}

// Fatalf logs a formatted fatal message and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Fatal(fmt.Sprintf(format, args...))
}