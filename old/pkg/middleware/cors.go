package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// CORSConfig holds configuration for CORS middleware
type CORSConfig struct {
	AllowOrigins     []string // Allowed origins
	AllowMethods     []string // Allowed HTTP methods
	AllowHeaders     []string // Allowed headers
	ExposeHeaders    []string // Headers to expose to client
	AllowCredentials bool     // Allow credentials
	MaxAge           int      // Max age for preflight requests
}

// DefaultCORSConfig returns default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS",
		},
		AllowHeaders: []string{
			"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With",
			"X-CSRF-Token", "X-Request-ID", "X-Forwarded-For", "X-Real-IP",
		},
		ExposeHeaders: []string{
			"Content-Length", "Content-Type", "X-Request-ID",
		},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}

// CORS creates a CORS middleware
func CORS(config ...CORSConfig) fiber.Handler {
	cfg := DefaultCORSConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	// Convert to Fiber CORS config
	fiberConfig := cors.Config{
		AllowOrigins:     strings.Join(cfg.AllowOrigins, ","),
		AllowMethods:     strings.Join(cfg.AllowMethods, ","),
		AllowHeaders:     strings.Join(cfg.AllowHeaders, ","),
		ExposeHeaders:    strings.Join(cfg.ExposeHeaders, ","),
		AllowCredentials: cfg.AllowCredentials,
		MaxAge:           cfg.MaxAge,
	}

	return cors.New(fiberConfig)
}

// DevelopmentCORS creates a CORS middleware for development
func DevelopmentCORS() fiber.Handler {
	return CORS(CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"*"},
		AllowHeaders:     []string{"*"},
		AllowCredentials: false,
	})
}

// ProductionCORS creates a CORS middleware for production
func ProductionCORS(allowedOrigins []string) fiber.Handler {
	return CORS(CORSConfig{
		AllowOrigins: allowedOrigins,
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{
			"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With",
			"X-CSRF-Token", "X-Request-ID",
		},
		ExposeHeaders:    []string{"Content-Length", "Content-Type", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           86400,
	})
}

// APICORS creates a CORS middleware specifically for API endpoints
func APICORS() fiber.Handler {
	return CORS(CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{
			"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With",
			"X-API-Key", "X-Request-ID",
		},
		ExposeHeaders:    []string{"Content-Length", "Content-Type", "X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           3600, // 1 hour
	})
}

// WebCORS creates a CORS middleware for web endpoints
func WebCORS(allowedOrigins []string) fiber.Handler {
	return CORS(CORSConfig{
		AllowOrigins: allowedOrigins,
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{
			"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With",
			"X-CSRF-Token", "X-Request-ID",
		},
		ExposeHeaders:    []string{"Content-Length", "Content-Type", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           86400,
	})
}
