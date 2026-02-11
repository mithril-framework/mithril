package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

// OutputProvider provides output functionality for CLI commands
type OutputProvider struct {
	writer  io.Writer
	colors  bool
	verbose bool
}

// NewOutputProvider creates a new output provider
func NewOutputProvider() *OutputProvider {
	return &OutputProvider{
		writer:  os.Stdout,
		colors:  true,
		verbose: false,
	}
}

// SetWriter sets the output writer
func (op *OutputProvider) SetWriter(writer io.Writer) {
	op.writer = writer
}

// SetColors enables or disables colors
func (op *OutputProvider) SetColors(colors bool) {
	op.colors = colors
}

// SetVerbose enables or disables verbose output
func (op *OutputProvider) SetVerbose(verbose bool) {
	op.verbose = verbose
}

// Print prints a message
func (op *OutputProvider) Print(message string) {
	fmt.Fprint(op.writer, message)
}

// Println prints a message with a newline
func (op *OutputProvider) Println(message string) {
	fmt.Fprintln(op.writer, message)
}

// Printf prints a formatted message
func (op *OutputProvider) Printf(format string, args ...interface{}) {
	fmt.Fprintf(op.writer, format, args...)
}

// Info prints an info message
func (op *OutputProvider) Info(message string) {
	if op.colors {
		fmt.Fprintf(op.writer, "\033[34m[INFO]\033[0m %s\n", message)
	} else {
		fmt.Fprintf(op.writer, "[INFO] %s\n", message)
	}
}

// Success prints a success message
func (op *OutputProvider) Success(message string) {
	if op.colors {
		fmt.Fprintf(op.writer, "\033[32m[SUCCESS]\033[0m %s\n", message)
	} else {
		fmt.Fprintf(op.writer, "[SUCCESS] %s\n", message)
	}
}

// Warning prints a warning message
func (op *OutputProvider) Warning(message string) {
	if op.colors {
		fmt.Fprintf(op.writer, "\033[33m[WARNING]\033[0m %s\n", message)
	} else {
		fmt.Fprintf(op.writer, "[WARNING] %s\n", message)
	}
}

// Error prints an error message
func (op *OutputProvider) Error(message string) {
	if op.colors {
		fmt.Fprintf(op.writer, "\033[31m[ERROR]\033[0m %s\n", message)
	} else {
		fmt.Fprintf(op.writer, "[ERROR] %s\n", message)
	}
}

// Debug prints a debug message
func (op *OutputProvider) Debug(message string) {
	if op.verbose {
		if op.colors {
			fmt.Fprintf(op.writer, "\033[36m[DEBUG]\033[0m %s\n", message)
		} else {
			fmt.Fprintf(op.writer, "[DEBUG] %s\n", message)
		}
	}
}

// Verbose prints a verbose message
func (op *OutputProvider) Verbose(message string) {
	if op.verbose {
		if op.colors {
			fmt.Fprintf(op.writer, "\033[90m[VERBOSE]\033[0m %s\n", message)
		} else {
			fmt.Fprintf(op.writer, "[VERBOSE] %s\n", message)
		}
	}
}

// Table prints a table
func (op *OutputProvider) Table(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(op.writer, 0, 0, 2, ' ', 0)
	defer w.Flush()
	
	// Print headers
	for i, header := range headers {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, header)
	}
	fmt.Fprintln(w)
	
	// Print separator
	for i, header := range headers {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, strings.Repeat("-", len(header)))
	}
	fmt.Fprintln(w)
	
	// Print rows
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				fmt.Fprint(w, "\t")
			}
			fmt.Fprint(w, cell)
		}
		fmt.Fprintln(w)
	}
}

// List prints a list
func (op *OutputProvider) List(items []string) {
	for _, item := range items {
		fmt.Fprintf(op.writer, "  • %s\n", item)
	}
}

// NumberedList prints a numbered list
func (op *OutputProvider) NumberedList(items []string) {
	for i, item := range items {
		fmt.Fprintf(op.writer, "  %d. %s\n", i+1, item)
	}
}

// Progress prints a progress bar
func (op *OutputProvider) Progress(current, total int, message string) {
	if total == 0 {
		return
	}
	
	percentage := float64(current) / float64(total) * 100
	barWidth := 50
	filledWidth := int(percentage / 100 * float64(barWidth))
	
	bar := strings.Repeat("=", filledWidth) + strings.Repeat("-", barWidth-filledWidth)
	
	if op.colors {
		fmt.Fprintf(op.writer, "\r\033[32m[%s]\033[0m %s (%.1f%%)", bar, message, percentage)
	} else {
		fmt.Fprintf(op.writer, "\r[%s] %s (%.1f%%)", bar, message, percentage)
	}
	
	if current == total {
		fmt.Fprintln(op.writer)
	}
}

// Spinner prints a spinner
func (op *OutputProvider) Spinner(message string) {
	spinnerChars := []string{"|", "/", "-", "\\"}
	for i := 0; i < 10; i++ {
		char := spinnerChars[i%len(spinnerChars)]
		if op.colors {
			fmt.Fprintf(op.writer, "\r\033[36m[%s]\033[0m %s", char, message)
		} else {
			fmt.Fprintf(op.writer, "\r[%s] %s", char, message)
		}
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Fprintln(op.writer)
}

// Section prints a section header
func (op *OutputProvider) Section(title string) {
	fmt.Fprintln(op.writer)
	if op.colors {
		fmt.Fprintf(op.writer, "\033[1m%s\033[0m\n", title)
	} else {
		fmt.Fprintf(op.writer, "%s\n", title)
	}
	fmt.Fprintln(op.writer, strings.Repeat("=", len(title)))
}

// Subsection prints a subsection header
func (op *OutputProvider) Subsection(title string) {
	fmt.Fprintln(op.writer)
	if op.colors {
		fmt.Fprintf(op.writer, "\033[1m%s\033[0m\n", title)
	} else {
		fmt.Fprintf(op.writer, "%s\n", title)
	}
	fmt.Fprintln(op.writer, strings.Repeat("-", len(title)))
}

// Code prints code with syntax highlighting
func (op *OutputProvider) Code(code string, language string) {
	fmt.Fprintln(op.writer)
	if op.colors {
		fmt.Fprintf(op.writer, "\033[90m```%s\033[0m\n", language)
		fmt.Fprintf(op.writer, "\033[37m%s\033[0m\n", code)
		fmt.Fprintf(op.writer, "\033[90m```\033[0m\n")
	} else {
		fmt.Fprintf(op.writer, "```%s\n", language)
		fmt.Fprintf(op.writer, "%s\n", code)
		fmt.Fprintf(op.writer, "```\n")
	}
}

// Quote prints a quote
func (op *OutputProvider) Quote(quote string, author string) {
	fmt.Fprintln(op.writer)
	if op.colors {
		fmt.Fprintf(op.writer, "\033[90m\"%s\"\033[0m\n", quote)
		if author != "" {
			fmt.Fprintf(op.writer, "\033[90m— %s\033[0m\n", author)
		}
	} else {
		fmt.Fprintf(op.writer, "\"%s\"\n", quote)
		if author != "" {
			fmt.Fprintf(op.writer, "— %s\n", author)
		}
	}
}

// Separator prints a separator line
func (op *OutputProvider) Separator() {
	fmt.Fprintln(op.writer, strings.Repeat("-", 80))
}

// Newline prints a newline
func (op *OutputProvider) Newline() {
	fmt.Fprintln(op.writer)
}

// Clear clears the screen
func (op *OutputProvider) Clear() {
	if op.colors {
		fmt.Fprint(op.writer, "\033[2J\033[H")
	} else {
		// Fallback for non-color terminals
		fmt.Fprint(op.writer, "\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n")
	}
}

// OutputFormatter formats output
type OutputFormatter struct {
	provider *OutputProvider
}

// NewOutputFormatter creates a new output formatter
func NewOutputFormatter() *OutputFormatter {
	return &OutputFormatter{
		provider: NewOutputProvider(),
	}
}

// FormatCommandOutput formats command output
func (of *OutputFormatter) FormatCommandOutput(command string, output string) string {
	var formatted strings.Builder
	
	formatted.WriteString(fmt.Sprintf("Command: %s\n", command))
	formatted.WriteString(strings.Repeat("=", len(command)+9))
	formatted.WriteString("\n")
	formatted.WriteString(output)
	formatted.WriteString("\n")
	
	return formatted.String()
}

// FormatErrorOutput formats error output
func (of *OutputFormatter) FormatErrorOutput(error string) string {
	var formatted strings.Builder
	
	formatted.WriteString("Error:\n")
	formatted.WriteString(strings.Repeat("=", 6))
	formatted.WriteString("\n")
	formatted.WriteString(error)
	formatted.WriteString("\n")
	
	return formatted.String()
}

// FormatSuccessOutput formats success output
func (of *OutputFormatter) FormatSuccessOutput(message string) string {
	var formatted strings.Builder
	
	formatted.WriteString("Success:\n")
	formatted.WriteString(strings.Repeat("=", 8))
	formatted.WriteString("\n")
	formatted.WriteString(message)
	formatted.WriteString("\n")
	
	return formatted.String()
}

// OutputLogger logs output
type OutputLogger struct {
	provider *OutputProvider
	logFile  string
}

// NewOutputLogger creates a new output logger
func NewOutputLogger(logFile string) *OutputLogger {
	return &OutputLogger{
		provider: NewOutputProvider(),
		logFile:  logFile,
	}
}

// Log logs a message
func (ol *OutputLogger) Log(level, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logMessage := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level, message)
	
	// Print to console
	ol.provider.Print(logMessage)
	
	// Write to log file
	if ol.logFile != "" {
		file, err := os.OpenFile(ol.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			defer file.Close()
			_, _ = file.WriteString(logMessage)
		}
	}
}

// LogInfo logs an info message
func (ol *OutputLogger) LogInfo(message string) {
	ol.Log("INFO", message)
}

// LogSuccess logs a success message
func (ol *OutputLogger) LogSuccess(message string) {
	ol.Log("SUCCESS", message)
}

// LogWarning logs a warning message
func (ol *OutputLogger) LogWarning(message string) {
	ol.Log("WARNING", message)
}

// LogError logs an error message
func (ol *OutputLogger) LogError(message string) {
	ol.Log("ERROR", message)
}

// LogDebug logs a debug message
func (ol *OutputLogger) LogDebug(message string) {
	ol.Log("DEBUG", message)
}

// OutputManager manages output
type OutputManager struct {
	provider  *OutputProvider
	formatter *OutputFormatter
	logger    *OutputLogger
}

// NewOutputManager creates a new output manager
func NewOutputManager() *OutputManager {
	return &OutputManager{
		provider:  NewOutputProvider(),
		formatter: NewOutputFormatter(),
		logger:    NewOutputLogger(""),
	}
}

// SetLogFile sets the log file
func (om *OutputManager) SetLogFile(logFile string) {
	om.logger = NewOutputLogger(logFile)
}

// SetVerbose sets verbose mode
func (om *OutputManager) SetVerbose(verbose bool) {
	om.provider.SetVerbose(verbose)
}

// SetColors sets color mode
func (om *OutputManager) SetColors(colors bool) {
	om.provider.SetColors(colors)
}

// Print prints a message
func (om *OutputManager) Print(message string) {
	om.provider.Print(message)
}

// Println prints a message with newline
func (om *OutputManager) Println(message string) {
	om.provider.Println(message)
}

// Printf prints a formatted message
func (om *OutputManager) Printf(format string, args ...interface{}) {
	om.provider.Printf(format, args...)
}

// Info prints an info message
func (om *OutputManager) Info(message string) {
	om.provider.Info(message)
	om.logger.LogInfo(message)
}

// Success prints a success message
func (om *OutputManager) Success(message string) {
	om.provider.Success(message)
	om.logger.LogSuccess(message)
}

// Warning prints a warning message
func (om *OutputManager) Warning(message string) {
	om.provider.Warning(message)
	om.logger.LogWarning(message)
}

// Error prints an error message
func (om *OutputManager) Error(message string) {
	om.provider.Error(message)
	om.logger.LogError(message)
}

// Debug prints a debug message
func (om *OutputManager) Debug(message string) {
	om.provider.Debug(message)
	om.logger.LogDebug(message)
}

// Table prints a table
func (om *OutputManager) Table(headers []string, rows [][]string) {
	om.provider.Table(headers, rows)
}

// List prints a list
func (om *OutputManager) List(items []string) {
	om.provider.List(items)
}

// NumberedList prints a numbered list
func (om *OutputManager) NumberedList(items []string) {
	om.provider.NumberedList(items)
}

// Progress prints a progress bar
func (om *OutputManager) Progress(current, total int, message string) {
	om.provider.Progress(current, total, message)
}

// Spinner prints a spinner
func (om *OutputManager) Spinner(message string) {
	om.provider.Spinner(message)
}

// Section prints a section header
func (om *OutputManager) Section(title string) {
	om.provider.Section(title)
}

// Subsection prints a subsection header
func (om *OutputManager) Subsection(title string) {
	om.provider.Subsection(title)
}

// Code prints code
func (om *OutputManager) Code(code string, language string) {
	om.provider.Code(code, language)
}

// Quote prints a quote
func (om *OutputManager) Quote(quote string, author string) {
	om.provider.Quote(quote, author)
}

// Separator prints a separator
func (om *OutputManager) Separator() {
	om.provider.Separator()
}

// Newline prints a newline
func (om *OutputManager) Newline() {
	om.provider.Newline()
}

// Clear clears the screen
func (om *OutputManager) Clear() {
	om.provider.Clear()
}
