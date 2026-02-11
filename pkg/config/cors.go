package config

// CORSConfig holds CORS configuration
type CORSConfig struct {
	// Basic CORS settings
	Enabled          bool     `env:"CORS_ENABLED" default:"true"`
	AllowedOrigins   []string `env:"CORS_ALLOWED_ORIGINS" default:"*"`
	AllowedMethods   []string `env:"CORS_ALLOWED_METHODS" default:"GET,POST,PUT,DELETE,OPTIONS"`
	AllowedHeaders   []string `env:"CORS_ALLOWED_HEADERS" default:"Content-Type,Authorization,X-Requested-With"`
	ExposedHeaders   []string `env:"CORS_EXPOSED_HEADERS" default:""`
	AllowCredentials bool     `env:"CORS_ALLOW_CREDENTIALS" default:"true"`
	MaxAge           int      `env:"CORS_MAX_AGE" default:"86400"` // 24 hours

	// Advanced settings
	AllowWildcard          bool `env:"CORS_ALLOW_WILDCARD" default:"true"`
	AllowBrowserExtensions bool `env:"CORS_ALLOW_BROWSER_EXTENSIONS" default:"false"`
	AllowWebSockets        bool `env:"CORS_ALLOW_WEB_SOCKETS" default:"true"`
	AllowFiles             bool `env:"CORS_ALLOW_FILES" default:"false"`

	// Security settings
	AllowPrivateNetwork bool   `env:"CORS_ALLOW_PRIVATE_NETWORK" default:"false"`
	VaryHeader          string `env:"CORS_VARY_HEADER" default:"Origin"`

	// Debug settings
	Debug    bool   `env:"CORS_DEBUG" default:"false"`
	LogLevel string `env:"CORS_LOG_LEVEL" default:"info"`

	// Custom settings
	CustomHeaders map[string]string `env:"CORS_CUSTOM_HEADERS" default:""`
	CustomMethods []string          `env:"CORS_CUSTOM_METHODS" default:""`
	CustomOrigins []string          `env:"CORS_CUSTOM_ORIGINS" default:""`

	// Rate limiting
	RateLimitEnabled bool `env:"CORS_RATE_LIMIT_ENABLED" default:"false"`
	RateLimitRPS     int  `env:"CORS_RATE_LIMIT_RPS" default:"100"`
	RateLimitBurst   int  `env:"CORS_RATE_LIMIT_BURST" default:"200"`

	// Health check
	HealthCheckEnabled bool   `env:"CORS_HEALTH_CHECK_ENABLED" default:"true"`
	HealthCheckPath    string `env:"CORS_HEALTH_CHECK_PATH" default:"/health"`
}

// IsEnabled returns whether CORS is enabled
func (c *CORSConfig) IsEnabled() bool {
	return c.Enabled
}

// GetAllowedOrigins returns the allowed origins
func (c *CORSConfig) GetAllowedOrigins() []string {
	if len(c.AllowedOrigins) == 0 {
		return []string{"*"}
	}
	return c.AllowedOrigins
}

// GetAllowedMethods returns the allowed methods
func (c *CORSConfig) GetAllowedMethods() []string {
	if len(c.AllowedMethods) == 0 {
		return []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	return c.AllowedMethods
}

// GetAllowedHeaders returns the allowed headers
func (c *CORSConfig) GetAllowedHeaders() []string {
	if len(c.AllowedHeaders) == 0 {
		return []string{"Content-Type", "Authorization", "X-Requested-With"}
	}
	return c.AllowedHeaders
}

// GetExposedHeaders returns the exposed headers
func (c *CORSConfig) GetExposedHeaders() []string {
	return c.ExposedHeaders
}

// IsAllowCredentials returns whether credentials are allowed
func (c *CORSConfig) IsAllowCredentials() bool {
	return c.AllowCredentials
}

// GetMaxAge returns the max age in seconds
func (c *CORSConfig) GetMaxAge() int {
	if c.MaxAge <= 0 {
		return 86400 // 24 hours
	}
	return c.MaxAge
}

// IsAllowWildcard returns whether wildcard is allowed
func (c *CORSConfig) IsAllowWildcard() bool {
	return c.AllowWildcard
}

// IsAllowBrowserExtensions returns whether browser extensions are allowed
func (c *CORSConfig) IsAllowBrowserExtensions() bool {
	return c.AllowBrowserExtensions
}

// IsAllowWebSockets returns whether WebSockets are allowed
func (c *CORSConfig) IsAllowWebSockets() bool {
	return c.AllowWebSockets
}

// IsAllowFiles returns whether files are allowed
func (c *CORSConfig) IsAllowFiles() bool {
	return c.AllowFiles
}

// IsAllowPrivateNetwork returns whether private network is allowed
func (c *CORSConfig) IsAllowPrivateNetwork() bool {
	return c.AllowPrivateNetwork
}

// GetVaryHeader returns the vary header
func (c *CORSConfig) GetVaryHeader() string {
	if c.VaryHeader == "" {
		return "Origin"
	}
	return c.VaryHeader
}

// IsDebug returns whether debug mode is enabled
func (c *CORSConfig) IsDebug() bool {
	return c.Debug
}

// GetLogLevel returns the log level
func (c *CORSConfig) GetLogLevel() string {
	if c.LogLevel == "" {
		return "info"
	}
	return c.LogLevel
}

// GetCustomHeaders returns the custom headers
func (c *CORSConfig) GetCustomHeaders() map[string]string {
	return c.CustomHeaders
}

// GetCustomMethods returns the custom methods
func (c *CORSConfig) GetCustomMethods() []string {
	return c.CustomMethods
}

// GetCustomOrigins returns the custom origins
func (c *CORSConfig) GetCustomOrigins() []string {
	return c.CustomOrigins
}

// IsRateLimitEnabled returns whether rate limiting is enabled
func (c *CORSConfig) IsRateLimitEnabled() bool {
	return c.RateLimitEnabled
}

// GetRateLimitRPS returns the rate limit RPS
func (c *CORSConfig) GetRateLimitRPS() int {
	if c.RateLimitRPS <= 0 {
		return 100
	}
	return c.RateLimitRPS
}

// GetRateLimitBurst returns the rate limit burst
func (c *CORSConfig) GetRateLimitBurst() int {
	if c.RateLimitBurst <= 0 {
		return 200
	}
	return c.RateLimitBurst
}

// IsHealthCheckEnabled returns whether health check is enabled
func (c *CORSConfig) IsHealthCheckEnabled() bool {
	return c.HealthCheckEnabled
}

// GetHealthCheckPath returns the health check path
func (c *CORSConfig) GetHealthCheckPath() string {
	if c.HealthCheckPath == "" {
		return "/health"
	}
	return c.HealthCheckPath
}

// IsOriginAllowed checks if an origin is allowed
func (c *CORSConfig) IsOriginAllowed(origin string) bool {
	if !c.IsEnabled() {
		return false
	}

	allowedOrigins := c.GetAllowedOrigins()

	// Check for wildcard
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		if allowed == origin {
			return true
		}
	}

	// Check custom origins
	customOrigins := c.GetCustomOrigins()
	for _, allowed := range customOrigins {
		if allowed == origin {
			return true
		}
	}

	return false
}

// IsMethodAllowed checks if a method is allowed
func (c *CORSConfig) IsMethodAllowed(method string) bool {
	if !c.IsEnabled() {
		return false
	}

	allowedMethods := c.GetAllowedMethods()

	for _, allowed := range allowedMethods {
		if allowed == method {
			return true
		}
	}

	// Check custom methods
	customMethods := c.GetCustomMethods()
	for _, allowed := range customMethods {
		if allowed == method {
			return true
		}
	}

	return false
}

// IsHeaderAllowed checks if a header is allowed
func (c *CORSConfig) IsHeaderAllowed(header string) bool {
	if !c.IsEnabled() {
		return false
	}

	allowedHeaders := c.GetAllowedHeaders()

	for _, allowed := range allowedHeaders {
		if allowed == "*" {
			return true
		}
		if allowed == header {
			return true
		}
	}

	return false
}
