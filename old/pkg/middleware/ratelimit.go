package middleware

import (
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// RateLimiterConfig holds configuration for rate limiting
type RateLimiterConfig struct {
	Max        int                     // Maximum number of requests
	Expiration time.Duration           // Time window for rate limiting
	KeyFunc    func(*fiber.Ctx) string // Function to generate rate limit key
	Message    string                  // Error message when rate limit exceeded
	StatusCode int                     // HTTP status code when rate limit exceeded
}

// DefaultRateLimiterConfig returns default configuration
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		Max:        100,
		Expiration: 1 * time.Minute,
		KeyFunc: func(c *fiber.Ctx) string {
			return c.IP()
		},
		Message:    "Too Many Requests",
		StatusCode: 429,
	}
}

// RateLimiter implements rate limiting using sliding window
type RateLimiter struct {
	mu       sync.RWMutex
	requests map[string][]time.Time
	config   RateLimiterConfig
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		config:   config,
	}
}

// Allow checks if a request is allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.config.Expiration)

	// Clean up old requests
	if requests, exists := rl.requests[key]; exists {
		var validRequests []time.Time
		for _, reqTime := range requests {
			if reqTime.After(cutoff) {
				validRequests = append(validRequests, reqTime)
			}
		}
		rl.requests[key] = validRequests
	}

	// Check if we're under the limit
	requestCount := len(rl.requests[key])
	if requestCount >= rl.config.Max {
		return false
	}

	// Add current request
	rl.requests[key] = append(rl.requests[key], now)
	return true
}

// RateLimit creates a rate limiting middleware
func RateLimit(config ...RateLimiterConfig) fiber.Handler {
	cfg := DefaultRateLimiterConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	limiter := NewRateLimiter(cfg)

	return func(c *fiber.Ctx) error {
		key := cfg.KeyFunc(c)

		if !limiter.Allow(key) {
			return c.Status(cfg.StatusCode).JSON(fiber.Map{
				"error":   "Rate limit exceeded",
				"message": cfg.Message,
			})
		}

		return c.Next()
	}
}

// RateLimitByIP creates a rate limiter that limits by IP address
func RateLimitByIP(max int, expiration time.Duration) fiber.Handler {
	return RateLimit(RateLimiterConfig{
		Max:        max,
		Expiration: expiration,
		KeyFunc: func(c *fiber.Ctx) string {
			return c.IP()
		},
	})
}

// RateLimitByUser creates a rate limiter that limits by user ID
func RateLimitByUser(max int, expiration time.Duration) fiber.Handler {
	return RateLimit(RateLimiterConfig{
		Max:        max,
		Expiration: expiration,
		KeyFunc: func(c *fiber.Ctx) string {
			// Try to get user ID from context or JWT token
			if userID := c.Locals("user_id"); userID != nil {
				return fmt.Sprintf("user:%v", userID)
			}
			// Fallback to IP if no user ID
			return c.IP()
		},
	})
}

// RateLimitByRoute creates a rate limiter that limits by route
func RateLimitByRoute(max int, expiration time.Duration) fiber.Handler {
	return RateLimit(RateLimiterConfig{
		Max:        max,
		Expiration: expiration,
		KeyFunc: func(c *fiber.Ctx) string {
			return fmt.Sprintf("route:%s:%s", c.Method(), c.Path())
		},
	})
}

// RedisRateLimiter implements rate limiting using Redis (for distributed systems)
type RedisRateLimiter struct {
	// This would be implemented with Redis client
	// For now, we'll use the in-memory implementation
	*RateLimiter
}

// NewRedisRateLimiter creates a new Redis-based rate limiter
func NewRedisRateLimiter(config RateLimiterConfig) *RedisRateLimiter {
	return &RedisRateLimiter{
		RateLimiter: NewRateLimiter(config),
	}
}

// RateLimitConfig holds configuration for different rate limit types
type RateLimitConfig struct {
	Global   RateLimiterConfig // Global rate limit
	PerIP    RateLimiterConfig // Per IP rate limit
	PerUser  RateLimiterConfig // Per user rate limit
	PerRoute RateLimiterConfig // Per route rate limit
}

// DefaultRateLimitConfig returns default rate limit configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Global: RateLimiterConfig{
			Max:        1000,
			Expiration: 1 * time.Minute,
			KeyFunc: func(c *fiber.Ctx) string {
				return "global"
			},
		},
		PerIP: RateLimiterConfig{
			Max:        100,
			Expiration: 1 * time.Minute,
			KeyFunc: func(c *fiber.Ctx) string {
				return c.IP()
			},
		},
		PerUser: RateLimiterConfig{
			Max:        200,
			Expiration: 1 * time.Minute,
			KeyFunc: func(c *fiber.Ctx) string {
				if userID := c.Locals("user_id"); userID != nil {
					return fmt.Sprintf("user:%v", userID)
				}
				return c.IP()
			},
		},
		PerRoute: RateLimiterConfig{
			Max:        50,
			Expiration: 1 * time.Minute,
			KeyFunc: func(c *fiber.Ctx) string {
				return fmt.Sprintf("route:%s:%s", c.Method(), c.Path())
			},
		},
	}
}

// ApplyRateLimits applies multiple rate limits
func ApplyRateLimits(config RateLimitConfig) []fiber.Handler {
	return []fiber.Handler{
		RateLimit(config.Global),
		RateLimit(config.PerIP),
		RateLimit(config.PerUser),
		RateLimit(config.PerRoute),
	}
}
