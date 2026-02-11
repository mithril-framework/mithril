package middleware

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

// SecurityConfig holds configuration for security middleware
type SecurityConfig struct {
	XSSProtection             bool   // XSS Protection header
	ContentTypeNosniff        bool   // Content Type No Sniff header
	XFrameOptions             string // X-Frame-Options header value
	XContentTypeOptions       bool   // X-Content-Type-Options header
	ReferrerPolicy            string // Referrer-Policy header value
	PermissionsPolicy         string // Permissions-Policy header value
	StrictTransportSecurity   string // Strict-Transport-Security header value
	ContentSecurityPolicy     string // Content-Security-Policy header value
	ExpectCT                  string // Expect-CT header value
	CrossOriginEmbedderPolicy string // Cross-Origin-Embedder-Policy header value
	CrossOriginOpenerPolicy   string // Cross-Origin-Opener-Policy header value
	CrossOriginResourcePolicy string // Cross-Origin-Resource-Policy header value
}

// DefaultSecurityConfig returns default security configuration
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		XSSProtection:             true,
		ContentTypeNosniff:        true,
		XFrameOptions:             "DENY",
		XContentTypeOptions:       true,
		ReferrerPolicy:            "strict-origin-when-cross-origin",
		PermissionsPolicy:         "geolocation=(), microphone=(), camera=()",
		StrictTransportSecurity:   "max-age=31536000; includeSubDomains",
		ContentSecurityPolicy:     "default-src 'self'",
		ExpectCT:                  "max-age=86400, enforce",
		CrossOriginEmbedderPolicy: "require-corp",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
	}
}

// Security creates a security middleware
func Security(config ...SecurityConfig) fiber.Handler {
	cfg := DefaultSecurityConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *fiber.Ctx) error {
		// XSS Protection
		if cfg.XSSProtection {
			c.Set("X-XSS-Protection", "1; mode=block")
		}

		// Content Type No Sniff
		if cfg.ContentTypeNosniff {
			c.Set("X-Content-Type-Options", "nosniff")
		}

		// X-Frame-Options
		if cfg.XFrameOptions != "" {
			c.Set("X-Frame-Options", cfg.XFrameOptions)
		}

		// Referrer Policy
		if cfg.ReferrerPolicy != "" {
			c.Set("Referrer-Policy", cfg.ReferrerPolicy)
		}

		// Permissions Policy
		if cfg.PermissionsPolicy != "" {
			c.Set("Permissions-Policy", cfg.PermissionsPolicy)
		}

		// Strict Transport Security (only for HTTPS)
		if cfg.StrictTransportSecurity != "" && c.Secure() {
			c.Set("Strict-Transport-Security", cfg.StrictTransportSecurity)
		}

		// Content Security Policy
		if cfg.ContentSecurityPolicy != "" {
			c.Set("Content-Security-Policy", cfg.ContentSecurityPolicy)
		}

		// Expect-CT
		if cfg.ExpectCT != "" {
			c.Set("Expect-CT", cfg.ExpectCT)
		}

		// Cross-Origin-Embedder-Policy
		if cfg.CrossOriginEmbedderPolicy != "" {
			c.Set("Cross-Origin-Embedder-Policy", cfg.CrossOriginEmbedderPolicy)
		}

		// Cross-Origin-Opener-Policy
		if cfg.CrossOriginOpenerPolicy != "" {
			c.Set("Cross-Origin-Opener-Policy", cfg.CrossOriginOpenerPolicy)
		}

		// Cross-Origin-Resource-Policy
		if cfg.CrossOriginResourcePolicy != "" {
			c.Set("Cross-Origin-Resource-Policy", cfg.CrossOriginResourcePolicy)
		}

		return c.Next()
	}
}

// Helmet creates a security middleware with helmet-style headers
func Helmet() fiber.Handler {
	return Security(SecurityConfig{
		XSSProtection:           true,
		ContentTypeNosniff:      true,
		XFrameOptions:           "DENY",
		XContentTypeOptions:     true,
		ReferrerPolicy:          "strict-origin-when-cross-origin",
		PermissionsPolicy:       "geolocation=(), microphone=(), camera=()",
		StrictTransportSecurity: "max-age=31536000; includeSubDomains",
		ContentSecurityPolicy:   "default-src 'self'",
	})
}

// DevelopmentSecurity creates a security middleware for development
func DevelopmentSecurity() fiber.Handler {
	return Security(SecurityConfig{
		XSSProtection:       true,
		ContentTypeNosniff:  true,
		XFrameOptions:       "SAMEORIGIN",
		XContentTypeOptions: true,
		ReferrerPolicy:      "no-referrer",
	})
}

// ProductionSecurity creates a security middleware for production
func ProductionSecurity() fiber.Handler {
	return Security(SecurityConfig{
		XSSProtection:             true,
		ContentTypeNosniff:        true,
		XFrameOptions:             "DENY",
		XContentTypeOptions:       true,
		ReferrerPolicy:            "strict-origin-when-cross-origin",
		PermissionsPolicy:         "geolocation=(), microphone=(), camera=()",
		StrictTransportSecurity:   "max-age=31536000; includeSubDomains; preload",
		ContentSecurityPolicy:     "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'",
		ExpectCT:                  "max-age=86400, enforce",
		CrossOriginEmbedderPolicy: "require-corp",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
	})
}

// APISecurity creates a security middleware for API endpoints
func APISecurity() fiber.Handler {
	return Security(SecurityConfig{
		XSSProtection:         true,
		ContentTypeNosniff:    true,
		XFrameOptions:         "DENY",
		XContentTypeOptions:   true,
		ReferrerPolicy:        "no-referrer",
		ContentSecurityPolicy: "default-src 'none'",
	})
}

// WebSecurity creates a security middleware for web endpoints
func WebSecurity() fiber.Handler {
	return Security(SecurityConfig{
		XSSProtection:             true,
		ContentTypeNosniff:        true,
		XFrameOptions:             "SAMEORIGIN",
		XContentTypeOptions:       true,
		ReferrerPolicy:            "strict-origin-when-cross-origin",
		PermissionsPolicy:         "geolocation=(), microphone=(), camera=()",
		ContentSecurityPolicy:     "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'",
		CrossOriginEmbedderPolicy: "require-corp",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
	})
}

// CacheControl creates a cache control middleware
func CacheControl(maxAge time.Duration, public bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		directive := "private"
		if public {
			directive = "public"
		}

		cacheValue := directive + ", max-age=" + strconv.Itoa(int(maxAge.Seconds()))
		c.Set("Cache-Control", cacheValue)

		return c.Next()
	}
}

// NoCache creates a no-cache middleware
func NoCache() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Set("Pragma", "no-cache")
		c.Set("Expires", "0")

		return c.Next()
	}
}

// ETag creates an ETag middleware
func ETag() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Generate ETag based on response content
		// This is a simple implementation - in production, use proper ETag generation
		etag := `"` + strconv.FormatInt(time.Now().Unix(), 10) + `"`

		// Check if client has the same ETag
		if c.Get("If-None-Match") == etag {
			return c.SendStatus(304)
		}

		c.Set("ETag", etag)
		return c.Next()
	}
}

// Vary creates a Vary header middleware
func Vary(headers ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if len(headers) > 0 {
			varyValue := ""
			for i, header := range headers {
				if i > 0 {
					varyValue += ", "
				}
				varyValue += header
			}
			c.Set("Vary", varyValue)
		}

		return c.Next()
	}
}
