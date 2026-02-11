package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
)

// CompressionConfig holds configuration for compression middleware
type CompressionConfig struct {
	Level    int      // Compression level (1-9)
	Types    []string // Content types to compress
	MinSize  int      // Minimum size to compress
	Excluded []string // Excluded paths
}

// DefaultCompressionConfig returns default compression configuration
func DefaultCompressionConfig() CompressionConfig {
	return CompressionConfig{
		Level: 6, // Default compression level
		Types: []string{
			"text/plain",
			"text/html",
			"text/css",
			"text/javascript",
			"application/javascript",
			"application/json",
			"application/xml",
			"application/rss+xml",
			"application/atom+xml",
			"image/svg+xml",
		},
		MinSize: 1024, // 1KB minimum size
		Excluded: []string{
			"/health",
			"/metrics",
		},
	}
}

// Compression creates a compression middleware
func Compression(config ...CompressionConfig) fiber.Handler {
	cfg := DefaultCompressionConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	// Convert to Fiber compress config
	fiberConfig := compress.Config{
		Level: compress.Level(cfg.Level),
	}

	return compress.New(fiberConfig)
}

// Gzip creates a GZIP compression middleware
func Gzip() fiber.Handler {
	return Compression(CompressionConfig{
		Level: 6,
		Types: []string{
			"text/plain",
			"text/html",
			"text/css",
			"text/javascript",
			"application/javascript",
			"application/json",
			"application/xml",
			"application/rss+xml",
			"application/atom+xml",
			"image/svg+xml",
		},
		MinSize: 1024,
	})
}

// Brotli creates a Brotli compression middleware
func Brotli() fiber.Handler {
	return Compression(CompressionConfig{
		Level: 6,
		Types: []string{
			"text/plain",
			"text/html",
			"text/css",
			"text/javascript",
			"application/javascript",
			"application/json",
			"application/xml",
			"application/rss+xml",
			"application/atom+xml",
			"image/svg+xml",
		},
		MinSize: 1024,
	})
}

// Deflate creates a Deflate compression middleware
func Deflate() fiber.Handler {
	return Compression(CompressionConfig{
		Level: 6,
		Types: []string{
			"text/plain",
			"text/html",
			"text/css",
			"text/javascript",
			"application/javascript",
			"application/json",
			"application/xml",
			"application/rss+xml",
			"application/atom+xml",
			"image/svg+xml",
		},
		MinSize: 1024,
	})
}

// APICompression creates a compression middleware optimized for API responses
func APICompression() fiber.Handler {
	return Compression(CompressionConfig{
		Level: 6,
		Types: []string{
			"application/json",
			"application/xml",
			"text/plain",
		},
		MinSize: 512, // Smaller minimum size for API responses
	})
}

// WebCompression creates a compression middleware optimized for web content
func WebCompression() fiber.Handler {
	return Compression(CompressionConfig{
		Level: 6,
		Types: []string{
			"text/html",
			"text/css",
			"text/javascript",
			"application/javascript",
			"application/json",
			"image/svg+xml",
		},
		MinSize: 1024,
	})
}

// isExcludedPath checks if the path should be excluded from compression
func isExcludedPath(path string, excluded []string) bool {
	for _, excludedPath := range excluded {
		if strings.HasPrefix(path, excludedPath) {
			return true
		}
	}
	return false
}

// ConditionalCompression creates a compression middleware with path exclusions
func ConditionalCompression(config CompressionConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if path should be excluded
		if isExcludedPath(c.Path(), config.Excluded) {
			return c.Next()
		}

		// Apply compression
		compression := Compression(config)
		return compression(c)
	}
}
