package config

import (
	"time"
)

// QueueConfig holds queue configuration
type QueueConfig struct {
	// Driver settings
	Driver string `env:"QUEUE_DRIVER" default:"memory" required:"true"`

	// Connection settings
	Connection string `env:"QUEUE_CONNECTION" default:"default"`

	// Redis settings
	RedisHost     string `env:"QUEUE_REDIS_HOST" default:"localhost"`
	RedisPort     int    `env:"QUEUE_REDIS_PORT" default:"6379"`
	RedisPassword string `env:"QUEUE_REDIS_PASSWORD" default:""`
	RedisDB       int    `env:"QUEUE_REDIS_DB" default:"1"`

	// RabbitMQ settings
	RabbitMQHost     string `env:"QUEUE_RABBITMQ_HOST" default:"localhost"`
	RabbitMQPort     int    `env:"QUEUE_RABBITMQ_PORT" default:"5672"`
	RabbitMQUsername string `env:"QUEUE_RABBITMQ_USERNAME" default:"guest"`
	RabbitMQPassword string `env:"QUEUE_RABBITMQ_PASSWORD" default:"guest"`
	RabbitMQVHost    string `env:"QUEUE_RABBITMQ_VHOST" default:"/"`

	// Kafka settings
	KafkaBrokers []string `env:"QUEUE_KAFKA_BROKERS" default:"localhost:9092"`

	// Queue settings
	DefaultQueue string `env:"QUEUE_DEFAULT" default:"default"`
	FailedQueue  string `env:"QUEUE_FAILED" default:"failed"`

	// Worker settings
	MaxWorkers    int           `env:"QUEUE_MAX_WORKERS" default:"10"`
	WorkerTimeout time.Duration `env:"QUEUE_WORKER_TIMEOUT" default:"30s"`
	WorkerSleep   time.Duration `env:"QUEUE_WORKER_SLEEP" default:"3s"`

	// Job settings
	MaxRetries    int           `env:"QUEUE_MAX_RETRIES" default:"3"`
	RetryDelay    time.Duration `env:"QUEUE_RETRY_DELAY" default:"5s"`
	RetryBackoff  bool          `env:"QUEUE_RETRY_BACKOFF" default:"true"`
	RetryMaxDelay time.Duration `env:"QUEUE_RETRY_MAX_DELAY" default:"1h"`

	// Job timeout
	JobTimeout time.Duration `env:"QUEUE_JOB_TIMEOUT" default:"5m"`

	// Dead letter queue
	DeadLetterEnabled bool   `env:"QUEUE_DEAD_LETTER_ENABLED" default:"true"`
	DeadLetterQueue   string `env:"QUEUE_DEAD_LETTER_QUEUE" default:"dead_letter"`

	// Monitoring
	MonitoringEnabled bool   `env:"QUEUE_MONITORING_ENABLED" default:"true"`
	MonitoringPath    string `env:"QUEUE_MONITORING_PATH" default:"/monitor/queue"`

	// Metrics
	MetricsEnabled bool   `env:"QUEUE_METRICS_ENABLED" default:"true"`
	MetricsPath    string `env:"QUEUE_METRICS_PATH" default:"/metrics/queue"`

	// Logging
	LogEnabled bool   `env:"QUEUE_LOG_ENABLED" default:"true"`
	LogLevel   string `env:"QUEUE_LOG_LEVEL" default:"info"`

	// Batch processing
	BatchSize    int           `env:"QUEUE_BATCH_SIZE" default:"100"`
	BatchTimeout time.Duration `env:"QUEUE_BATCH_TIMEOUT" default:"1s"`

	// Priority queues
	PriorityEnabled bool `env:"QUEUE_PRIORITY_ENABLED" default:"false"`
	MaxPriority     int  `env:"QUEUE_MAX_PRIORITY" default:"10"`

	// Rate limiting
	RateLimitEnabled bool          `env:"QUEUE_RATE_LIMIT_ENABLED" default:"false"`
	RateLimitRPS     int           `env:"QUEUE_RATE_LIMIT_RPS" default:"100"`
	RateLimitBurst   int           `env:"QUEUE_RATE_LIMIT_BURST" default:"200"`
	RateLimitWindow  time.Duration `env:"QUEUE_RATE_LIMIT_WINDOW" default:"1m"`

	// Health check
	HealthCheckEnabled  bool          `env:"QUEUE_HEALTH_CHECK_ENABLED" default:"true"`
	HealthCheckInterval time.Duration `env:"QUEUE_HEALTH_CHECK_INTERVAL" default:"30s"`
}

// GetDriver returns the queue driver
func (c *QueueConfig) GetDriver() string {
	if c.Driver == "" {
		return "memory"
	}
	return c.Driver
}

// GetConnection returns the queue connection
func (c *QueueConfig) GetConnection() string {
	if c.Connection == "" {
		return "default"
	}
	return c.Connection
}

// GetRedisHost returns the Redis host
func (c *QueueConfig) GetRedisHost() string {
	if c.RedisHost == "" {
		return "localhost"
	}
	return c.RedisHost
}

// GetRedisPort returns the Redis port
func (c *QueueConfig) GetRedisPort() int {
	if c.RedisPort <= 0 {
		return 6379
	}
	return c.RedisPort
}

// GetRedisPassword returns the Redis password
func (c *QueueConfig) GetRedisPassword() string {
	return c.RedisPassword
}

// GetRedisDB returns the Redis database number
func (c *QueueConfig) GetRedisDB() int {
	return c.RedisDB
}

// GetRedisAddr returns the Redis address
func (c *QueueConfig) GetRedisAddr() string {
	return c.GetRedisHost() + ":" + string(rune(c.GetRedisPort()))
}

// GetRabbitMQHost returns the RabbitMQ host
func (c *QueueConfig) GetRabbitMQHost() string {
	if c.RabbitMQHost == "" {
		return "localhost"
	}
	return c.RabbitMQHost
}

// GetRabbitMQPort returns the RabbitMQ port
func (c *QueueConfig) GetRabbitMQPort() int {
	if c.RabbitMQPort <= 0 {
		return 5672
	}
	return c.RabbitMQPort
}

// GetRabbitMQUsername returns the RabbitMQ username
func (c *QueueConfig) GetRabbitMQUsername() string {
	if c.RabbitMQUsername == "" {
		return "guest"
	}
	return c.RabbitMQUsername
}

// GetRabbitMQPassword returns the RabbitMQ password
func (c *QueueConfig) GetRabbitMQPassword() string {
	if c.RabbitMQPassword == "" {
		return "guest"
	}
	return c.RabbitMQPassword
}

// GetRabbitMQVHost returns the RabbitMQ virtual host
func (c *QueueConfig) GetRabbitMQVHost() string {
	if c.RabbitMQVHost == "" {
		return "/"
	}
	return c.RabbitMQVHost
}

// GetRabbitMQURL returns the RabbitMQ connection URL
func (c *QueueConfig) GetRabbitMQURL() string {
	return "amqp://" + c.GetRabbitMQUsername() + ":" + c.GetRabbitMQPassword() + "@" + c.GetRabbitMQHost() + ":" + string(rune(c.GetRabbitMQPort())) + c.GetRabbitMQVHost()
}

// GetKafkaBrokers returns the Kafka brokers
func (c *QueueConfig) GetKafkaBrokers() []string {
	if len(c.KafkaBrokers) == 0 {
		return []string{"localhost:9092"}
	}
	return c.KafkaBrokers
}

// GetDefaultQueue returns the default queue name
func (c *QueueConfig) GetDefaultQueue() string {
	if c.DefaultQueue == "" {
		return "default"
	}
	return c.DefaultQueue
}

// GetFailedQueue returns the failed queue name
func (c *QueueConfig) GetFailedQueue() string {
	if c.FailedQueue == "" {
		return "failed"
	}
	return c.FailedQueue
}

// GetMaxWorkers returns the maximum number of workers
func (c *QueueConfig) GetMaxWorkers() int {
	if c.MaxWorkers <= 0 {
		return 10
	}
	return c.MaxWorkers
}

// GetWorkerTimeout returns the worker timeout
func (c *QueueConfig) GetWorkerTimeout() time.Duration {
	if c.WorkerTimeout <= 0 {
		return 30 * time.Second
	}
	return c.WorkerTimeout
}

// GetWorkerSleep returns the worker sleep duration
func (c *QueueConfig) GetWorkerSleep() time.Duration {
	if c.WorkerSleep <= 0 {
		return 3 * time.Second
	}
	return c.WorkerSleep
}

// GetMaxRetries returns the maximum number of retries
func (c *QueueConfig) GetMaxRetries() int {
	if c.MaxRetries <= 0 {
		return 3
	}
	return c.MaxRetries
}

// GetRetryDelay returns the retry delay
func (c *QueueConfig) GetRetryDelay() time.Duration {
	if c.RetryDelay <= 0 {
		return 5 * time.Second
	}
	return c.RetryDelay
}

// IsRetryBackoff returns whether retry backoff is enabled
func (c *QueueConfig) IsRetryBackoff() bool {
	return c.RetryBackoff
}

// GetRetryMaxDelay returns the maximum retry delay
func (c *QueueConfig) GetRetryMaxDelay() time.Duration {
	if c.RetryMaxDelay <= 0 {
		return time.Hour
	}
	return c.RetryMaxDelay
}

// GetJobTimeout returns the job timeout
func (c *QueueConfig) GetJobTimeout() time.Duration {
	if c.JobTimeout <= 0 {
		return 5 * time.Minute
	}
	return c.JobTimeout
}

// IsDeadLetterEnabled returns whether dead letter queue is enabled
func (c *QueueConfig) IsDeadLetterEnabled() bool {
	return c.DeadLetterEnabled
}

// GetDeadLetterQueue returns the dead letter queue name
func (c *QueueConfig) GetDeadLetterQueue() string {
	if c.DeadLetterQueue == "" {
		return "dead_letter"
	}
	return c.DeadLetterQueue
}

// IsMonitoringEnabled returns whether monitoring is enabled
func (c *QueueConfig) IsMonitoringEnabled() bool {
	return c.MonitoringEnabled
}

// GetMonitoringPath returns the monitoring path
func (c *QueueConfig) GetMonitoringPath() string {
	if c.MonitoringPath == "" {
		return "/monitor/queue"
	}
	return c.MonitoringPath
}

// IsMetricsEnabled returns whether metrics are enabled
func (c *QueueConfig) IsMetricsEnabled() bool {
	return c.MetricsEnabled
}

// GetMetricsPath returns the metrics path
func (c *QueueConfig) GetMetricsPath() string {
	if c.MetricsPath == "" {
		return "/metrics/queue"
	}
	return c.MetricsPath
}

// IsLogEnabled returns whether logging is enabled
func (c *QueueConfig) IsLogEnabled() bool {
	return c.LogEnabled
}

// GetLogLevel returns the log level
func (c *QueueConfig) GetLogLevel() string {
	if c.LogLevel == "" {
		return "info"
	}
	return c.LogLevel
}

// GetBatchSize returns the batch size
func (c *QueueConfig) GetBatchSize() int {
	if c.BatchSize <= 0 {
		return 100
	}
	return c.BatchSize
}

// GetBatchTimeout returns the batch timeout
func (c *QueueConfig) GetBatchTimeout() time.Duration {
	if c.BatchTimeout <= 0 {
		return time.Second
	}
	return c.BatchTimeout
}

// IsPriorityEnabled returns whether priority queues are enabled
func (c *QueueConfig) IsPriorityEnabled() bool {
	return c.PriorityEnabled
}

// GetMaxPriority returns the maximum priority
func (c *QueueConfig) GetMaxPriority() int {
	if c.MaxPriority <= 0 {
		return 10
	}
	return c.MaxPriority
}

// IsRateLimitEnabled returns whether rate limiting is enabled
func (c *QueueConfig) IsRateLimitEnabled() bool {
	return c.RateLimitEnabled
}

// GetRateLimitRPS returns the rate limit RPS
func (c *QueueConfig) GetRateLimitRPS() int {
	if c.RateLimitRPS <= 0 {
		return 100
	}
	return c.RateLimitRPS
}

// GetRateLimitBurst returns the rate limit burst
func (c *QueueConfig) GetRateLimitBurst() int {
	if c.RateLimitBurst <= 0 {
		return 200
	}
	return c.RateLimitBurst
}

// GetRateLimitWindow returns the rate limit window
func (c *QueueConfig) GetRateLimitWindow() time.Duration {
	if c.RateLimitWindow <= 0 {
		return time.Minute
	}
	return c.RateLimitWindow
}

// IsHealthCheckEnabled returns whether health check is enabled
func (c *QueueConfig) IsHealthCheckEnabled() bool {
	return c.HealthCheckEnabled
}

// GetHealthCheckInterval returns the health check interval
func (c *QueueConfig) GetHealthCheckInterval() time.Duration {
	if c.HealthCheckInterval <= 0 {
		return 30 * time.Second
	}
	return c.HealthCheckInterval
}

// IsMemory returns true if using memory driver
func (c *QueueConfig) IsMemory() bool {
	return c.GetDriver() == "memory"
}

// IsRedis returns true if using Redis driver
func (c *QueueConfig) IsRedis() bool {
	return c.GetDriver() == "redis"
}

// IsRabbitMQ returns true if using RabbitMQ driver
func (c *QueueConfig) IsRabbitMQ() bool {
	return c.GetDriver() == "rabbitmq"
}

// IsKafka returns true if using Kafka driver
func (c *QueueConfig) IsKafka() bool {
	return c.GetDriver() == "kafka"
}
