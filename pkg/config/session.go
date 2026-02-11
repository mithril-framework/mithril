package config

import (
	"time"
)

// SessionConfig holds session configuration
type SessionConfig struct {
	// Basic settings
	Enabled        bool          `env:"SESSION_ENABLED" default:"true"`
	Driver         string        `env:"SESSION_DRIVER" default:"database" required:"true"`
	Lifetime       time.Duration `env:"SESSION_LIFETIME" default:"24h"`
	IdleTimeout    time.Duration `env:"SESSION_IDLE_TIMEOUT" default:"30m"`
	RefreshTimeout time.Duration `env:"SESSION_REFRESH_TIMEOUT" default:"5m"`

	// Cookie settings
	CookieName     string `env:"SESSION_COOKIE_NAME" default:"mithril_session"`
	CookiePath     string `env:"SESSION_COOKIE_PATH" default:"/"`
	CookieDomain   string `env:"SESSION_COOKIE_DOMAIN" default:""`
	CookieSecure   bool   `env:"SESSION_COOKIE_SECURE" default:"false"`
	CookieHTTPOnly bool   `env:"SESSION_COOKIE_HTTP_ONLY" default:"true"`
	CookieSameSite string `env:"SESSION_COOKIE_SAME_SITE" default:"Lax"`

	// Security settings
	SecretKey     string `env:"SESSION_SECRET_KEY" required:"true" min:"32"`
	EncryptionKey string `env:"SESSION_ENCRYPTION_KEY" default:""`
	HashKey       string `env:"SESSION_HASH_KEY" default:""`

	// Database settings (if using database driver)
	DBTable       string        `env:"SESSION_DB_TABLE" default:"sessions"`
	DBMaxIdle     int           `env:"SESSION_DB_MAX_IDLE" default:"10"`
	DBMaxOpen     int           `env:"SESSION_DB_MAX_OPEN" default:"100"`
	DBMaxLifetime time.Duration `env:"SESSION_DB_MAX_LIFETIME" default:"1h"`

	// Redis settings (if using Redis driver)
	RedisHost     string `env:"SESSION_REDIS_HOST" default:"localhost"`
	RedisPort     int    `env:"SESSION_REDIS_PORT" default:"6379"`
	RedisPassword string `env:"SESSION_REDIS_PASSWORD" default:""`
	RedisDB       int    `env:"SESSION_REDIS_DB" default:"3"`
	RedisPrefix   string `env:"SESSION_REDIS_PREFIX" default:"session:"`

	// File settings (if using file driver)
	FilePath string `env:"SESSION_FILE_PATH" default:"./storage/sessions"`

	// Memory settings (if using memory driver)
	MemoryMaxSessions int `env:"SESSION_MEMORY_MAX_SESSIONS" default:"10000"`

	// Cleanup settings
	CleanupEnabled   bool          `env:"SESSION_CLEANUP_ENABLED" default:"true"`
	CleanupInterval  time.Duration `env:"SESSION_CLEANUP_INTERVAL" default:"1h"`
	CleanupOlderThan time.Duration `env:"SESSION_CLEANUP_OLDER_THAN" default:"24h"`

	// Monitoring
	MonitoringEnabled bool   `env:"SESSION_MONITORING_ENABLED" default:"true"`
	MonitoringPath    string `env:"SESSION_MONITORING_PATH" default:"/monitor/session"`

	// Metrics
	MetricsEnabled bool   `env:"SESSION_METRICS_ENABLED" default:"true"`
	MetricsPath    string `env:"SESSION_METRICS_PATH" default:"/metrics/session"`

	// Logging
	LogEnabled bool   `env:"SESSION_LOG_ENABLED" default:"true"`
	LogLevel   string `env:"SESSION_LOG_LEVEL" default:"info"`

	// Health check
	HealthCheckEnabled  bool          `env:"SESSION_HEALTH_CHECK_ENABLED" default:"true"`
	HealthCheckInterval time.Duration `env:"SESSION_HEALTH_CHECK_INTERVAL" default:"30s"`

	// CSRF protection
	CSRFEnabled     bool   `env:"SESSION_CSRF_ENABLED" default:"true"`
	CSRFTokenLength int    `env:"SESSION_CSRF_TOKEN_LENGTH" default:"32"`
	CSRFHeaderName  string `env:"SESSION_CSRF_HEADER_NAME" default:"X-CSRF-Token"`

	// Flash messages
	FlashEnabled bool `env:"SESSION_FLASH_ENABLED" default:"true"`

	// Session regeneration
	RegenerateOnLogin  bool `env:"SESSION_REGENERATE_ON_LOGIN" default:"true"`
	RegenerateOnLogout bool `env:"SESSION_REGENERATE_ON_LOGOUT" default:"true"`

	// Session hijacking protection
	HijackingProtection bool `env:"SESSION_HIJACKING_PROTECTION" default:"true"`

	// Session fixation protection
	FixationProtection bool `env:"SESSION_FIXATION_PROTECTION" default:"true"`
}

// IsEnabled returns whether sessions are enabled
func (c *SessionConfig) IsEnabled() bool {
	return c.Enabled
}

// GetDriver returns the session driver
func (c *SessionConfig) GetDriver() string {
	if c.Driver == "" {
		return "database"
	}
	return c.Driver
}

// GetLifetime returns the session lifetime
func (c *SessionConfig) GetLifetime() time.Duration {
	if c.Lifetime <= 0 {
		return 24 * time.Hour
	}
	return c.Lifetime
}

// GetIdleTimeout returns the idle timeout
func (c *SessionConfig) GetIdleTimeout() time.Duration {
	if c.IdleTimeout <= 0 {
		return 30 * time.Minute
	}
	return c.IdleTimeout
}

// GetRefreshTimeout returns the refresh timeout
func (c *SessionConfig) GetRefreshTimeout() time.Duration {
	if c.RefreshTimeout <= 0 {
		return 5 * time.Minute
	}
	return c.RefreshTimeout
}

// GetCookieName returns the cookie name
func (c *SessionConfig) GetCookieName() string {
	if c.CookieName == "" {
		return "mithril_session"
	}
	return c.CookieName
}

// GetCookiePath returns the cookie path
func (c *SessionConfig) GetCookiePath() string {
	if c.CookiePath == "" {
		return "/"
	}
	return c.CookiePath
}

// GetCookieDomain returns the cookie domain
func (c *SessionConfig) GetCookieDomain() string {
	return c.CookieDomain
}

// IsCookieSecure returns whether the cookie should be secure
func (c *SessionConfig) IsCookieSecure() bool {
	return c.CookieSecure
}

// IsCookieHTTPOnly returns whether the cookie should be HTTP only
func (c *SessionConfig) IsCookieHTTPOnly() bool {
	return c.CookieHTTPOnly
}

// GetCookieSameSite returns the cookie SameSite setting
func (c *SessionConfig) GetCookieSameSite() string {
	if c.CookieSameSite == "" {
		return "Lax"
	}
	return c.CookieSameSite
}

// GetSecretKey returns the secret key
func (c *SessionConfig) GetSecretKey() string {
	return c.SecretKey
}

// GetEncryptionKey returns the encryption key
func (c *SessionConfig) GetEncryptionKey() string {
	return c.EncryptionKey
}

// GetHashKey returns the hash key
func (c *SessionConfig) GetHashKey() string {
	return c.HashKey
}

// GetDBTable returns the database table name
func (c *SessionConfig) GetDBTable() string {
	if c.DBTable == "" {
		return "sessions"
	}
	return c.DBTable
}

// GetDBMaxIdle returns the database max idle connections
func (c *SessionConfig) GetDBMaxIdle() int {
	if c.DBMaxIdle <= 0 {
		return 10
	}
	return c.DBMaxIdle
}

// GetDBMaxOpen returns the database max open connections
func (c *SessionConfig) GetDBMaxOpen() int {
	if c.DBMaxOpen <= 0 {
		return 100
	}
	return c.DBMaxOpen
}

// GetDBMaxLifetime returns the database max connection lifetime
func (c *SessionConfig) GetDBMaxLifetime() time.Duration {
	if c.DBMaxLifetime <= 0 {
		return time.Hour
	}
	return c.DBMaxLifetime
}

// GetRedisHost returns the Redis host
func (c *SessionConfig) GetRedisHost() string {
	if c.RedisHost == "" {
		return "localhost"
	}
	return c.RedisHost
}

// GetRedisPort returns the Redis port
func (c *SessionConfig) GetRedisPort() int {
	if c.RedisPort <= 0 {
		return 6379
	}
	return c.RedisPort
}

// GetRedisPassword returns the Redis password
func (c *SessionConfig) GetRedisPassword() string {
	return c.RedisPassword
}

// GetRedisDB returns the Redis database number
func (c *SessionConfig) GetRedisDB() int {
	return c.RedisDB
}

// GetRedisPrefix returns the Redis prefix
func (c *SessionConfig) GetRedisPrefix() string {
	if c.RedisPrefix == "" {
		return "session:"
	}
	return c.RedisPrefix
}

// GetRedisAddr returns the Redis address
func (c *SessionConfig) GetRedisAddr() string {
	return c.GetRedisHost() + ":" + string(rune(c.GetRedisPort()))
}

// GetFilePath returns the file path
func (c *SessionConfig) GetFilePath() string {
	if c.FilePath == "" {
		return "./storage/sessions"
	}
	return c.FilePath
}

// GetMemoryMaxSessions returns the maximum number of sessions in memory
func (c *SessionConfig) GetMemoryMaxSessions() int {
	if c.MemoryMaxSessions <= 0 {
		return 10000
	}
	return c.MemoryMaxSessions
}

// IsCleanupEnabled returns whether cleanup is enabled
func (c *SessionConfig) IsCleanupEnabled() bool {
	return c.CleanupEnabled
}

// GetCleanupInterval returns the cleanup interval
func (c *SessionConfig) GetCleanupInterval() time.Duration {
	if c.CleanupInterval <= 0 {
		return time.Hour
	}
	return c.CleanupInterval
}

// GetCleanupOlderThan returns the cleanup older than duration
func (c *SessionConfig) GetCleanupOlderThan() time.Duration {
	if c.CleanupOlderThan <= 0 {
		return 24 * time.Hour
	}
	return c.CleanupOlderThan
}

// IsMonitoringEnabled returns whether monitoring is enabled
func (c *SessionConfig) IsMonitoringEnabled() bool {
	return c.MonitoringEnabled
}

// GetMonitoringPath returns the monitoring path
func (c *SessionConfig) GetMonitoringPath() string {
	if c.MonitoringPath == "" {
		return "/monitor/session"
	}
	return c.MonitoringPath
}

// IsMetricsEnabled returns whether metrics are enabled
func (c *SessionConfig) IsMetricsEnabled() bool {
	return c.MetricsEnabled
}

// GetMetricsPath returns the metrics path
func (c *SessionConfig) GetMetricsPath() string {
	if c.MetricsPath == "" {
		return "/metrics/session"
	}
	return c.MetricsPath
}

// IsLogEnabled returns whether logging is enabled
func (c *SessionConfig) IsLogEnabled() bool {
	return c.LogEnabled
}

// GetLogLevel returns the log level
func (c *SessionConfig) GetLogLevel() string {
	if c.LogLevel == "" {
		return "info"
	}
	return c.LogLevel
}

// IsHealthCheckEnabled returns whether health check is enabled
func (c *SessionConfig) IsHealthCheckEnabled() bool {
	return c.HealthCheckEnabled
}

// GetHealthCheckInterval returns the health check interval
func (c *SessionConfig) GetHealthCheckInterval() time.Duration {
	if c.HealthCheckInterval <= 0 {
		return 30 * time.Second
	}
	return c.HealthCheckInterval
}

// IsCSRFEnabled returns whether CSRF protection is enabled
func (c *SessionConfig) IsCSRFEnabled() bool {
	return c.CSRFEnabled
}

// GetCSRFTokenLength returns the CSRF token length
func (c *SessionConfig) GetCSRFTokenLength() int {
	if c.CSRFTokenLength <= 0 {
		return 32
	}
	return c.CSRFTokenLength
}

// GetCSRFHeaderName returns the CSRF header name
func (c *SessionConfig) GetCSRFHeaderName() string {
	if c.CSRFHeaderName == "" {
		return "X-CSRF-Token"
	}
	return c.CSRFHeaderName
}

// IsFlashEnabled returns whether flash messages are enabled
func (c *SessionConfig) IsFlashEnabled() bool {
	return c.FlashEnabled
}

// IsRegenerateOnLogin returns whether to regenerate session on login
func (c *SessionConfig) IsRegenerateOnLogin() bool {
	return c.RegenerateOnLogin
}

// IsRegenerateOnLogout returns whether to regenerate session on logout
func (c *SessionConfig) IsRegenerateOnLogout() bool {
	return c.RegenerateOnLogout
}

// IsHijackingProtection returns whether hijacking protection is enabled
func (c *SessionConfig) IsHijackingProtection() bool {
	return c.HijackingProtection
}

// IsFixationProtection returns whether fixation protection is enabled
func (c *SessionConfig) IsFixationProtection() bool {
	return c.FixationProtection
}

// IsDatabase returns true if using database driver
func (c *SessionConfig) IsDatabase() bool {
	return c.GetDriver() == "database"
}

// IsRedis returns true if using Redis driver
func (c *SessionConfig) IsRedis() bool {
	return c.GetDriver() == "redis"
}

// IsFile returns true if using file driver
func (c *SessionConfig) IsFile() bool {
	return c.GetDriver() == "file"
}

// IsMemory returns true if using memory driver
func (c *SessionConfig) IsMemory() bool {
	return c.GetDriver() == "memory"
}
