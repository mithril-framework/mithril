package config

import (
	"os"
	"time"
)

// AppConfig holds application configuration
type AppConfig struct {
	// Basic app settings
	Name        string `env:"APP_NAME" default:"Mithril App" required:"true" min:"1" max:"100"`
	Version     string `env:"APP_VERSION" default:"1.0.0" required:"true"`
	Environment string `env:"APP_ENV" default:"development" required:"true"`
	Debug       bool   `env:"APP_DEBUG" default:"false"`

	// Server settings
	Host string `env:"APP_HOST" default:"0.0.0.0" required:"true"`
	Port string `env:"APP_PORT" default:"4000" required:"true"`

	// Timezone
	Timezone string `env:"APP_TIMEZONE" default:"UTC" required:"true"`

	// Security
	SecretKey string `env:"APP_SECRET_KEY" required:"true" min:"32"`

	// CORS settings
	CORSAllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS" default:"*"`
	CORSAllowedMethods []string `env:"CORS_ALLOWED_METHODS" default:"GET,POST,PUT,DELETE,OPTIONS"`
	CORSAllowedHeaders []string `env:"CORS_ALLOWED_HEADERS" default:"Content-Type,Authorization,X-Requested-With"`

	// Rate limiting
	RateLimitEnabled bool          `env:"RATE_LIMIT_ENABLED" default:"true"`
	RateLimitRPS     int           `env:"RATE_LIMIT_RPS" default:"100"`
	RateLimitBurst   int           `env:"RATE_LIMIT_BURST" default:"200"`
	RateLimitWindow  time.Duration `env:"RATE_LIMIT_WINDOW" default:"1m"`

	// File upload settings
	MaxUploadSize int64 `env:"MAX_UPLOAD_SIZE" default:"10485760"` // 10MB

	// Session settings
	SessionName     string        `env:"SESSION_NAME" default:"mithril_session"`
	SessionSecret   string        `env:"SESSION_SECRET" required:"true" min:"32"`
	SessionLifetime time.Duration `env:"SESSION_LIFETIME" default:"24h"`

	// Logging
	LogLevel  string `env:"LOG_LEVEL" default:"info"`
	LogFormat string `env:"LOG_FORMAT" default:"json"`

	// Health check settings
	HealthCheckPath string `env:"HEALTH_CHECK_PATH" default:"/health"`
	ReadyCheckPath  string `env:"READY_CHECK_PATH" default:"/ready"`

	// Metrics
	MetricsEnabled bool   `env:"METRICS_ENABLED" default:"true"`
	MetricsPath    string `env:"METRICS_PATH" default:"/metrics"`

	// Monitoring
	MonitoringEnabled bool   `env:"MONITORING_ENABLED" default:"true"`
	MonitoringPath    string `env:"MONITORING_PATH" default:"/monitor"`
}

// IsDevelopment returns true if the app is in development mode
func (c *AppConfig) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if the app is in production mode
func (c *AppConfig) IsProduction() bool {
	return c.Environment == "production"
}

// IsStaging returns true if the app is in staging mode
func (c *AppConfig) IsStaging() bool {
	return c.Environment == "staging"
}

// IsTesting returns true if the app is in testing mode
func (c *AppConfig) IsTesting() bool {
	return c.Environment == "testing"
}

// GetAddress returns the full server address
func (c *AppConfig) GetAddress() string {
	return c.Host + ":" + c.Port
}

// GetCORSAllowedOrigins returns CORS allowed origins as a slice
func (c *AppConfig) GetCORSAllowedOrigins() []string {
	if len(c.CORSAllowedOrigins) == 0 {
		return []string{"*"}
	}
	return c.CORSAllowedOrigins
}

// GetCORSAllowedMethods returns CORS allowed methods as a slice
func (c *AppConfig) GetCORSAllowedMethods() []string {
	if len(c.CORSAllowedMethods) == 0 {
		return []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	return c.CORSAllowedMethods
}

// GetCORSAllowedHeaders returns CORS allowed headers as a slice
func (c *AppConfig) GetCORSAllowedHeaders() []string {
	if len(c.CORSAllowedHeaders) == 0 {
		return []string{"Content-Type", "Authorization", "X-Requested-With"}
	}
	return c.CORSAllowedHeaders
}

// GetMaxUploadSizeMB returns max upload size in MB
func (c *AppConfig) GetMaxUploadSizeMB() int64 {
	return c.MaxUploadSize / 1024 / 1024
}

// GetLogLevel returns the log level
func (c *AppConfig) GetLogLevel() string {
	if c.IsDevelopment() {
		return "debug"
	}
	return c.LogLevel
}

// GetLogFormat returns the log format
func (c *AppConfig) GetLogFormat() string {
	if c.IsDevelopment() {
		return "text"
	}
	return c.LogFormat
}

// NewAppConfig creates a new AppConfig from environment variables
func NewAppConfig() *AppConfig {
	return &AppConfig{
		Name:        getEnvOrDefault("APP_NAME", "Mithril App"),
		Version:     getEnvOrDefault("APP_VERSION", "1.0.0"),
		Environment: getEnvOrDefault("APP_ENV", "development"),
		Debug:       getEnvOrDefault("APP_DEBUG", "false") == "true",
		Host:        getEnvOrDefault("APP_HOST", "0.0.0.0"),
		Port:        getEnvOrDefault("APP_PORT", "4000"),
		Timezone:    getEnvOrDefault("APP_TIMEZONE", "UTC"),
		SecretKey:   getEnvOrDefault("APP_SECRET_KEY", "change-me-in-production-min-32-chars"),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
