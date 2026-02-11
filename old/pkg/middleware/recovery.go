package middleware

import (
	"log"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// RecoveryConfig holds configuration for recovery middleware
type RecoveryConfig struct {
	EnableStackTrace bool                          // Enable stack trace in response
	StackTraceSize   int                           // Stack trace size limit
	ErrorHandler     func(*fiber.Ctx, error) error // Custom error handler
}

// DefaultRecoveryConfig returns default recovery configuration
func DefaultRecoveryConfig() RecoveryConfig {
	return RecoveryConfig{
		EnableStackTrace: false,    // Disable in production
		StackTraceSize:   1024 * 4, // 4KB
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}

			response := fiber.Map{
				"error":   true,
				"message": "Internal Server Error",
				"code":    code,
			}

			// Add stack trace in development
			if c.Locals("env") == "development" {
				response["stack"] = string(debug.Stack())
			}

			return c.Status(code).JSON(response)
		},
	}
}

// Recovery creates a recovery middleware
func Recovery(config ...RecoveryConfig) fiber.Handler {
	cfg := DefaultRecoveryConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	// Use Fiber's built-in recovery middleware
	fiberConfig := recover.Config{
		EnableStackTrace: cfg.EnableStackTrace,
	}

	return recover.New(fiberConfig)
}

// DevelopmentRecovery creates a recovery middleware for development
func DevelopmentRecovery() fiber.Handler {
	return Recovery(RecoveryConfig{
		EnableStackTrace: true,
		StackTraceSize:   1024 * 8, // 8KB
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}

			// Log the error
			log.Printf("PANIC: %v\n%s", err, debug.Stack())

			return c.Status(code).JSON(fiber.Map{
				"error":   true,
				"message": err.Error(),
				"code":    code,
				"stack":   string(debug.Stack()),
			})
		},
	})
}

// ProductionRecovery creates a recovery middleware for production
func ProductionRecovery() fiber.Handler {
	return Recovery(RecoveryConfig{
		EnableStackTrace: false,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}

			// Log the error without stack trace
			log.Printf("PANIC: %v", err)

			return c.Status(code).JSON(fiber.Map{
				"error":   true,
				"message": "Internal Server Error",
				"code":    code,
			})
		},
	})
}

// APIRecovery creates a recovery middleware for API endpoints
func APIRecovery() fiber.Handler {
	return Recovery(RecoveryConfig{
		EnableStackTrace: false,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}

			// Log the error
			log.Printf("API PANIC: %v", err)

			return c.Status(code).JSON(fiber.Map{
				"error":   true,
				"message": "Internal Server Error",
				"code":    code,
			})
		},
	})
}

// WebRecovery creates a recovery middleware for web endpoints
func WebRecovery() fiber.Handler {
	return Recovery(RecoveryConfig{
		EnableStackTrace: false,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}

			// Log the error
			log.Printf("WEB PANIC: %v", err)

			// For web requests, redirect to error page or return HTML
			if c.Get("Accept") == "text/html" {
				return c.Status(code).Render("errors/500", fiber.Map{
					"error":   true,
					"message": "Internal Server Error",
					"code":    code,
				})
			}

			return c.Status(code).JSON(fiber.Map{
				"error":   true,
				"message": "Internal Server Error",
				"code":    code,
			})
		},
	})
}

// CustomRecovery creates a recovery middleware with custom error handler
func CustomRecovery(errorHandler func(*fiber.Ctx, error) error) fiber.Handler {
	return Recovery(RecoveryConfig{
		EnableStackTrace: false,
		ErrorHandler:     errorHandler,
	})
}
