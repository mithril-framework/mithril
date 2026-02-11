package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Validator provides validation functionality for CLI commands
type Validator struct {
	rules map[string]ValidationRule
}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{
		rules: make(map[string]ValidationRule),
	}
}

// ValidationRule defines a validation rule
type ValidationRule struct {
	Required bool
	Min      int
	Max      int
	Pattern  string
	Custom   func(interface{}) error
}

// AddRule adds a validation rule
func (v *Validator) AddRule(field string, rule ValidationRule) {
	v.rules[field] = rule
}

// Validate validates a field
func (v *Validator) Validate(field string, value interface{}) error {
	rule, exists := v.rules[field]
	if !exists {
		return nil
	}
	
	// Check required
	if rule.Required && isEmpty(value) {
		return fmt.Errorf("field '%s' is required", field)
	}
	
	// Skip other validations if value is empty and not required
	if isEmpty(value) && !rule.Required {
		return nil
	}
	
	// Check min/max for strings and slices
	if str, ok := value.(string); ok {
		if rule.Min > 0 && len(str) < rule.Min {
			return fmt.Errorf("field '%s' must be at least %d characters", field, rule.Min)
		}
		if rule.Max > 0 && len(str) > rule.Max {
			return fmt.Errorf("field '%s' must be at most %d characters", field, rule.Max)
		}
	}
	
	// Check pattern
	if rule.Pattern != "" {
		if str, ok := value.(string); ok {
			matched, err := regexp.MatchString(rule.Pattern, str)
			if err != nil {
				return fmt.Errorf("invalid pattern for field '%s': %v", field, err)
			}
			if !matched {
				return fmt.Errorf("field '%s' does not match required pattern", field)
			}
		}
	}
	
	// Check custom validation
	if rule.Custom != nil {
		if err := rule.Custom(value); err != nil {
			return fmt.Errorf("field '%s': %v", field, err)
		}
	}
	
	return nil
}

// ValidateAll validates all fields
func (v *Validator) ValidateAll(data map[string]interface{}) error {
	validation := NewErrorValidation()
	
	for field, value := range data {
		if err := v.Validate(field, value); err != nil {
			validation.AddError(field, err.Error())
		}
	}
	
	if validation.HasErrors() {
		return validation.ToCLIError()
	}
	
	return nil
}

// isEmpty checks if a value is empty
func isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}
	
	switch v := value.(type) {
	case string:
		return v == ""
	case []string:
		return len(v) == 0
	case int:
		return v == 0
	case bool:
		return !v
	default:
		return false
	}
}

// Common validation rules
var (
	// String validations
	RequiredString = ValidationRule{Required: true}
	OptionalString = ValidationRule{Required: false}
	MinString      = func(min int) ValidationRule { return ValidationRule{Min: min} }
	MaxString      = func(max int) ValidationRule { return ValidationRule{Max: max} }
	MinMaxString   = func(min, max int) ValidationRule { return ValidationRule{Min: min, Max: max} }
	
	// Email validation
	EmailPattern = ValidationRule{
		Required: true,
		Pattern:  `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
	}
	
	// URL validation
	URLPattern = ValidationRule{
		Required: true,
		Pattern:  `^https?://[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`,
	}
	
	// Numeric validations
	RequiredInt = ValidationRule{Required: true}
	OptionalInt = ValidationRule{Required: false}
	MinInt      = func(min int) ValidationRule { return ValidationRule{Min: min} }
	MaxInt      = func(max int) ValidationRule { return ValidationRule{Max: max} }
	MinMaxInt   = func(min, max int) ValidationRule { return ValidationRule{Min: min, Max: max} }
	
	// Boolean validations
	RequiredBool = ValidationRule{Required: true}
	OptionalBool = ValidationRule{Required: false}
)

// FileValidator provides file validation
type FileValidator struct {
	validator *Validator
}

// NewFileValidator creates a new file validator
func NewFileValidator() *FileValidator {
	return &FileValidator{
		validator: NewValidator(),
	}
}

// ValidateFile validates a file path
func (fv *FileValidator) ValidateFile(path string) error {
	if path == "" {
		return fmt.Errorf("file path is required")
	}
	
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", path)
	}
	
	return nil
}

// ValidateDirectory validates a directory path
func (fv *FileValidator) ValidateDirectory(path string) error {
	if path == "" {
		return fmt.Errorf("directory path is required")
	}
	
	// Check if directory exists
	if info, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", path)
	} else if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}
	
	return nil
}

// ValidateFileExtension validates file extension
func (fv *FileValidator) ValidateFileExtension(path string, allowedExtensions []string) error {
	ext := strings.ToLower(filepath.Ext(path))
	
	for _, allowed := range allowedExtensions {
		if ext == strings.ToLower(allowed) {
			return nil
		}
	}
	
	return fmt.Errorf("file extension '%s' is not allowed. Allowed extensions: %s", ext, strings.Join(allowedExtensions, ", "))
}

// ValidateFileSize validates file size
func (fv *FileValidator) ValidateFileSize(path string, maxSize int64) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("cannot get file info: %v", err)
	}
	
	if info.Size() > maxSize {
		return fmt.Errorf("file size (%d bytes) exceeds maximum allowed size (%d bytes)", info.Size(), maxSize)
	}
	
	return nil
}

// ProjectValidator provides project validation
type ProjectValidator struct {
	validator *Validator
}

// NewProjectValidator creates a new project validator
func NewProjectValidator() *ProjectValidator {
	return &ProjectValidator{
		validator: NewValidator(),
	}
}

// ValidateProjectName validates a project name
func (pv *ProjectValidator) ValidateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name is required")
	}
	
	// Check for valid characters (alphanumeric, hyphens, underscores)
	matched, err := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, name)
	if err != nil {
		return fmt.Errorf("invalid project name pattern: %v", err)
	}
	if !matched {
		return fmt.Errorf("project name can only contain letters, numbers, hyphens, and underscores")
	}
	
	// Check length
	if len(name) < 3 {
		return fmt.Errorf("project name must be at least 3 characters long")
	}
	if len(name) > 50 {
		return fmt.Errorf("project name must be at most 50 characters long")
	}
	
	// Check if directory already exists
	if _, err := os.Stat(name); !os.IsNotExist(err) {
		return fmt.Errorf("directory '%s' already exists", name)
	}
	
	return nil
}

// ValidateModuleName validates a module name
func (pv *ProjectValidator) ValidateModuleName(name string) error {
	if name == "" {
		return fmt.Errorf("module name is required")
	}
	
	// Check for valid characters (alphanumeric, hyphens, underscores)
	matched, err := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, name)
	if err != nil {
		return fmt.Errorf("invalid module name pattern: %v", err)
	}
	if !matched {
		return fmt.Errorf("module name can only contain letters, numbers, hyphens, and underscores")
	}
	
	// Check length
	if len(name) < 2 {
		return fmt.Errorf("module name must be at least 2 characters long")
	}
	if len(name) > 30 {
		return fmt.Errorf("module name must be at most 30 characters long")
	}
	
	return nil
}

// DatabaseValidator provides database validation
type DatabaseValidator struct {
	validator *Validator
}

// NewDatabaseValidator creates a new database validator
func NewDatabaseValidator() *DatabaseValidator {
	return &DatabaseValidator{
		validator: NewValidator(),
	}
}

// ValidateConnectionString validates a database connection string
func (dv *DatabaseValidator) ValidateConnectionString(driver, dsn string) error {
	if driver == "" {
		return fmt.Errorf("database driver is required")
	}
	
	if dsn == "" {
		return fmt.Errorf("database connection string is required")
	}
	
	// Validate driver
	validDrivers := []string{"postgres", "mysql", "sqlite", "mongodb"}
	valid := false
	for _, validDriver := range validDrivers {
		if driver == validDriver {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid database driver '%s'. Valid drivers: %s", driver, strings.Join(validDrivers, ", "))
	}
	
	return nil
}

// ValidateMigrationName validates a migration name
func (dv *DatabaseValidator) ValidateMigrationName(name string) error {
	if name == "" {
		return fmt.Errorf("migration name is required")
	}
	
	// Check for valid characters (alphanumeric, hyphens, underscores)
	matched, err := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, name)
	if err != nil {
		return fmt.Errorf("invalid migration name pattern: %v", err)
	}
	if !matched {
		return fmt.Errorf("migration name can only contain letters, numbers, hyphens, and underscores")
	}
	
	// Check length
	if len(name) < 3 {
		return fmt.Errorf("migration name must be at least 3 characters long")
	}
	if len(name) > 100 {
		return fmt.Errorf("migration name must be at most 100 characters long")
	}
	
	return nil
}

// QueueValidator provides queue validation
type QueueValidator struct {
	validator *Validator
}

// NewQueueValidator creates a new queue validator
func NewQueueValidator() *QueueValidator {
	return &QueueValidator{
		validator: NewValidator(),
	}
}

// ValidateQueueName validates a queue name
func (qv *QueueValidator) ValidateQueueName(name string) error {
	if name == "" {
		return fmt.Errorf("queue name is required")
	}
	
	// Check for valid characters (alphanumeric, hyphens, underscores, dots)
	matched, err := regexp.MatchString(`^[a-zA-Z0-9._-]+$`, name)
	if err != nil {
		return fmt.Errorf("invalid queue name pattern: %v", err)
	}
	if !matched {
		return fmt.Errorf("queue name can only contain letters, numbers, hyphens, underscores, and dots")
	}
	
	// Check length
	if len(name) < 1 {
		return fmt.Errorf("queue name must be at least 1 character long")
	}
	if len(name) > 50 {
		return fmt.Errorf("queue name must be at most 50 characters long")
	}
	
	return nil
}

// ValidateConnectionName validates a connection name
func (qv *QueueValidator) ValidateConnectionName(name string) error {
	if name == "" {
		return fmt.Errorf("connection name is required")
	}
	
	// Check for valid characters (alphanumeric, hyphens, underscores)
	matched, err := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, name)
	if err != nil {
		return fmt.Errorf("invalid connection name pattern: %v", err)
	}
	if !matched {
		return fmt.Errorf("connection name can only contain letters, numbers, hyphens, and underscores")
	}
	
	// Check length
	if len(name) < 1 {
		return fmt.Errorf("connection name must be at least 1 character long")
	}
	if len(name) > 30 {
		return fmt.Errorf("connection name must be at most 30 characters long")
	}
	
	return nil
}

// StorageValidator provides storage validation
type StorageValidator struct {
	validator *Validator
}

// NewStorageValidator creates a new storage validator
func NewStorageValidator() *StorageValidator {
	return &StorageValidator{
		validator: NewValidator(),
	}
}

// ValidateStoragePath validates a storage path
func (sv *StorageValidator) ValidateStoragePath(path string) error {
	if path == "" {
		return fmt.Errorf("storage path is required")
	}
	
	// Check for valid characters (alphanumeric, hyphens, underscores, slashes, dots)
	matched, err := regexp.MatchString(`^[a-zA-Z0-9._/-]+$`, path)
	if err != nil {
		return fmt.Errorf("invalid storage path pattern: %v", err)
	}
	if !matched {
		return fmt.Errorf("storage path can only contain letters, numbers, hyphens, underscores, slashes, and dots")
	}
	
	// Check length
	if len(path) < 1 {
		return fmt.Errorf("storage path must be at least 1 character long")
	}
	if len(path) > 200 {
		return fmt.Errorf("storage path must be at most 200 characters long")
	}
	
	return nil
}

// ValidateBucketName validates a bucket name
func (sv *StorageValidator) ValidateBucketName(name string) error {
	if name == "" {
		return fmt.Errorf("bucket name is required")
	}
	
	// Check for valid characters (alphanumeric, hyphens, dots)
	matched, err := regexp.MatchString(`^[a-zA-Z0-9.-]+$`, name)
	if err != nil {
		return fmt.Errorf("invalid bucket name pattern: %v", err)
	}
	if !matched {
		return fmt.Errorf("bucket name can only contain letters, numbers, hyphens, and dots")
	}
	
	// Check length
	if len(name) < 3 {
		return fmt.Errorf("bucket name must be at least 3 characters long")
	}
	if len(name) > 63 {
		return fmt.Errorf("bucket name must be at most 63 characters long")
	}
	
	// Check for consecutive dots
	if strings.Contains(name, "..") {
		return fmt.Errorf("bucket name cannot contain consecutive dots")
	}
	
	// Check for leading/trailing dots
	if strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".") {
		return fmt.Errorf("bucket name cannot start or end with a dot")
	}
	
	return nil
}

// ValidationResult represents the result of validation
type ValidationResult struct {
	Valid   bool
	Errors  map[string][]string
	Warnings map[string][]string
}

// NewValidationResult creates a new validation result
func NewValidationResult() *ValidationResult {
	return &ValidationResult{
		Valid:    true,
		Errors:   make(map[string][]string),
		Warnings: make(map[string][]string),
	}
}

// AddError adds an error to the result
func (vr *ValidationResult) AddError(field, message string) {
	vr.Valid = false
	vr.Errors[field] = append(vr.Errors[field], message)
}

// AddWarning adds a warning to the result
func (vr *ValidationResult) AddWarning(field, message string) {
	vr.Warnings[field] = append(vr.Warnings[field], message)
}

// HasErrors checks if there are errors
func (vr *ValidationResult) HasErrors() bool {
	return len(vr.Errors) > 0
}

// HasWarnings checks if there are warnings
func (vr *ValidationResult) HasWarnings() bool {
	return len(vr.Warnings) > 0
}

// GetErrorMessages gets all error messages
func (vr *ValidationResult) GetErrorMessages() []string {
	var messages []string
	for field, errors := range vr.Errors {
		for _, err := range errors {
			messages = append(messages, fmt.Sprintf("%s: %s", field, err))
		}
	}
	return messages
}

// GetWarningMessages gets all warning messages
func (vr *ValidationResult) GetWarningMessages() []string {
	var messages []string
	for field, warnings := range vr.Warnings {
		for _, warning := range warnings {
			messages = append(messages, fmt.Sprintf("%s: %s", field, warning))
		}
	}
	return messages
}
