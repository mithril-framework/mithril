package middleware

import (
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

// LoggerConfig holds configuration for logger middleware
type LoggerConfig struct {
	Format     string                // Log format
	TimeFormat string                // Time format
	TimeZone   string                // Time zone
	Output     *os.File              // Output destination
	SkipPaths  []string              // Paths to skip logging
	SkipFunc   func(*fiber.Ctx) bool // Function to determine if request should be skipped
}

// DefaultLoggerConfig returns default logger configuration
func DefaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Format:     "[${time}] ${status} - ${method} ${path} (${ip}) ${latency}\n",
		TimeFormat: "2006-01-02 15:04:05",
		TimeZone:   "UTC",
		Output:     os.Stdout,
		SkipPaths: []string{
			"/health",
			"/metrics",
			"/favicon.ico",
		},
		SkipFunc: nil,
	}
}

// Logger creates a logger middleware
func Logger(config ...LoggerConfig) fiber.Handler {
	cfg := DefaultLoggerConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	// Convert to Fiber logger config
	fiberConfig := logger.Config{
		Format:     cfg.Format,
		TimeFormat: cfg.TimeFormat,
		TimeZone:   cfg.TimeZone,
		Output:     cfg.Output,
	}

	return logger.New(fiberConfig)
}

// DevelopmentLogger creates a logger middleware for development
func DevelopmentLogger() fiber.Handler {
	return Logger(LoggerConfig{
		Format:     "[${time}] ${status} - ${method} ${path} (${ip}) ${latency} ${bytesSent}B\n",
		TimeFormat: "15:04:05",
		TimeZone:   "Local",
		Output:     os.Stdout,
		SkipPaths: []string{
			"/health",
			"/metrics",
		},
	})
}

// ProductionLogger creates a logger middleware for production
func ProductionLogger() fiber.Handler {
	return Logger(LoggerConfig{
		Format:     `{"time":"${time}","status":${status},"method":"${method}","path":"${path}","ip":"${ip}","latency":"${latency}","bytes_sent":${bytesSent}}` + "\n",
		TimeFormat: time.RFC3339,
		TimeZone:   "UTC",
		Output:     os.Stdout,
		SkipPaths: []string{
			"/health",
			"/metrics",
			"/favicon.ico",
		},
	})
}

// APILogger creates a logger middleware for API endpoints
func APILogger() fiber.Handler {
	return Logger(LoggerConfig{
		Format:     `{"timestamp":"${time}","level":"info","status":${status},"method":"${method}","path":"${path}","ip":"${ip}","latency":"${latency}","bytes_sent":${bytesSent},"user_agent":"${ua}","referer":"${referer}"}` + "\n",
		TimeFormat: time.RFC3339,
		TimeZone:   "UTC",
		Output:     os.Stdout,
		SkipPaths: []string{
			"/health",
			"/metrics",
		},
	})
}

// WebLogger creates a logger middleware for web endpoints
func WebLogger() fiber.Handler {
	return Logger(LoggerConfig{
		Format:     "[${time}] ${status} - ${method} ${path} (${ip}) ${latency} ${bytesSent}B \"${ua}\"\n",
		TimeFormat: "2006-01-02 15:04:05",
		TimeZone:   "Local",
		Output:     os.Stdout,
		SkipPaths: []string{
			"/health",
			"/metrics",
			"/favicon.ico",
			"/robots.txt",
		},
	})
}

// ErrorLogger creates a logger middleware that only logs errors
func ErrorLogger() fiber.Handler {
	return Logger(LoggerConfig{
		Format:     "[${time}] ERROR ${status} - ${method} ${path} (${ip}) ${latency} ${error}\n",
		TimeFormat: "2006-01-02 15:04:05",
		TimeZone:   "UTC",
		Output:     os.Stderr,
		SkipFunc: func(c *fiber.Ctx) bool {
			// Only log if status code is 4xx or 5xx
			return c.Response().StatusCode() < 400
		},
	})
}

// AccessLogger creates a logger middleware for access logs
func AccessLogger() fiber.Handler {
	return Logger(LoggerConfig{
		Format:     `{"timestamp":"${time}","method":"${method}","path":"${path}","status":${status},"ip":"${ip}","latency":"${latency}","bytes_sent":${bytesSent},"user_agent":"${ua}","referer":"${referer}"}` + "\n",
		TimeFormat: time.RFC3339,
		TimeZone:   "UTC",
		Output:     os.Stdout,
		SkipPaths: []string{
			"/health",
			"/metrics",
		},
	})
}

// CustomLogger creates a logger middleware with custom configuration
func CustomLogger(format string, output *os.File) fiber.Handler {
	return Logger(LoggerConfig{
		Format:     format,
		TimeFormat: time.RFC3339,
		TimeZone:   "UTC",
		Output:     output,
	})
}
