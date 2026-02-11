package config

import (
	"os"
	"strings"
)

type StorageBackend string

const (
	StorageS3    StorageBackend = "s3"
	StorageMinIO StorageBackend = "minio"
)

type StorageConfig struct {
	Backend StorageBackend

	// S3
	S3Region         string
	S3Bucket         string
	S3Endpoint       string
	S3ForcePathStyle bool

	// MinIO
	MinIOEndpoint string
	MinIOAccess   string
	MinIOSecret   string
	MinIOSecure   bool
	MinIOBucket   string
	MinIORegion   string
}

func LoadStorage() *StorageConfig {
	b := strings.ToLower(getEnv("STORAGE_BACKEND", ""))
	cfg := &StorageConfig{}
	switch b {
	case "s3":
		cfg.Backend = StorageS3
		cfg.S3Region = getEnv("S3_REGION", "us-east-1")
		cfg.S3Bucket = getEnv("S3_BUCKET", "")
		cfg.S3Endpoint = getEnv("S3_ENDPOINT", "")
		cfg.S3ForcePathStyle = strings.ToLower(getEnv("S3_FORCE_PATH_STYLE", "false")) == "true"
	case "minio":
		cfg.Backend = StorageMinIO
		cfg.MinIOEndpoint = getEnv("MINIO_ENDPOINT", "")
		cfg.MinIOAccess = getEnv("MINIO_ACCESS_KEY", "")
		cfg.MinIOSecret = getEnv("MINIO_SECRET_KEY", "")
		cfg.MinIOSecure = strings.ToLower(getEnv("MINIO_SECURE", "false")) == "true"
		cfg.MinIOBucket = getEnv("MINIO_BUCKET", "")
		cfg.MinIORegion = getEnv("MINIO_REGION", "us-east-1")
	default:
		// leave Backend empty; user must set envs
	}
	return cfg
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
