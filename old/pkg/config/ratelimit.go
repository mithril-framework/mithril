package config

import (
	"time"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	// Global settings
	Enabled bool `env:"RATE_LIMIT_ENABLED" default:"true"`

	// Default rate limit settings
	DefaultRPS    int           `env:"RATE_LIMIT_DEFAULT_RPS" default:"100"`
	DefaultBurst  int           `env:"RATE_LIMIT_DEFAULT_BURST" default:"200"`
	DefaultWindow time.Duration `env:"RATE_LIMIT_DEFAULT_WINDOW" default:"1m"`

	// IP-based rate limiting
	IPEnabled bool          `env:"RATE_LIMIT_IP_ENABLED" default:"true"`
	IPRPS     int           `env:"RATE_LIMIT_IP_RPS" default:"100"`
	IPBurst   int           `env:"RATE_LIMIT_IP_BURST" default:"200"`
	IPWindow  time.Duration `env:"RATE_LIMIT_IP_WINDOW" default:"1m"`

	// User-based rate limiting
	UserEnabled bool          `env:"RATE_LIMIT_USER_ENABLED" default:"true"`
	UserRPS     int           `env:"RATE_LIMIT_USER_RPS" default:"50"`
	UserBurst   int           `env:"RATE_LIMIT_USER_BURST" default:"100"`
	UserWindow  time.Duration `env:"RATE_LIMIT_USER_WINDOW" default:"1m"`

	// Route-based rate limiting
	RouteEnabled bool          `env:"RATE_LIMIT_ROUTE_ENABLED" default:"true"`
	RouteRPS     int           `env:"RATE_LIMIT_ROUTE_RPS" default:"200"`
	RouteBurst   int           `env:"RATE_LIMIT_ROUTE_BURST" default:"400"`
	RouteWindow  time.Duration `env:"RATE_LIMIT_ROUTE_WINDOW" default:"1m"`

	// API rate limiting
	APIEnabled bool          `env:"RATE_LIMIT_API_ENABLED" default:"true"`
	APIRPS     int           `env:"RATE_LIMIT_API_RPS" default:"1000"`
	APIBurst   int           `env:"RATE_LIMIT_API_BURST" default:"2000"`
	APIWindow  time.Duration `env:"RATE_LIMIT_API_WINDOW" default:"1m"`

	// Auth rate limiting
	AuthEnabled bool          `env:"RATE_LIMIT_AUTH_ENABLED" default:"true"`
	AuthRPS     int           `env:"RATE_LIMIT_AUTH_RPS" default:"5"`
	AuthBurst   int           `env:"RATE_LIMIT_AUTH_BURST" default:"10"`
	AuthWindow  time.Duration `env:"RATE_LIMIT_AUTH_WINDOW" default:"1m"`

	// Upload rate limiting
	UploadEnabled bool          `env:"RATE_LIMIT_UPLOAD_ENABLED" default:"true"`
	UploadRPS     int           `env:"RATE_LIMIT_UPLOAD_RPS" default:"10"`
	UploadBurst   int           `env:"RATE_LIMIT_UPLOAD_BURST" default:"20"`
	UploadWindow  time.Duration `env:"RATE_LIMIT_UPLOAD_WINDOW" default:"1m"`

	// Storage settings
	StorageDriver string `env:"RATE_LIMIT_STORAGE_DRIVER" default:"memory"`
	StoragePath   string `env:"RATE_LIMIT_STORAGE_PATH" default:"./storage/ratelimit"`

	// Redis settings (if using Redis storage)
	RedisHost     string `env:"RATE_LIMIT_REDIS_HOST" default:"localhost"`
	RedisPort     int    `env:"RATE_LIMIT_REDIS_PORT" default:"6379"`
	RedisPassword string `env:"RATE_LIMIT_REDIS_PASSWORD" default:""`
	RedisDB       int    `env:"RATE_LIMIT_REDIS_DB" default:"2"`

	// Cleanup settings
	CleanupEnabled  bool          `env:"RATE_LIMIT_CLEANUP_ENABLED" default:"true"`
	CleanupInterval time.Duration `env:"RATE_LIMIT_CLEANUP_INTERVAL" default:"5m"`

	// Monitoring
	MonitoringEnabled bool   `env:"RATE_LIMIT_MONITORING_ENABLED" default:"true"`
	MonitoringPath    string `env:"RATE_LIMIT_MONITORING_PATH" default:"/monitor/ratelimit"`

	// Metrics
	MetricsEnabled bool   `env:"RATE_LIMIT_METRICS_ENABLED" default:"true"`
	MetricsPath    string `env:"RATE_LIMIT_METRICS_PATH" default:"/metrics/ratelimit"`

	// Logging
	LogEnabled bool   `env:"RATE_LIMIT_LOG_ENABLED" default:"true"`
	LogLevel   string `env:"RATE_LIMIT_LOG_LEVEL" default:"info"`

	// Health check
	HealthCheckEnabled  bool          `env:"RATE_LIMIT_HEALTH_CHECK_ENABLED" default:"true"`
	HealthCheckInterval time.Duration `env:"RATE_LIMIT_HEALTH_CHECK_INTERVAL" default:"30s"`

	// Whitelist settings
	WhitelistEnabled bool     `env:"RATE_LIMIT_WHITELIST_ENABLED" default:"false"`
	WhitelistIPs     []string `env:"RATE_LIMIT_WHITELIST_IPS" default:""`

	// Blacklist settings
	BlacklistEnabled bool     `env:"RATE_LIMIT_BLACKLIST_ENABLED" default:"false"`
	BlacklistIPs     []string `env:"RATE_LIMIT_BLACKLIST_IPS" default:""`

	// Custom rules
	CustomRulesEnabled bool `env:"RATE_LIMIT_CUSTOM_RULES_ENABLED" default:"false"`
}

// IsEnabled returns whether rate limiting is enabled
func (c *RateLimitConfig) IsEnabled() bool {
	return c.Enabled
}

// GetDefaultRPS returns the default RPS
func (c *RateLimitConfig) GetDefaultRPS() int {
	if c.DefaultRPS <= 0 {
		return 100
	}
	return c.DefaultRPS
}

// GetDefaultBurst returns the default burst
func (c *RateLimitConfig) GetDefaultBurst() int {
	if c.DefaultBurst <= 0 {
		return 200
	}
	return c.DefaultBurst
}

// GetDefaultWindow returns the default window
func (c *RateLimitConfig) GetDefaultWindow() time.Duration {
	if c.DefaultWindow <= 0 {
		return time.Minute
	}
	return c.DefaultWindow
}

// IsIPEnabled returns whether IP-based rate limiting is enabled
func (c *RateLimitConfig) IsIPEnabled() bool {
	return c.IPEnabled
}

// GetIPRPS returns the IP RPS
func (c *RateLimitConfig) GetIPRPS() int {
	if c.IPRPS <= 0 {
		return 100
	}
	return c.IPRPS
}

// GetIPBurst returns the IP burst
func (c *RateLimitConfig) GetIPBurst() int {
	if c.IPBurst <= 0 {
		return 200
	}
	return c.IPBurst
}

// GetIPWindow returns the IP window
func (c *RateLimitConfig) GetIPWindow() time.Duration {
	if c.IPWindow <= 0 {
		return time.Minute
	}
	return c.IPWindow
}

// IsUserEnabled returns whether user-based rate limiting is enabled
func (c *RateLimitConfig) IsUserEnabled() bool {
	return c.UserEnabled
}

// GetUserRPS returns the user RPS
func (c *RateLimitConfig) GetUserRPS() int {
	if c.UserRPS <= 0 {
		return 50
	}
	return c.UserRPS
}

// GetUserBurst returns the user burst
func (c *RateLimitConfig) GetUserBurst() int {
	if c.UserBurst <= 0 {
		return 100
	}
	return c.UserBurst
}

// GetUserWindow returns the user window
func (c *RateLimitConfig) GetUserWindow() time.Duration {
	if c.UserWindow <= 0 {
		return time.Minute
	}
	return c.UserWindow
}

// IsRouteEnabled returns whether route-based rate limiting is enabled
func (c *RateLimitConfig) IsRouteEnabled() bool {
	return c.RouteEnabled
}

// GetRouteRPS returns the route RPS
func (c *RateLimitConfig) GetRouteRPS() int {
	if c.RouteRPS <= 0 {
		return 200
	}
	return c.RouteRPS
}

// GetRouteBurst returns the route burst
func (c *RateLimitConfig) GetRouteBurst() int {
	if c.RouteBurst <= 0 {
		return 400
	}
	return c.RouteBurst
}

// GetRouteWindow returns the route window
func (c *RateLimitConfig) GetRouteWindow() time.Duration {
	if c.RouteWindow <= 0 {
		return time.Minute
	}
	return c.RouteWindow
}

// IsAPIEnabled returns whether API rate limiting is enabled
func (c *RateLimitConfig) IsAPIEnabled() bool {
	return c.APIEnabled
}

// GetAPIRPS returns the API RPS
func (c *RateLimitConfig) GetAPIRPS() int {
	if c.APIRPS <= 0 {
		return 1000
	}
	return c.APIRPS
}

// GetAPIBurst returns the API burst
func (c *RateLimitConfig) GetAPIBurst() int {
	if c.APIBurst <= 0 {
		return 2000
	}
	return c.APIBurst
}

// GetAPIWindow returns the API window
func (c *RateLimitConfig) GetAPIWindow() time.Duration {
	if c.APIWindow <= 0 {
		return time.Minute
	}
	return c.APIWindow
}

// IsAuthEnabled returns whether auth rate limiting is enabled
func (c *RateLimitConfig) IsAuthEnabled() bool {
	return c.AuthEnabled
}

// GetAuthRPS returns the auth RPS
func (c *RateLimitConfig) GetAuthRPS() int {
	if c.AuthRPS <= 0 {
		return 5
	}
	return c.AuthRPS
}

// GetAuthBurst returns the auth burst
func (c *RateLimitConfig) GetAuthBurst() int {
	if c.AuthBurst <= 0 {
		return 10
	}
	return c.AuthBurst
}

// GetAuthWindow returns the auth window
func (c *RateLimitConfig) GetAuthWindow() time.Duration {
	if c.AuthWindow <= 0 {
		return time.Minute
	}
	return c.AuthWindow
}

// IsUploadEnabled returns whether upload rate limiting is enabled
func (c *RateLimitConfig) IsUploadEnabled() bool {
	return c.UploadEnabled
}

// GetUploadRPS returns the upload RPS
func (c *RateLimitConfig) GetUploadRPS() int {
	if c.UploadRPS <= 0 {
		return 10
	}
	return c.UploadRPS
}

// GetUploadBurst returns the upload burst
func (c *RateLimitConfig) GetUploadBurst() int {
	if c.UploadBurst <= 0 {
		return 20
	}
	return c.UploadBurst
}

// GetUploadWindow returns the upload window
func (c *RateLimitConfig) GetUploadWindow() time.Duration {
	if c.UploadWindow <= 0 {
		return time.Minute
	}
	return c.UploadWindow
}

// GetStorageDriver returns the storage driver
func (c *RateLimitConfig) GetStorageDriver() string {
	if c.StorageDriver == "" {
		return "memory"
	}
	return c.StorageDriver
}

// GetStoragePath returns the storage path
func (c *RateLimitConfig) GetStoragePath() string {
	if c.StoragePath == "" {
		return "./storage/ratelimit"
	}
	return c.StoragePath
}

// GetRedisHost returns the Redis host
func (c *RateLimitConfig) GetRedisHost() string {
	if c.RedisHost == "" {
		return "localhost"
	}
	return c.RedisHost
}

// GetRedisPort returns the Redis port
func (c *RateLimitConfig) GetRedisPort() int {
	if c.RedisPort <= 0 {
		return 6379
	}
	return c.RedisPort
}

// GetRedisPassword returns the Redis password
func (c *RateLimitConfig) GetRedisPassword() string {
	return c.RedisPassword
}

// GetRedisDB returns the Redis database number
func (c *RateLimitConfig) GetRedisDB() int {
	return c.RedisDB
}

// GetRedisAddr returns the Redis address
func (c *RateLimitConfig) GetRedisAddr() string {
	return c.GetRedisHost() + ":" + string(rune(c.GetRedisPort()))
}

// IsCleanupEnabled returns whether cleanup is enabled
func (c *RateLimitConfig) IsCleanupEnabled() bool {
	return c.CleanupEnabled
}

// GetCleanupInterval returns the cleanup interval
func (c *RateLimitConfig) GetCleanupInterval() time.Duration {
	if c.CleanupInterval <= 0 {
		return 5 * time.Minute
	}
	return c.CleanupInterval
}

// IsMonitoringEnabled returns whether monitoring is enabled
func (c *RateLimitConfig) IsMonitoringEnabled() bool {
	return c.MonitoringEnabled
}

// GetMonitoringPath returns the monitoring path
func (c *RateLimitConfig) GetMonitoringPath() string {
	if c.MonitoringPath == "" {
		return "/monitor/ratelimit"
	}
	return c.MonitoringPath
}

// IsMetricsEnabled returns whether metrics are enabled
func (c *RateLimitConfig) IsMetricsEnabled() bool {
	return c.MetricsEnabled
}

// GetMetricsPath returns the metrics path
func (c *RateLimitConfig) GetMetricsPath() string {
	if c.MetricsPath == "" {
		return "/metrics/ratelimit"
	}
	return c.MetricsPath
}

// IsLogEnabled returns whether logging is enabled
func (c *RateLimitConfig) IsLogEnabled() bool {
	return c.LogEnabled
}

// GetLogLevel returns the log level
func (c *RateLimitConfig) GetLogLevel() string {
	if c.LogLevel == "" {
		return "info"
	}
	return c.LogLevel
}

// IsHealthCheckEnabled returns whether health check is enabled
func (c *RateLimitConfig) IsHealthCheckEnabled() bool {
	return c.HealthCheckEnabled
}

// GetHealthCheckInterval returns the health check interval
func (c *RateLimitConfig) GetHealthCheckInterval() time.Duration {
	if c.HealthCheckInterval <= 0 {
		return 30 * time.Second
	}
	return c.HealthCheckInterval
}

// IsWhitelistEnabled returns whether whitelist is enabled
func (c *RateLimitConfig) IsWhitelistEnabled() bool {
	return c.WhitelistEnabled
}

// GetWhitelistIPs returns the whitelist IPs
func (c *RateLimitConfig) GetWhitelistIPs() []string {
	return c.WhitelistIPs
}

// IsBlacklistEnabled returns whether blacklist is enabled
func (c *RateLimitConfig) IsBlacklistEnabled() bool {
	return c.BlacklistEnabled
}

// GetBlacklistIPs returns the blacklist IPs
func (c *RateLimitConfig) GetBlacklistIPs() []string {
	return c.BlacklistIPs
}

// IsCustomRulesEnabled returns whether custom rules are enabled
func (c *RateLimitConfig) IsCustomRulesEnabled() bool {
	return c.CustomRulesEnabled
}

// IsIPWhitelisted checks if an IP is whitelisted
func (c *RateLimitConfig) IsIPWhitelisted(ip string) bool {
	if !c.IsWhitelistEnabled() {
		return false
	}

	whitelistIPs := c.GetWhitelistIPs()
	for _, whitelistedIP := range whitelistIPs {
		if whitelistedIP == ip {
			return true
		}
	}

	return false
}

// IsIPBlacklisted checks if an IP is blacklisted
func (c *RateLimitConfig) IsIPBlacklisted(ip string) bool {
	if !c.IsBlacklistEnabled() {
		return false
	}

	blacklistIPs := c.GetBlacklistIPs()
	for _, blacklistedIP := range blacklistIPs {
		if blacklistedIP == ip {
			return true
		}
	}

	return false
}

// IsMemory returns true if using memory storage
func (c *RateLimitConfig) IsMemory() bool {
	return c.GetStorageDriver() == "memory"
}

// IsRedis returns true if using Redis storage
func (c *RateLimitConfig) IsRedis() bool {
	return c.GetStorageDriver() == "redis"
}
