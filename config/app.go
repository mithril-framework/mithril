package config

import (
	"os"
	"strconv"
)

// AppConfig holds application configuration
type AppConfig struct {
	Name        string
	Version     string
	Environment string
	Debug       bool
	Port        string
	Host        string
	Timezone    string
	Secret      string
}

// LoadAppConfig loads application configuration from environment variables
func LoadAppConfig() *AppConfig {
	return &AppConfig{
		Name:        getEnv("APP_NAME", "myproject7"),
		Version:     getEnv("APP_VERSION", "1.0.0"),
		Environment: getEnv("APP_ENV", "development"),
		Debug:       getBoolEnv("APP_DEBUG", true),
		Port:        getEnv("PORT", "4000"),
		Host:        getEnv("HOST", "localhost"),
		Timezone:    getEnv("APP_TIMEZONE", "UTC"),
		Secret:      getEnv("APP_SECRET", "your-secret-key-change-in-production"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
