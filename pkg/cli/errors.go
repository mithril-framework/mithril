package cli

import (
	"fmt"
	"os"
)

// CLIError represents a CLI-specific error
type CLIError struct {
	Code    int
	Message string
	Err     error
}

// Error implements the error interface
func (e *CLIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *CLIError) Unwrap() error {
	return e.Err
}

// ExitCode returns the exit code for this error
func (e *CLIError) ExitCode() int {
	return e.Code
}

// NewCLIError creates a new CLI error
func NewCLIError(code int, message string, err error) *CLIError {
	return &CLIError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Error codes
const (
	ErrCodeSuccess = iota
	ErrCodeGeneral
	ErrCodeInvalidCommand
	ErrCodeInvalidArgument
	ErrCodeInvalidFlag
	ErrCodeMissingArgument
	ErrCodeMissingFlag
	ErrCodeFileNotFound
	ErrCodePermissionDenied
	ErrCodeConfiguration
	ErrCodeDatabase
	ErrCodeNetwork
	ErrCodeTimeout
	ErrCodeValidation
	ErrCodeAuthentication
	ErrCodeAuthorization
	ErrCodeNotFound
	ErrCodeAlreadyExists
	ErrCodeInvalidOperation
	ErrCodeDependency
	ErrCodeInternal
)

// Error messages
var (
	ErrInvalidCommand      = NewCLIError(ErrCodeInvalidCommand, "invalid command", nil)
	ErrInvalidArgument     = NewCLIError(ErrCodeInvalidArgument, "invalid argument", nil)
	ErrInvalidFlag         = NewCLIError(ErrCodeInvalidFlag, "invalid flag", nil)
	ErrMissingArgument     = NewCLIError(ErrCodeMissingArgument, "missing required argument", nil)
	ErrMissingFlag         = NewCLIError(ErrCodeMissingFlag, "missing required flag", nil)
	ErrFileNotFound        = NewCLIError(ErrCodeFileNotFound, "file not found", nil)
	ErrPermissionDenied    = NewCLIError(ErrCodePermissionDenied, "permission denied", nil)
	ErrConfiguration       = NewCLIError(ErrCodeConfiguration, "configuration error", nil)
	ErrDatabase            = NewCLIError(ErrCodeDatabase, "database error", nil)
	ErrNetwork             = NewCLIError(ErrCodeNetwork, "network error", nil)
	ErrTimeout             = NewCLIError(ErrCodeTimeout, "operation timeout", nil)
	ErrValidation          = NewCLIError(ErrCodeValidation, "validation error", nil)
	ErrAuthentication      = NewCLIError(ErrCodeAuthentication, "authentication error", nil)
	ErrAuthorization       = NewCLIError(ErrCodeAuthorization, "authorization error", nil)
	ErrNotFound            = NewCLIError(ErrCodeNotFound, "resource not found", nil)
	ErrAlreadyExists       = NewCLIError(ErrCodeAlreadyExists, "resource already exists", nil)
	ErrInvalidOperation    = NewCLIError(ErrCodeInvalidOperation, "invalid operation", nil)
	ErrDependency          = NewCLIError(ErrCodeDependency, "dependency error", nil)
	ErrInternal            = NewCLIError(ErrCodeInternal, "internal error", nil)
)

// ErrorHandler handles CLI errors
type ErrorHandler struct {
	verbose bool
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(verbose bool) *ErrorHandler {
	return &ErrorHandler{
		verbose: verbose,
	}
}

// Handle handles an error
func (eh *ErrorHandler) Handle(err error) {
	if err == nil {
		return
	}
	
	// Check if it's a CLI error
	if cliErr, ok := err.(*CLIError); ok {
		eh.handleCLIError(cliErr)
	} else {
		eh.handleGenericError(err)
	}
}

// handleCLIError handles a CLI-specific error
func (eh *ErrorHandler) handleCLIError(err *CLIError) {
	// Print error message
	fmt.Fprintf(os.Stderr, "Error: %s\n", err.Message)
	
	// Print underlying error if verbose and available
	if eh.verbose && err.Err != nil {
		fmt.Fprintf(os.Stderr, "Details: %v\n", err.Err)
	}
	
	// Print usage if it's a command/argument error
	if err.Code == ErrCodeInvalidCommand || err.Code == ErrCodeInvalidArgument || err.Code == ErrCodeMissingArgument {
		fmt.Fprintf(os.Stderr, "Use 'mithril help' for usage information\n")
	}
	
	// Exit with appropriate code
	os.Exit(err.ExitCode())
}

// handleGenericError handles a generic error
func (eh *ErrorHandler) handleGenericError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	
	if eh.verbose {
		fmt.Fprintf(os.Stderr, "Use 'mithril help' for usage information\n")
	}
	
	os.Exit(ErrCodeGeneral)
}

// ErrorFormatter formats error messages
type ErrorFormatter struct {
	colors bool
}

// NewErrorFormatter creates a new error formatter
func NewErrorFormatter(colors bool) *ErrorFormatter {
	return &ErrorFormatter{
		colors: colors,
	}
}

// FormatError formats an error message
func (ef *ErrorFormatter) FormatError(err error) string {
	if ef.colors {
		return fmt.Sprintf("\033[31mError:\033[0m %v", err)
	}
	return fmt.Sprintf("Error: %v", err)
}

// FormatWarning formats a warning message
func (ef *ErrorFormatter) FormatWarning(message string) string {
	if ef.colors {
		return fmt.Sprintf("\033[33mWarning:\033[0m %s", message)
	}
	return fmt.Sprintf("Warning: %s", message)
}

// FormatInfo formats an info message
func (ef *ErrorFormatter) FormatInfo(message string) string {
	if ef.colors {
		return fmt.Sprintf("\033[34mInfo:\033[0m %s", message)
	}
	return fmt.Sprintf("Info: %s", message)
}

// FormatSuccess formats a success message
func (ef *ErrorFormatter) FormatSuccess(message string) string {
	if ef.colors {
		return fmt.Sprintf("\033[32mSuccess:\033[0m %s", message)
	}
	return fmt.Sprintf("Success: %s", message)
}

// ErrorLogger logs errors
type ErrorLogger struct {
	verbose bool
}

// NewErrorLogger creates a new error logger
func NewErrorLogger(verbose bool) *ErrorLogger {
	return &ErrorLogger{
		verbose: verbose,
	}
}

// LogError logs an error
func (el *ErrorLogger) LogError(err error) {
	if el.verbose {
		fmt.Fprintf(os.Stderr, "Error logged: %v\n", err)
	}
}

// LogWarning logs a warning
func (el *ErrorLogger) LogWarning(message string) {
	if el.verbose {
		fmt.Fprintf(os.Stderr, "Warning logged: %s\n", message)
	}
}

// LogInfo logs an info message
func (el *ErrorLogger) LogInfo(message string) {
	if el.verbose {
		fmt.Fprintf(os.Stderr, "Info logged: %s\n", message)
	}
}

// ErrorRecovery provides error recovery functionality
type ErrorRecovery struct {
	handler *ErrorHandler
	logger  *ErrorLogger
}

// NewErrorRecovery creates a new error recovery
func NewErrorRecovery(verbose bool) *ErrorRecovery {
	return &ErrorRecovery{
		handler: NewErrorHandler(verbose),
		logger:  NewErrorLogger(verbose),
	}
}

// Recover recovers from panics
func (er *ErrorRecovery) Recover() {
	if r := recover(); r != nil {
		er.logger.LogError(fmt.Errorf("panic recovered: %v", r))
		er.handler.Handle(NewCLIError(ErrCodeInternal, "internal error", fmt.Errorf("panic: %v", r)))
	}
}

// HandleError handles an error with recovery
func (er *ErrorRecovery) HandleError(err error) {
	er.logger.LogError(err)
	er.handler.Handle(err)
}

// ErrorContext provides context for errors
type ErrorContext struct {
	Command   string
	Arguments []string
	Flags     map[string]interface{}
}

// NewErrorContext creates a new error context
func NewErrorContext(command string, arguments []string, flags map[string]interface{}) *ErrorContext {
	return &ErrorContext{
		Command:   command,
		Arguments: arguments,
		Flags:     flags,
	}
}

// AddErrorContext adds context to an error
func (ec *ErrorContext) AddErrorContext(err error) error {
	if cliErr, ok := err.(*CLIError); ok {
		cliErr.Message = fmt.Sprintf("%s (command: %s)", cliErr.Message, ec.Command)
		return cliErr
	}
	return fmt.Errorf("%s (command: %s): %w", err.Error(), ec.Command, err)
}

// ErrorValidation provides validation error functionality
type ErrorValidation struct {
	errors map[string][]string
}

// NewErrorValidation creates a new validation error
func NewErrorValidation() *ErrorValidation {
	return &ErrorValidation{
		errors: make(map[string][]string),
	}
}

// AddError adds a validation error
func (ev *ErrorValidation) AddError(field, message string) {
	ev.errors[field] = append(ev.errors[field], message)
}

// HasErrors checks if there are validation errors
func (ev *ErrorValidation) HasErrors() bool {
	return len(ev.errors) > 0
}

// GetErrors gets all validation errors
func (ev *ErrorValidation) GetErrors() map[string][]string {
	return ev.errors
}

// Error returns the validation error message
func (ev *ErrorValidation) Error() string {
	if !ev.HasErrors() {
		return ""
	}
	
	var message string
	for field, errors := range ev.errors {
		for _, err := range errors {
			if message != "" {
				message += "; "
			}
			message += fmt.Sprintf("%s: %s", field, err)
		}
	}
	
	return message
}

// ToCLIError converts validation errors to CLI error
func (ev *ErrorValidation) ToCLIError() *CLIError {
	if !ev.HasErrors() {
		return nil
	}
	
	return NewCLIError(ErrCodeValidation, ev.Error(), nil)
}
