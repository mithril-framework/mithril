package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Manager holds all configuration instances
type Manager struct {
	configManager *ConfigManager
	App           *AppConfig
	Database      *DatabaseConfig
	JWT           *JWTConfig
	Mail          *MailConfig
	Queue         *QueueConfig
	Storage       *StorageConfig
	CORS          *CORSConfig
	RateLimit     *RateLimitConfig
	Session       *SessionConfig
}

// NewManager creates a new configuration manager
func NewManager(envFile string) (*Manager, error) {
	// Initialize config manager
	configManager := NewConfigManager(envFile)
	if err := configManager.Load(); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize all config structs
	manager := &Manager{
		configManager: configManager,
		App:           &AppConfig{},
		Database:      &DatabaseConfig{},
		JWT:           &JWTConfig{},
		Mail:          &MailConfig{},
		Queue:         &QueueConfig{},
		Storage:       &StorageConfig{},
		CORS:          &CORSConfig{},
		RateLimit:     &RateLimitConfig{},
		Session:       &SessionConfig{},
	}

	// Register all configurations
	configs := map[string]interface{}{
		"app":       manager.App,
		"database":  manager.Database,
		"jwt":       manager.JWT,
		"mail":      manager.Mail,
		"queue":     manager.Queue,
		"storage":   manager.Storage,
		"cors":      manager.CORS,
		"ratelimit": manager.RateLimit,
		"session":   manager.Session,
	}

	// Load each configuration
	for name, config := range configs {
		if err := configManager.Register(name, config); err != nil {
			return nil, fmt.Errorf("failed to register %s config: %w", name, err)
		}
	}

	return manager, nil
}

// NewManagerFromEnv creates a new configuration manager from environment
func NewManagerFromEnv() (*Manager, error) {
	// Try to find .env file in common locations
	envFiles := []string{
		".env",
		".env.local",
		".env.development",
		".env.staging",
		".env.production",
		".env.testing",
	}

	var envFile string
	for _, file := range envFiles {
		if _, err := os.Stat(file); err == nil {
			envFile = file
			break
		}
	}

	return NewManager(envFile)
}

// NewManagerFromFile creates a new configuration manager from a specific file
func NewManagerFromFile(envFile string) (*Manager, error) {
	// Check if file exists
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("environment file %s does not exist", envFile)
	}

	return NewManager(envFile)
}

// NewManagerFromDir creates a new configuration manager from a directory
func NewManagerFromDir(dir string) (*Manager, error) {
	// Look for .env files in the directory
	envFiles := []string{
		filepath.Join(dir, ".env"),
		filepath.Join(dir, ".env.local"),
		filepath.Join(dir, ".env.development"),
		filepath.Join(dir, ".env.staging"),
		filepath.Join(dir, ".env.production"),
		filepath.Join(dir, ".env.testing"),
	}

	var envFile string
	for _, file := range envFiles {
		if _, err := os.Stat(file); err == nil {
			envFile = file
			break
		}
	}

	return NewManager(envFile)
}

// GetConfig returns a configuration by name
func (m *Manager) GetConfig(name string) (interface{}, bool) {
	return m.configManager.Get(name)
}

// GetTypedConfig returns a typed configuration
func (m *Manager) GetTypedConfig(name string, target interface{}) error {
	return m.configManager.GetTyped(name, target)
}

// Reload reloads all configurations
func (m *Manager) Reload() error {
	// Reload environment variables
	if err := m.configManager.Load(); err != nil {
		return fmt.Errorf("failed to reload configuration: %w", err)
	}

	// Re-register all configurations
	configs := map[string]interface{}{
		"app":       m.App,
		"database":  m.Database,
		"jwt":       m.JWT,
		"mail":      m.Mail,
		"queue":     m.Queue,
		"storage":   m.Storage,
		"cors":      m.CORS,
		"ratelimit": m.RateLimit,
		"session":   m.Session,
	}

	for name, config := range configs {
		if err := m.configManager.Register(name, config); err != nil {
			return fmt.Errorf("failed to re-register %s config: %w", name, err)
		}
	}

	return nil
}

// Validate validates all configurations
func (m *Manager) Validate() error {
	// Validate each configuration
	configs := map[string]interface{}{
		"app":       m.App,
		"database":  m.Database,
		"jwt":       m.JWT,
		"mail":      m.Mail,
		"queue":     m.Queue,
		"storage":   m.Storage,
		"cors":      m.CORS,
		"ratelimit": m.RateLimit,
		"session":   m.Session,
	}

	for name, config := range configs {
		if err := m.configManager.validateConfig(config); err != nil {
			return fmt.Errorf("validation failed for %s config: %w", name, err)
		}
	}

	return nil
}

// GetEnvironment returns the current environment
func (m *Manager) GetEnvironment() string {
	return m.App.Environment
}

// IsDevelopment returns true if in development mode
func (m *Manager) IsDevelopment() bool {
	return m.App.IsDevelopment()
}

// IsProduction returns true if in production mode
func (m *Manager) IsProduction() bool {
	return m.App.IsProduction()
}

// IsStaging returns true if in staging mode
func (m *Manager) IsStaging() bool {
	return m.App.IsStaging()
}

// IsTesting returns true if in testing mode
func (m *Manager) IsTesting() bool {
	return m.App.IsTesting()
}

// GetAppConfig returns the app configuration
func (m *Manager) GetAppConfig() *AppConfig {
	return m.App
}

// GetDatabaseConfig returns the database configuration
func (m *Manager) GetDatabaseConfig() *DatabaseConfig {
	return m.Database
}

// GetJWTConfig returns the JWT configuration
func (m *Manager) GetJWTConfig() *JWTConfig {
	return m.JWT
}

// GetMailConfig returns the mail configuration
func (m *Manager) GetMailConfig() *MailConfig {
	return m.Mail
}

// GetQueueConfig returns the queue configuration
func (m *Manager) GetQueueConfig() *QueueConfig {
	return m.Queue
}

// GetStorageConfig returns the storage configuration
func (m *Manager) GetStorageConfig() *StorageConfig {
	return m.Storage
}

// GetCORSConfig returns the CORS configuration
func (m *Manager) GetCORSConfig() *CORSConfig {
	return m.CORS
}

// GetRateLimitConfig returns the rate limit configuration
func (m *Manager) GetRateLimitConfig() *RateLimitConfig {
	return m.RateLimit
}

// GetSessionConfig returns the session configuration
func (m *Manager) GetSessionConfig() *SessionConfig {
	return m.Session
}

// SetEnvironment sets the environment
func (m *Manager) SetEnvironment(env string) {
	m.App.Environment = env
}

// SetDebug sets the debug mode
func (m *Manager) SetDebug(debug bool) {
	m.App.Debug = debug
}

// SetPort sets the port
func (m *Manager) SetPort(port string) {
	m.App.Port = port
}

// SetHost sets the host
func (m *Manager) SetHost(host string) {
	m.App.Host = host
}

// SetSecretKey sets the secret key
func (m *Manager) SetSecretKey(secretKey string) {
	m.App.SecretKey = secretKey
}

// SetDatabaseConfig sets the database configuration
func (m *Manager) SetDatabaseConfig(config *DatabaseConfig) {
	m.Database = config
}

// SetJWTConfig sets the JWT configuration
func (m *Manager) SetJWTConfig(config *JWTConfig) {
	m.JWT = config
}

// SetMailConfig sets the mail configuration
func (m *Manager) SetMailConfig(config *MailConfig) {
	m.Mail = config
}

// SetQueueConfig sets the queue configuration
func (m *Manager) SetQueueConfig(config *QueueConfig) {
	m.Queue = config
}

// SetStorageConfig sets the storage configuration
func (m *Manager) SetStorageConfig(config *StorageConfig) {
	m.Storage = config
}

// SetCORSConfig sets the CORS configuration
func (m *Manager) SetCORSConfig(config *CORSConfig) {
	m.CORS = config
}

// SetRateLimitConfig sets the rate limit configuration
func (m *Manager) SetRateLimitConfig(config *RateLimitConfig) {
	m.RateLimit = config
}

// SetSessionConfig sets the session configuration
func (m *Manager) SetSessionConfig(config *SessionConfig) {
	m.Session = config
}
