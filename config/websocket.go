package config

import (
	"os"
	"strconv"
)

// WebSocketConfig holds WebSocket configuration
type WebSocketConfig struct {
	Enabled         bool
	Port            int
	Path            string
	ReadBufferSize  int
	WriteBufferSize int
	MaxMessageSize  int64
	PingInterval    int // seconds
	PongTimeout     int // seconds
}

// LoadWebSocketConfig loads WebSocket configuration from environment
func LoadWebSocketConfig() *WebSocketConfig {
	return &WebSocketConfig{
		Enabled:         wsGetEnvAsBool("WEBSOCKET_ENABLED", true),
		Port:            wsGetEnvAsInt("WEBSOCKET_PORT", 8080),
		Path:            wsGetEnv("WEBSOCKET_PATH", "/ws"),
		ReadBufferSize:  wsGetEnvAsInt("WEBSOCKET_READ_BUFFER", 1024),
		WriteBufferSize: wsGetEnvAsInt("WEBSOCKET_WRITE_BUFFER", 1024),
		MaxMessageSize:  int64(wsGetEnvAsInt("WEBSOCKET_MAX_MESSAGE_SIZE", 512*1024)), // 512KB default
		PingInterval:    wsGetEnvAsInt("WEBSOCKET_PING_INTERVAL", 54),                 // seconds
		PongTimeout:     wsGetEnvAsInt("WEBSOCKET_PONG_TIMEOUT", 60),                  // seconds
	}
}

func wsGetEnvAsBool(key string, defaultVal bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	boolVal, err := strconv.ParseBool(val)
	if err != nil {
		return defaultVal
	}
	return boolVal
}

func wsGetEnvAsInt(key string, defaultVal int) int {
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

func wsGetEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

