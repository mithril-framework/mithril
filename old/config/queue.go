package config

import (
	"os"
	"strconv"
	"time"
)

// QueueConfig holds queue configuration
type QueueConfig struct {
	Driver     string
	Connection string
	Prefix     string

	// Redis configuration
	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int

	// RabbitMQ configuration (for future implementation)
	RabbitMQHost     string
	RabbitMQPort     int
	RabbitMQUser     string
	RabbitMQPassword string
	RabbitMQVHost    string

	// Job defaults
	DefaultTimeout time.Duration
	DefaultRetries int
}

// LoadQueueConfig loads queue configuration from environment
func LoadQueueConfig() *QueueConfig {
	return &QueueConfig{
		Driver:     queueGetEnv("QUEUE_DRIVER", "memory"),
		Connection: queueGetEnv("QUEUE_CONNECTION", "default"),
		Prefix:     queueGetEnv("QUEUE_PREFIX", "mithril:queue"),

		// Redis
		RedisHost:     queueGetEnv("REDIS_HOST", "localhost"),
		RedisPort:     queueGetEnvAsInt("REDIS_PORT", 6379),
		RedisPassword: queueGetEnv("REDIS_PASSWORD", ""),
		RedisDB:       queueGetEnvAsInt("REDIS_DB", 0),

		// RabbitMQ
		RabbitMQHost:     queueGetEnv("RABBITMQ_HOST", "localhost"),
		RabbitMQPort:     queueGetEnvAsInt("RABBITMQ_PORT", 5672),
		RabbitMQUser:     queueGetEnv("RABBITMQ_USER", "guest"),
		RabbitMQPassword: queueGetEnv("RABBITMQ_PASSWORD", "guest"),
		RabbitMQVHost:    queueGetEnv("RABBITMQ_VHOST", "/"),

		// Defaults
		DefaultTimeout: getEnvAsDuration("QUEUE_TIMEOUT", time.Minute*5),
		DefaultRetries: queueGetEnvAsInt("QUEUE_RETRIES", 3),
	}
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	duration, err := time.ParseDuration(val)
	if err != nil {
		return defaultVal
	}
	return duration
}

func queueGetEnvAsInt(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return intVal
}

func queueGetEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}
