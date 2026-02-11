package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// ConfigManager handles configuration loading and management
type ConfigManager struct {
	configs map[string]interface{}
	envFile string
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(envFile string) *ConfigManager {
	return &ConfigManager{
		configs: make(map[string]interface{}),
		envFile: envFile,
	}
}

// Load loads configuration from environment and .env file
func (cm *ConfigManager) Load() error {
	// Load .env file if it exists
	if cm.envFile != "" {
		if err := godotenv.Load(cm.envFile); err != nil {
			// Don't fail if .env file doesn't exist
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to load .env file: %w", err)
			}
		}
	}

	return nil
}

// Register registers a configuration struct
func (cm *ConfigManager) Register(name string, config interface{}) error {
	// Load environment variables into the config struct
	if err := cm.loadConfigFromEnv(config); err != nil {
		return fmt.Errorf("failed to load config %s: %w", name, err)
	}

	// Validate the config
	if err := cm.validateConfig(config); err != nil {
		return fmt.Errorf("config %s validation failed: %w", name, err)
	}

	cm.configs[name] = config
	return nil
}

// Get retrieves a configuration by name
func (cm *ConfigManager) Get(name string) (interface{}, bool) {
	config, exists := cm.configs[name]
	return config, exists
}

// GetTyped retrieves a typed configuration
func (cm *ConfigManager) GetTyped(name string, target interface{}) error {
	config, exists := cm.configs[name]
	if !exists {
		return fmt.Errorf("config %s not found", name)
	}

	// Use reflection to copy the config to target
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}

	targetValue = targetValue.Elem()
	sourceValue := reflect.ValueOf(config)

	if !targetValue.Type().AssignableTo(sourceValue.Type()) {
		return fmt.Errorf("type mismatch: expected %s, got %s", targetValue.Type(), sourceValue.Type())
	}

	targetValue.Set(sourceValue)
	return nil
}

// loadConfigFromEnv loads configuration from environment variables
func (cm *ConfigManager) loadConfigFromEnv(config interface{}) error {
	configValue := reflect.ValueOf(config)
	if configValue.Kind() != reflect.Ptr {
		return fmt.Errorf("config must be a pointer")
	}

	configValue = configValue.Elem()
	configType := configValue.Type()

	for i := 0; i < configValue.NumField(); i++ {
		field := configValue.Field(i)
		fieldType := configType.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Get environment variable name from struct tag
		envKey := fieldType.Tag.Get("env")
		if envKey == "" {
			continue
		}

		// Get default value from struct tag
		defaultValue := fieldType.Tag.Get("default")

		// Get environment variable value
		envValue := os.Getenv(envKey)
		if envValue == "" {
			envValue = defaultValue
		}

		// Skip if no value is available
		if envValue == "" {
			continue
		}

		// Set the field value based on its type
		if err := cm.setFieldValue(field, envValue); err != nil {
			return fmt.Errorf("failed to set field %s: %w", fieldType.Name, err)
		}
	}

	return nil
}

// setFieldValue sets a field value from string
func (cm *ConfigManager) setFieldValue(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Handle time.Duration (which is int64)
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			duration, err := cm.parseDuration(value)
			if err != nil {
				return err
			}
			field.Set(reflect.ValueOf(duration))
		} else {
			intValue, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return err
			}
			field.SetInt(intValue)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintValue, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(uintValue)
	case reflect.Float32, reflect.Float64:
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatValue)
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(boolValue)
	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.String {
			// Handle string slices (comma-separated values)
			values := strings.Split(value, ",")
			slice := reflect.MakeSlice(field.Type(), len(values), len(values))
			for i, v := range values {
				slice.Index(i).SetString(strings.TrimSpace(v))
			}
			field.Set(slice)
		}
	case reflect.Struct:
		// Handle time.Duration
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			duration, err := time.ParseDuration(value)
			if err != nil {
				return err
			}
			field.Set(reflect.ValueOf(duration))
		}
	}

	return nil
}

// parseDuration parses a duration string with support for days and seconds
func (cm *ConfigManager) parseDuration(value string) (time.Duration, error) {
	// Handle days (d) by converting to hours
	if strings.HasSuffix(value, "d") {
		daysStr := strings.TrimSuffix(value, "d")
		days, err := strconv.Atoi(daysStr)
		if err != nil {
			return 0, fmt.Errorf("invalid days value: %s", daysStr)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	// Handle plain numbers as seconds
	if _, err := strconv.Atoi(value); err == nil {
		seconds, _ := strconv.Atoi(value)
		return time.Duration(seconds) * time.Second, nil
	}

	// Use standard time.ParseDuration for other units
	return time.ParseDuration(value)
}

// validateConfig validates a configuration struct
func (cm *ConfigManager) validateConfig(config interface{}) error {
	configValue := reflect.ValueOf(config)
	if configValue.Kind() == reflect.Ptr {
		configValue = configValue.Elem()
	}

	configType := configValue.Type()

	for i := 0; i < configValue.NumField(); i++ {
		field := configValue.Field(i)
		fieldType := configType.Field(i)

		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}

		// Check required fields
		required := fieldType.Tag.Get("required")
		if required == "true" && field.IsZero() {
			return fmt.Errorf("required field %s is not set", fieldType.Name)
		}

		// Validate field value
		if err := cm.validateField(field, fieldType); err != nil {
			return fmt.Errorf("field %s validation failed: %w", fieldType.Name, err)
		}
	}

	return nil
}

// validateField validates a single field
func (cm *ConfigManager) validateField(field reflect.Value, fieldType reflect.StructField) error {
	// Add custom validation rules here
	// For now, just check basic constraints

	switch field.Kind() {
	case reflect.String:
		value := field.String()
		if minLen := fieldType.Tag.Get("min"); minLen != "" {
			if min, err := strconv.Atoi(minLen); err == nil && len(value) < min {
				return fmt.Errorf("string length must be at least %d", min)
			}
		}
		if maxLen := fieldType.Tag.Get("max"); maxLen != "" {
			if max, err := strconv.Atoi(maxLen); err == nil && len(value) > max {
				return fmt.Errorf("string length must be at most %d", max)
			}
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value := field.Int()
		if min := fieldType.Tag.Get("min"); min != "" {
			if minVal, err := strconv.ParseInt(min, 10, 64); err == nil && value < minVal {
				return fmt.Errorf("value must be at least %d", minVal)
			}
		}
		if max := fieldType.Tag.Get("max"); max != "" {
			if maxVal, err := strconv.ParseInt(max, 10, 64); err == nil && value > maxVal {
				return fmt.Errorf("value must be at most %d", maxVal)
			}
		}
	}

	return nil
}

// GetEnv gets an environment variable with default value
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvInt gets an environment variable as integer with default value
func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetEnvBool gets an environment variable as boolean with default value
func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// GetEnvDuration gets an environment variable as duration with default value
func GetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
