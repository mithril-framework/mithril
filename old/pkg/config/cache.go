package config

import (
	"os"
	"strconv"
	"time"
)

// CacheConfig holds cache configuration
type CacheConfig struct {
	Driver   string
	Host     string
	Port     int
	Password string
	Database int
	Prefix   string
	TTL      time.Duration
}

// NewCacheConfig creates a new cache configuration from environment variables
func NewCacheConfig() *CacheConfig {
	port, _ := strconv.Atoi(getEnv("CACHE_PORT", "6379"))
	db, _ := strconv.Atoi(getEnv("CACHE_DATABASE", "0"))
	ttl, _ := time.ParseDuration(getEnv("CACHE_TTL", "1h"))

	return &CacheConfig{
		Driver:   getEnv("CACHE_DRIVER", "redis"),
		Host:     getEnv("CACHE_HOST", "localhost"),
		Port:     port,
		Password: getEnv("CACHE_PASSWORD", ""),
		Database: db,
		Prefix:   getEnv("CACHE_PREFIX", "mithril_cache"),
		TTL:      ttl,
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
