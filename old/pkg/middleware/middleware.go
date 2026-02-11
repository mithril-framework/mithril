package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// MiddlewareStack represents a collection of middleware
type MiddlewareStack struct {
	middlewares []fiber.Handler
}

// NewMiddlewareStack creates a new middleware stack
func NewMiddlewareStack() *MiddlewareStack {
	return &MiddlewareStack{
		middlewares: make([]fiber.Handler, 0),
	}
}

// Add adds a middleware to the stack
func (ms *MiddlewareStack) Add(middleware fiber.Handler) *MiddlewareStack {
	ms.middlewares = append(ms.middlewares, middleware)
	return ms
}

// Apply applies all middleware to the app
func (ms *MiddlewareStack) Apply(app *fiber.App) {
	for _, middleware := range ms.middlewares {
		app.Use(middleware)
	}
}

// ApplyToGroup applies all middleware to a group
func (ms *MiddlewareStack) ApplyToGroup(group fiber.Router) {
	for _, middleware := range ms.middlewares {
		group.Use(middleware)
	}
}

// DefaultStack returns the default middleware stack
func DefaultStack() *MiddlewareStack {
	return NewMiddlewareStack().
		Add(RequestID()).
		Add(Logger()).
		Add(Recovery()).
		Add(CORS()).
		Add(Security()).
		Add(Compression())
}

// DevelopmentStack returns the development middleware stack
func DevelopmentStack() *MiddlewareStack {
	return NewMiddlewareStack().
		Add(RequestID()).
		Add(DevelopmentLogger()).
		Add(DevelopmentRecovery()).
		Add(DevelopmentCORS()).
		Add(DevelopmentSecurity()).
		Add(Compression())
}

// ProductionStack returns the production middleware stack
func ProductionStack() *MiddlewareStack {
	return NewMiddlewareStack().
		Add(RequestID()).
		Add(ProductionLogger()).
		Add(ProductionRecovery()).
		Add(ProductionCORS([]string{"https://yourdomain.com"})).
		Add(ProductionSecurity()).
		Add(Compression())
}

// APIStack returns the API middleware stack
func APIStack() *MiddlewareStack {
	return NewMiddlewareStack().
		Add(RequestID()).
		Add(APILogger()).
		Add(APIRecovery()).
		Add(APICORS()).
		Add(APISecurity()).
		Add(APICompression())
}

// WebStack returns the web middleware stack
func WebStack() *MiddlewareStack {
	return NewMiddlewareStack().
		Add(RequestID()).
		Add(WebLogger()).
		Add(WebRecovery()).
		Add(WebCORS([]string{"https://yourdomain.com"})).
		Add(WebSecurity()).
		Add(WebCompression())
}

// AuthStack returns the authentication middleware stack
func AuthStack() *MiddlewareStack {
	return NewMiddlewareStack().
		Add(RequestID()).
		Add(Logger()).
		Add(Recovery()).
		Add(CORS()).
		Add(Security()).
		Add(Compression()).
		Add(SessionMiddleware()).
		Add(CSRFMiddleware())
}

// RateLimitedStack returns a rate-limited middleware stack
func RateLimitedStack() *MiddlewareStack {
	return NewMiddlewareStack().
		Add(RequestID()).
		Add(Logger()).
		Add(Recovery()).
		Add(CORS()).
		Add(Security()).
		Add(Compression()).
		Add(RateLimitByIP(100, time.Minute))
}

// SecureStack returns a highly secure middleware stack
func SecureStack() *MiddlewareStack {
	return NewMiddlewareStack().
		Add(RequestID()).
		Add(ProductionLogger()).
		Add(ProductionRecovery()).
		Add(ProductionCORS([]string{"https://yourdomain.com"})).
		Add(ProductionSecurity()).
		Add(Compression()).
		Add(SessionMiddleware()).
		Add(CSRFMiddleware()).
		Add(RateLimitByIP(50, time.Minute))
}

// CustomStack creates a custom middleware stack
func CustomStack(middlewares ...fiber.Handler) *MiddlewareStack {
	stack := NewMiddlewareStack()
	for _, middleware := range middlewares {
		stack.Add(middleware)
	}
	return stack
}

// Environment-specific middleware stacks
func GetStackForEnvironment(env string) *MiddlewareStack {
	switch env {
	case "development":
		return DevelopmentStack()
	case "production":
		return ProductionStack()
	case "api":
		return APIStack()
	case "web":
		return WebStack()
	case "auth":
		return AuthStack()
	case "rate_limited":
		return RateLimitedStack()
	case "secure":
		return SecureStack()
	default:
		return DefaultStack()
	}
}

// Middleware configuration helpers
type MiddlewareConfig struct {
	Environment       string
	EnableCORS        bool
	EnableCSRF        bool
	EnableSession     bool
	EnableRateLimit   bool
	EnableCompression bool
	EnableSecurity    bool
	EnableLogging     bool
	EnableRecovery    bool
	EnableRequestID   bool
}

// ConfiguredStack creates a middleware stack based on configuration
func ConfiguredStack(config MiddlewareConfig) *MiddlewareStack {
	stack := NewMiddlewareStack()

	if config.EnableRequestID {
		stack.Add(RequestID())
	}

	if config.EnableLogging {
		if config.Environment == "production" {
			stack.Add(ProductionLogger())
		} else {
			stack.Add(DevelopmentLogger())
		}
	}

	if config.EnableRecovery {
		if config.Environment == "production" {
			stack.Add(ProductionRecovery())
		} else {
			stack.Add(DevelopmentRecovery())
		}
	}

	if config.EnableCORS {
		if config.Environment == "production" {
			stack.Add(ProductionCORS([]string{"https://yourdomain.com"}))
		} else {
			stack.Add(DevelopmentCORS())
		}
	}

	if config.EnableSecurity {
		if config.Environment == "production" {
			stack.Add(ProductionSecurity())
		} else {
			stack.Add(DevelopmentSecurity())
		}
	}

	if config.EnableCompression {
		stack.Add(Compression())
	}

	if config.EnableSession {
		stack.Add(SessionMiddleware())
	}

	if config.EnableCSRF {
		stack.Add(CSRFMiddleware())
	}

	if config.EnableRateLimit {
		stack.Add(RateLimitByIP(100, time.Minute))
	}

	return stack
}

// Default configuration
func DefaultConfig() MiddlewareConfig {
	return MiddlewareConfig{
		Environment:       "development",
		EnableCORS:        true,
		EnableCSRF:        false,
		EnableSession:     false,
		EnableRateLimit:   false,
		EnableCompression: true,
		EnableSecurity:    true,
		EnableLogging:     true,
		EnableRecovery:    true,
		EnableRequestID:   true,
	}
}

// Production configuration
func ProductionConfig() MiddlewareConfig {
	return MiddlewareConfig{
		Environment:       "production",
		EnableCORS:        true,
		EnableCSRF:        true,
		EnableSession:     true,
		EnableRateLimit:   true,
		EnableCompression: true,
		EnableSecurity:    true,
		EnableLogging:     true,
		EnableRecovery:    true,
		EnableRequestID:   true,
	}
}

// API configuration
func APIConfig() MiddlewareConfig {
	return MiddlewareConfig{
		Environment:       "api",
		EnableCORS:        true,
		EnableCSRF:        false,
		EnableSession:     false,
		EnableRateLimit:   true,
		EnableCompression: true,
		EnableSecurity:    true,
		EnableLogging:     true,
		EnableRecovery:    true,
		EnableRequestID:   true,
	}
}

// Web configuration
func WebConfig() MiddlewareConfig {
	return MiddlewareConfig{
		Environment:       "web",
		EnableCORS:        true,
		EnableCSRF:        true,
		EnableSession:     true,
		EnableRateLimit:   false,
		EnableCompression: true,
		EnableSecurity:    true,
		EnableLogging:     true,
		EnableRecovery:    true,
		EnableRequestID:   true,
	}
}
