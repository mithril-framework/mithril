package config

import (
	"os"
	"strconv"
)

// GRPCConfig holds gRPC configuration
type GRPCConfig struct {
	Enabled         bool
	Host            string
	Port            int
	MaxRecvMsgSize  int
	MaxSendMsgSize  int
	ReflectionEnabled bool
}

// LoadGRPCConfig loads gRPC configuration from environment
func LoadGRPCConfig() *GRPCConfig {
	return &GRPCConfig{
		Enabled:         grpcGetEnvAsBool("GRPC_ENABLED", false),
		Host:            grpcGetEnv("GRPC_HOST", "0.0.0.0"),
		Port:            grpcGetEnvAsInt("GRPC_PORT", 50051),
		MaxRecvMsgSize:  grpcGetEnvAsInt("GRPC_MAX_RECV_MSG_SIZE", 4*1024*1024), // 4MB
		MaxSendMsgSize:  grpcGetEnvAsInt("GRPC_MAX_SEND_MSG_SIZE", 4*1024*1024), // 4MB
		ReflectionEnabled: grpcGetEnvAsBool("GRPC_REFLECTION", true),
	}
}

func grpcGetEnvAsBool(key string, defaultVal bool) bool {
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

func grpcGetEnvAsInt(key string, defaultVal int) int {
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

func grpcGetEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

