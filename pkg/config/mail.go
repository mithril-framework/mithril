package config

import (
	"time"
)

// MailConfig holds mail configuration
type MailConfig struct {
	// Driver settings
	Driver string `env:"MAIL_DRIVER" default:"smtp" required:"true"`

	// SMTP settings
	SMTPHost       string `env:"MAIL_HOST" default:"localhost"`
	SMTPPort       int    `env:"MAIL_PORT" default:"587"`
	SMTPUsername   string `env:"MAIL_USERNAME" default:""`
	SMTPPassword   string `env:"MAIL_PASSWORD" default:""`
	SMTPEncryption string `env:"MAIL_ENCRYPTION" default:"tls"`

	// From address
	FromAddress string `env:"MAIL_FROM_ADDRESS" default:"noreply@example.com" required:"true"`
	FromName    string `env:"MAIL_FROM_NAME" default:"Mithril App"`

	// SendGrid settings
	SendGridAPIKey string `env:"SENDGRID_API_KEY" default:""`

	// Mailgun settings
	MailgunDomain string `env:"MAILGUN_DOMAIN" default:""`
	MailgunAPIKey string `env:"MAILGUN_API_KEY" default:""`

	// Queue settings
	QueueEnabled bool          `env:"MAIL_QUEUE_ENABLED" default:"true"`
	QueueName    string        `env:"MAIL_QUEUE_NAME" default:"emails"`
	QueueTimeout time.Duration `env:"MAIL_QUEUE_TIMEOUT" default:"30s"`

	// Retry settings
	MaxRetries int           `env:"MAIL_MAX_RETRIES" default:"3"`
	RetryDelay time.Duration `env:"MAIL_RETRY_DELAY" default:"5s"`

	// Rate limiting
	RateLimitEnabled bool          `env:"MAIL_RATE_LIMIT_ENABLED" default:"true"`
	RateLimitRPS     int           `env:"MAIL_RATE_LIMIT_RPS" default:"10"`
	RateLimitBurst   int           `env:"MAIL_RATE_LIMIT_BURST" default:"20"`
	RateLimitWindow  time.Duration `env:"MAIL_RATE_LIMIT_WINDOW" default:"1m"`

	// Template settings
	TemplatePath string `env:"MAIL_TEMPLATE_PATH" default:"./templates/emails"`

	// Testing settings
	TestingEnabled bool   `env:"MAIL_TESTING_ENABLED" default:"false"`
	TestingEmail   string `env:"MAIL_TESTING_EMAIL" default:""`

	// Logging
	LogEnabled bool   `env:"MAIL_LOG_ENABLED" default:"true"`
	LogLevel   string `env:"MAIL_LOG_LEVEL" default:"info"`

	// Timeout settings
	ConnectTimeout time.Duration `env:"MAIL_CONNECT_TIMEOUT" default:"10s"`
	SendTimeout    time.Duration `env:"MAIL_SEND_TIMEOUT" default:"30s"`

	// Authentication
	AuthEnabled bool   `env:"MAIL_AUTH_ENABLED" default:"true"`
	AuthMethod  string `env:"MAIL_AUTH_METHOD" default:"PLAIN"`

	// TLS settings
	TLSEnabled            bool `env:"MAIL_TLS_ENABLED" default:"true"`
	TLSInsecureSkipVerify bool `env:"MAIL_TLS_INSECURE_SKIP_VERIFY" default:"false"`

	// Keep alive
	KeepAlive        bool          `env:"MAIL_KEEP_ALIVE" default:"true"`
	KeepAliveTimeout time.Duration `env:"MAIL_KEEP_ALIVE_TIMEOUT" default:"30s"`
}

// GetDriver returns the mail driver
func (c *MailConfig) GetDriver() string {
	if c.Driver == "" {
		return "smtp"
	}
	return c.Driver
}

// GetSMTPHost returns the SMTP host
func (c *MailConfig) GetSMTPHost() string {
	if c.SMTPHost == "" {
		return "localhost"
	}
	return c.SMTPHost
}

// GetSMTPPort returns the SMTP port
func (c *MailConfig) GetSMTPPort() int {
	if c.SMTPPort <= 0 {
		return 587
	}
	return c.SMTPPort
}

// GetSMTPUsername returns the SMTP username
func (c *MailConfig) GetSMTPUsername() string {
	return c.SMTPUsername
}

// GetSMTPPassword returns the SMTP password
func (c *MailConfig) GetSMTPPassword() string {
	return c.SMTPPassword
}

// GetSMTPEncryption returns the SMTP encryption method
func (c *MailConfig) GetSMTPEncryption() string {
	if c.SMTPEncryption == "" {
		return "tls"
	}
	return c.SMTPEncryption
}

// GetFromAddress returns the from address
func (c *MailConfig) GetFromAddress() string {
	if c.FromAddress == "" {
		return "noreply@example.com"
	}
	return c.FromAddress
}

// GetFromName returns the from name
func (c *MailConfig) GetFromName() string {
	if c.FromName == "" {
		return "Mithril App"
	}
	return c.FromName
}

// GetSendGridAPIKey returns the SendGrid API key
func (c *MailConfig) GetSendGridAPIKey() string {
	return c.SendGridAPIKey
}

// GetMailgunDomain returns the Mailgun domain
func (c *MailConfig) GetMailgunDomain() string {
	return c.MailgunDomain
}

// GetMailgunAPIKey returns the Mailgun API key
func (c *MailConfig) GetMailgunAPIKey() string {
	return c.MailgunAPIKey
}

// IsQueueEnabled returns whether queuing is enabled
func (c *MailConfig) IsQueueEnabled() bool {
	return c.QueueEnabled
}

// GetQueueName returns the queue name
func (c *MailConfig) GetQueueName() string {
	if c.QueueName == "" {
		return "emails"
	}
	return c.QueueName
}

// GetQueueTimeout returns the queue timeout
func (c *MailConfig) GetQueueTimeout() time.Duration {
	if c.QueueTimeout <= 0 {
		return 30 * time.Second
	}
	return c.QueueTimeout
}

// GetMaxRetries returns the maximum number of retries
func (c *MailConfig) GetMaxRetries() int {
	if c.MaxRetries <= 0 {
		return 3
	}
	return c.MaxRetries
}

// GetRetryDelay returns the retry delay
func (c *MailConfig) GetRetryDelay() time.Duration {
	if c.RetryDelay <= 0 {
		return 5 * time.Second
	}
	return c.RetryDelay
}

// IsRateLimitEnabled returns whether rate limiting is enabled
func (c *MailConfig) IsRateLimitEnabled() bool {
	return c.RateLimitEnabled
}

// GetRateLimitRPS returns the rate limit RPS
func (c *MailConfig) GetRateLimitRPS() int {
	if c.RateLimitRPS <= 0 {
		return 10
	}
	return c.RateLimitRPS
}

// GetRateLimitBurst returns the rate limit burst
func (c *MailConfig) GetRateLimitBurst() int {
	if c.RateLimitBurst <= 0 {
		return 20
	}
	return c.RateLimitBurst
}

// GetRateLimitWindow returns the rate limit window
func (c *MailConfig) GetRateLimitWindow() time.Duration {
	if c.RateLimitWindow <= 0 {
		return time.Minute
	}
	return c.RateLimitWindow
}

// GetTemplatePath returns the template path
func (c *MailConfig) GetTemplatePath() string {
	if c.TemplatePath == "" {
		return "./templates/emails"
	}
	return c.TemplatePath
}

// IsTestingEnabled returns whether testing mode is enabled
func (c *MailConfig) IsTestingEnabled() bool {
	return c.TestingEnabled
}

// GetTestingEmail returns the testing email
func (c *MailConfig) GetTestingEmail() string {
	return c.TestingEmail
}

// IsLogEnabled returns whether logging is enabled
func (c *MailConfig) IsLogEnabled() bool {
	return c.LogEnabled
}

// GetLogLevel returns the log level
func (c *MailConfig) GetLogLevel() string {
	if c.LogLevel == "" {
		return "info"
	}
	return c.LogLevel
}

// GetConnectTimeout returns the connection timeout
func (c *MailConfig) GetConnectTimeout() time.Duration {
	if c.ConnectTimeout <= 0 {
		return 10 * time.Second
	}
	return c.ConnectTimeout
}

// GetSendTimeout returns the send timeout
func (c *MailConfig) GetSendTimeout() time.Duration {
	if c.SendTimeout <= 0 {
		return 30 * time.Second
	}
	return c.SendTimeout
}

// IsAuthEnabled returns whether authentication is enabled
func (c *MailConfig) IsAuthEnabled() bool {
	return c.AuthEnabled
}

// GetAuthMethod returns the authentication method
func (c *MailConfig) GetAuthMethod() string {
	if c.AuthMethod == "" {
		return "PLAIN"
	}
	return c.AuthMethod
}

// IsTLSEnabled returns whether TLS is enabled
func (c *MailConfig) IsTLSEnabled() bool {
	return c.TLSEnabled
}

// IsTLSInsecureSkipVerify returns whether to skip TLS verification
func (c *MailConfig) IsTLSInsecureSkipVerify() bool {
	return c.TLSInsecureSkipVerify
}

// IsKeepAlive returns whether to keep connections alive
func (c *MailConfig) IsKeepAlive() bool {
	return c.KeepAlive
}

// GetKeepAliveTimeout returns the keep alive timeout
func (c *MailConfig) GetKeepAliveTimeout() time.Duration {
	if c.KeepAliveTimeout <= 0 {
		return 30 * time.Second
	}
	return c.KeepAliveTimeout
}

// IsSMTP returns true if using SMTP driver
func (c *MailConfig) IsSMTP() bool {
	return c.GetDriver() == "smtp"
}

// IsSendGrid returns true if using SendGrid driver
func (c *MailConfig) IsSendGrid() bool {
	return c.GetDriver() == "sendgrid"
}

// IsMailgun returns true if using Mailgun driver
func (c *MailConfig) IsMailgun() bool {
	return c.GetDriver() == "mailgun"
}
