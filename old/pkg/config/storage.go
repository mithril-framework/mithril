package config

import (
	"time"
)

// StorageConfig holds storage configuration
type StorageConfig struct {
	// Driver settings
	Driver string `env:"STORAGE_DRIVER" default:"local" required:"true"`

	// Local storage settings
	LocalPath string `env:"STORAGE_LOCAL_PATH" default:"./storage"`

	// S3 settings
	S3Region         string `env:"S3_REGION" default:"us-east-1"`
	S3Bucket         string `env:"S3_BUCKET" default:""`
	S3AccessKey      string `env:"S3_ACCESS_KEY" default:""`
	S3SecretKey      string `env:"S3_SECRET_KEY" default:""`
	S3Endpoint       string `env:"S3_ENDPOINT" default:""`
	S3PathStyle      bool   `env:"S3_PATH_STYLE" default:"false"`
	S3DisableSSL     bool   `env:"S3_DISABLE_SSL" default:"false"`
	S3ForcePathStyle bool   `env:"S3_FORCE_PATH_STYLE" default:"false"`

	// MinIO settings
	MinIOEndpoint  string `env:"MINIO_ENDPOINT" default:"localhost:9000"`
	MinIOAccessKey string `env:"MINIO_ACCESS_KEY" default:"minioadmin"`
	MinIOSecretKey string `env:"MINIO_SECRET_KEY" default:"minioadmin"`
	MinIOBucket    string `env:"MINIO_BUCKET" default:"mithril"`
	MinIOUseSSL    bool   `env:"MINIO_USE_SSL" default:"false"`
	MinIOPathStyle bool   `env:"MINIO_PATH_STYLE" default:"true"`
	MinIORegion    string `env:"MINIO_REGION" default:"us-east-1"`

	// CDN settings
	CDNURL string `env:"CDN_URL" default:""`

	// File settings
	MaxFileSize     int64    `env:"STORAGE_MAX_FILE_SIZE" default:"10485760"` // 10MB
	AllowedTypes    []string `env:"STORAGE_ALLOWED_TYPES" default:"image/jpeg,image/png,image/gif,application/pdf"`
	DisallowedTypes []string `env:"STORAGE_DISALLOWED_TYPES" default:""`

	// Upload settings
	UploadPath  string `env:"STORAGE_UPLOAD_PATH" default:"uploads"`
	TempPath    string `env:"STORAGE_TEMP_PATH" default:"temp"`
	PublicPath  string `env:"STORAGE_PUBLIC_PATH" default:"public"`
	PrivatePath string `env:"STORAGE_PRIVATE_PATH" default:"private"`

	// Backup settings
	BackupEnabled  bool   `env:"STORAGE_BACKUP_ENABLED" default:"true"`
	BackupPath     string `env:"STORAGE_BACKUP_PATH" default:"./storage/backups"`
	BackupSchedule string `env:"STORAGE_BACKUP_SCHEDULE" default:"0 2 * * *"` // Daily at 2 AM

	// Cleanup settings
	CleanupEnabled       bool          `env:"STORAGE_CLEANUP_ENABLED" default:"true"`
	CleanupSchedule      string        `env:"STORAGE_CLEANUP_SCHEDULE" default:"0 3 * * *"`  // Daily at 3 AM
	CleanupOlderThan     time.Duration `env:"STORAGE_CLEANUP_OLDER_THAN" default:"720h"`     // 30 days
	CleanupTempOlderThan time.Duration `env:"STORAGE_CLEANUP_TEMP_OLDER_THAN" default:"24h"` // 1 day

	// Security settings
	VirusScanEnabled bool   `env:"STORAGE_VIRUS_SCAN_ENABLED" default:"false"`
	VirusScanPath    string `env:"STORAGE_VIRUS_SCAN_PATH" default:"/usr/bin/clamscan"`

	// Encryption settings
	EncryptionEnabled bool   `env:"STORAGE_ENCRYPTION_ENABLED" default:"false"`
	EncryptionKey     string `env:"STORAGE_ENCRYPTION_KEY" default:""`

	// Compression settings
	CompressionEnabled bool `env:"STORAGE_COMPRESSION_ENABLED" default:"false"`

	// Caching settings
	CacheEnabled bool          `env:"STORAGE_CACHE_ENABLED" default:"true"`
	CacheTTL     time.Duration `env:"STORAGE_CACHE_TTL" default:"1h"`

	// Monitoring
	MonitoringEnabled bool   `env:"STORAGE_MONITORING_ENABLED" default:"true"`
	MonitoringPath    string `env:"STORAGE_MONITORING_PATH" default:"/monitor/storage"`

	// Metrics
	MetricsEnabled bool   `env:"STORAGE_METRICS_ENABLED" default:"true"`
	MetricsPath    string `env:"STORAGE_METRICS_PATH" default:"/metrics/storage"`

	// Logging
	LogEnabled bool   `env:"STORAGE_LOG_ENABLED" default:"true"`
	LogLevel   string `env:"STORAGE_LOG_LEVEL" default:"info"`

	// Rate limiting
	RateLimitEnabled bool          `env:"STORAGE_RATE_LIMIT_ENABLED" default:"true"`
	RateLimitRPS     int           `env:"STORAGE_RATE_LIMIT_RPS" default:"10"`
	RateLimitBurst   int           `env:"STORAGE_RATE_LIMIT_BURST" default:"20"`
	RateLimitWindow  time.Duration `env:"STORAGE_RATE_LIMIT_WINDOW" default:"1m"`

	// Health check
	HealthCheckEnabled  bool          `env:"STORAGE_HEALTH_CHECK_ENABLED" default:"true"`
	HealthCheckInterval time.Duration `env:"STORAGE_HEALTH_CHECK_INTERVAL" default:"30s"`
}

// GetDriver returns the storage driver
func (c *StorageConfig) GetDriver() string {
	if c.Driver == "" {
		return "local"
	}
	return c.Driver
}

// GetLocalPath returns the local storage path
func (c *StorageConfig) GetLocalPath() string {
	if c.LocalPath == "" {
		return "./storage"
	}
	return c.LocalPath
}

// GetS3Region returns the S3 region
func (c *StorageConfig) GetS3Region() string {
	if c.S3Region == "" {
		return "us-east-1"
	}
	return c.S3Region
}

// GetS3Bucket returns the S3 bucket name
func (c *StorageConfig) GetS3Bucket() string {
	return c.S3Bucket
}

// GetS3AccessKey returns the S3 access key
func (c *StorageConfig) GetS3AccessKey() string {
	return c.S3AccessKey
}

// GetS3SecretKey returns the S3 secret key
func (c *StorageConfig) GetS3SecretKey() string {
	return c.S3SecretKey
}

// GetS3Endpoint returns the S3 endpoint
func (c *StorageConfig) GetS3Endpoint() string {
	return c.S3Endpoint
}

// IsS3PathStyle returns whether S3 path style is enabled
func (c *StorageConfig) IsS3PathStyle() bool {
	return c.S3PathStyle
}

// IsS3DisableSSL returns whether S3 SSL is disabled
func (c *StorageConfig) IsS3DisableSSL() bool {
	return c.S3DisableSSL
}

// IsS3ForcePathStyle returns whether S3 force path style is enabled
func (c *StorageConfig) IsS3ForcePathStyle() bool {
	return c.S3ForcePathStyle
}

// GetMinIOEndpoint returns the MinIO endpoint
func (c *StorageConfig) GetMinIOEndpoint() string {
	if c.MinIOEndpoint == "" {
		return "localhost:9000"
	}
	return c.MinIOEndpoint
}

// GetMinIOAccessKey returns the MinIO access key
func (c *StorageConfig) GetMinIOAccessKey() string {
	if c.MinIOAccessKey == "" {
		return "minioadmin"
	}
	return c.MinIOAccessKey
}

// GetMinIOSecretKey returns the MinIO secret key
func (c *StorageConfig) GetMinIOSecretKey() string {
	if c.MinIOSecretKey == "" {
		return "minioadmin"
	}
	return c.MinIOSecretKey
}

// GetMinIOBucket returns the MinIO bucket name
func (c *StorageConfig) GetMinIOBucket() string {
	if c.MinIOBucket == "" {
		return "mithril"
	}
	return c.MinIOBucket
}

// IsMinIOUseSSL returns whether MinIO uses SSL
func (c *StorageConfig) IsMinIOUseSSL() bool {
	return c.MinIOUseSSL
}

// IsMinIOPathStyle returns whether MinIO path style is enabled
func (c *StorageConfig) IsMinIOPathStyle() bool {
	return c.MinIOPathStyle
}

// GetMinIORegion returns the MinIO region
func (c *StorageConfig) GetMinIORegion() string {
	if c.MinIORegion == "" {
		return "us-east-1"
	}
	return c.MinIORegion
}

// GetCDNURL returns the CDN URL
func (c *StorageConfig) GetCDNURL() string {
	return c.CDNURL
}

// GetMaxFileSize returns the maximum file size in bytes
func (c *StorageConfig) GetMaxFileSize() int64 {
	if c.MaxFileSize <= 0 {
		return 10 * 1024 * 1024 // 10MB
	}
	return c.MaxFileSize
}

// GetMaxFileSizeMB returns the maximum file size in MB
func (c *StorageConfig) GetMaxFileSizeMB() int64 {
	return c.GetMaxFileSize() / 1024 / 1024
}

// GetAllowedTypes returns the allowed file types
func (c *StorageConfig) GetAllowedTypes() []string {
	if len(c.AllowedTypes) == 0 {
		return []string{"image/jpeg", "image/png", "image/gif", "application/pdf"}
	}
	return c.AllowedTypes
}

// GetDisallowedTypes returns the disallowed file types
func (c *StorageConfig) GetDisallowedTypes() []string {
	return c.DisallowedTypes
}

// GetUploadPath returns the upload path
func (c *StorageConfig) GetUploadPath() string {
	if c.UploadPath == "" {
		return "uploads"
	}
	return c.UploadPath
}

// GetTempPath returns the temp path
func (c *StorageConfig) GetTempPath() string {
	if c.TempPath == "" {
		return "temp"
	}
	return c.TempPath
}

// GetPublicPath returns the public path
func (c *StorageConfig) GetPublicPath() string {
	if c.PublicPath == "" {
		return "public"
	}
	return c.PublicPath
}

// GetPrivatePath returns the private path
func (c *StorageConfig) GetPrivatePath() string {
	if c.PrivatePath == "" {
		return "private"
	}
	return c.PrivatePath
}

// IsBackupEnabled returns whether backup is enabled
func (c *StorageConfig) IsBackupEnabled() bool {
	return c.BackupEnabled
}

// GetBackupPath returns the backup path
func (c *StorageConfig) GetBackupPath() string {
	if c.BackupPath == "" {
		return "./storage/backups"
	}
	return c.BackupPath
}

// GetBackupSchedule returns the backup schedule
func (c *StorageConfig) GetBackupSchedule() string {
	if c.BackupSchedule == "" {
		return "0 2 * * *" // Daily at 2 AM
	}
	return c.BackupSchedule
}

// IsCleanupEnabled returns whether cleanup is enabled
func (c *StorageConfig) IsCleanupEnabled() bool {
	return c.CleanupEnabled
}

// GetCleanupSchedule returns the cleanup schedule
func (c *StorageConfig) GetCleanupSchedule() string {
	if c.CleanupSchedule == "" {
		return "0 3 * * *" // Daily at 3 AM
	}
	return c.CleanupSchedule
}

// GetCleanupOlderThan returns the cleanup older than duration
func (c *StorageConfig) GetCleanupOlderThan() time.Duration {
	if c.CleanupOlderThan <= 0 {
		return 30 * 24 * time.Hour // 30 days
	}
	return c.CleanupOlderThan
}

// GetCleanupTempOlderThan returns the cleanup temp older than duration
func (c *StorageConfig) GetCleanupTempOlderThan() time.Duration {
	if c.CleanupTempOlderThan <= 0 {
		return 24 * time.Hour // 1 day
	}
	return c.CleanupTempOlderThan
}

// IsVirusScanEnabled returns whether virus scanning is enabled
func (c *StorageConfig) IsVirusScanEnabled() bool {
	return c.VirusScanEnabled
}

// GetVirusScanPath returns the virus scan path
func (c *StorageConfig) GetVirusScanPath() string {
	if c.VirusScanPath == "" {
		return "/usr/bin/clamscan"
	}
	return c.VirusScanPath
}

// IsEncryptionEnabled returns whether encryption is enabled
func (c *StorageConfig) IsEncryptionEnabled() bool {
	return c.EncryptionEnabled
}

// GetEncryptionKey returns the encryption key
func (c *StorageConfig) GetEncryptionKey() string {
	return c.EncryptionKey
}

// IsCompressionEnabled returns whether compression is enabled
func (c *StorageConfig) IsCompressionEnabled() bool {
	return c.CompressionEnabled
}

// IsCacheEnabled returns whether caching is enabled
func (c *StorageConfig) IsCacheEnabled() bool {
	return c.CacheEnabled
}

// GetCacheTTL returns the cache TTL
func (c *StorageConfig) GetCacheTTL() time.Duration {
	if c.CacheTTL <= 0 {
		return time.Hour
	}
	return c.CacheTTL
}

// IsMonitoringEnabled returns whether monitoring is enabled
func (c *StorageConfig) IsMonitoringEnabled() bool {
	return c.MonitoringEnabled
}

// GetMonitoringPath returns the monitoring path
func (c *StorageConfig) GetMonitoringPath() string {
	if c.MonitoringPath == "" {
		return "/monitor/storage"
	}
	return c.MonitoringPath
}

// IsMetricsEnabled returns whether metrics are enabled
func (c *StorageConfig) IsMetricsEnabled() bool {
	return c.MetricsEnabled
}

// GetMetricsPath returns the metrics path
func (c *StorageConfig) GetMetricsPath() string {
	if c.MetricsPath == "" {
		return "/metrics/storage"
	}
	return c.MetricsPath
}

// IsLogEnabled returns whether logging is enabled
func (c *StorageConfig) IsLogEnabled() bool {
	return c.LogEnabled
}

// GetLogLevel returns the log level
func (c *StorageConfig) GetLogLevel() string {
	if c.LogLevel == "" {
		return "info"
	}
	return c.LogLevel
}

// IsRateLimitEnabled returns whether rate limiting is enabled
func (c *StorageConfig) IsRateLimitEnabled() bool {
	return c.RateLimitEnabled
}

// GetRateLimitRPS returns the rate limit RPS
func (c *StorageConfig) GetRateLimitRPS() int {
	if c.RateLimitRPS <= 0 {
		return 10
	}
	return c.RateLimitRPS
}

// GetRateLimitBurst returns the rate limit burst
func (c *StorageConfig) GetRateLimitBurst() int {
	if c.RateLimitBurst <= 0 {
		return 20
	}
	return c.RateLimitBurst
}

// GetRateLimitWindow returns the rate limit window
func (c *StorageConfig) GetRateLimitWindow() time.Duration {
	if c.RateLimitWindow <= 0 {
		return time.Minute
	}
	return c.RateLimitWindow
}

// IsHealthCheckEnabled returns whether health check is enabled
func (c *StorageConfig) IsHealthCheckEnabled() bool {
	return c.HealthCheckEnabled
}

// GetHealthCheckInterval returns the health check interval
func (c *StorageConfig) GetHealthCheckInterval() time.Duration {
	if c.HealthCheckInterval <= 0 {
		return 30 * time.Second
	}
	return c.HealthCheckInterval
}

// IsLocal returns true if using local driver
func (c *StorageConfig) IsLocal() bool {
	return c.GetDriver() == "local"
}

// IsS3 returns true if using S3 driver
func (c *StorageConfig) IsS3() bool {
	return c.GetDriver() == "s3"
}

// IsMinIO returns true if using MinIO driver
func (c *StorageConfig) IsMinIO() bool {
	return c.GetDriver() == "minio"
}
