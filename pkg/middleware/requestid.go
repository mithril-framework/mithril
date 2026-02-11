package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

// RequestIDConfig holds configuration for request ID middleware
type RequestIDConfig struct {
	Header     string        // Header name for request ID
	Generator  func() string // Function to generate request ID
	ContextKey string        // Key to store request ID in context
}

// DefaultRequestIDConfig returns default request ID configuration
func DefaultRequestIDConfig() RequestIDConfig {
	return RequestIDConfig{
		Header:     "X-Request-ID",
		Generator:  generateRequestID,
		ContextKey: "request_id",
	}
}

// generateRequestID generates a random request ID
func generateRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID
		return strings.ReplaceAll(strings.ToUpper(hex.EncodeToString(bytes)), "-", "")
	}
	return strings.ToUpper(hex.EncodeToString(bytes))
}

// RequestID creates a request ID middleware
func RequestID(config ...RequestIDConfig) fiber.Handler {
	cfg := DefaultRequestIDConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	// Use Fiber's built-in request ID middleware
	fiberConfig := requestid.Config{
		Header:     cfg.Header,
		Generator:  cfg.Generator,
		ContextKey: cfg.ContextKey,
	}

	return requestid.New(fiberConfig)
}

// GetRequestID retrieves the request ID from context
func GetRequestID(c *fiber.Ctx) string {
	if requestID := c.Locals("request_id"); requestID != nil {
		return requestID.(string)
	}
	return ""
}

// SetRequestID sets a custom request ID in context
func SetRequestID(c *fiber.Ctx, requestID string) {
	c.Locals("request_id", requestID)
	c.Set("X-Request-ID", requestID)
}

// RequestIDFromHeader creates a request ID middleware that uses existing header
func RequestIDFromHeader(headerName string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := c.Get(headerName)
		if requestID == "" {
			requestID = generateRequestID()
		}

		c.Locals("request_id", requestID)
		c.Set("X-Request-ID", requestID)

		return c.Next()
	}
}

// RequestIDWithPrefix creates a request ID middleware with a prefix
func RequestIDWithPrefix(prefix string) fiber.Handler {
	return RequestID(RequestIDConfig{
		Header: "X-Request-ID",
		Generator: func() string {
			return prefix + "-" + generateRequestID()
		},
		ContextKey: "request_id",
	})
}

// UUIDRequestID creates a request ID middleware that generates UUID-style IDs
func UUIDRequestID() fiber.Handler {
	return RequestID(RequestIDConfig{
		Header: "X-Request-ID",
		Generator: func() string {
			bytes := make([]byte, 16)
			if _, err := rand.Read(bytes); err != nil {
				return generateRequestID()
			}

			// Format as UUID
			hexStr := hex.EncodeToString(bytes)
			return strings.Join([]string{
				hexStr[0:8],
				hexStr[8:12],
				hexStr[12:16],
				hexStr[16:20],
				hexStr[20:32],
			}, "-")
		},
		ContextKey: "request_id",
	})
}

// ShortRequestID creates a request ID middleware that generates short IDs
func ShortRequestID() fiber.Handler {
	return RequestID(RequestIDConfig{
		Header: "X-Request-ID",
		Generator: func() string {
			bytes := make([]byte, 8)
			if _, err := rand.Read(bytes); err != nil {
				return generateRequestID()[:8]
			}
			return strings.ToUpper(hex.EncodeToString(bytes))
		},
		ContextKey: "request_id",
	})
}
