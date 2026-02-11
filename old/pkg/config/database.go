package config

import (
	"strconv"
	"time"
)

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	// Primary database connection
	Driver   string `env:"DB_DRIVER" default:"postgres" required:"true"`
	Host     string `env:"DB_HOST" default:"localhost" required:"true"`
	Port     int    `env:"DB_PORT" default:"5432" required:"true"`
	Username string `env:"DB_USERNAME" default:"mithril" required:"true"`
	Password string `env:"DB_PASSWORD" required:"true"`
	Database string `env:"DB_NAME" default:"mithril" required:"true"`

	// Connection pool settings
	MaxOpenConns    int           `env:"DB_MAX_OPEN_CONNS" default:"25"`
	MaxIdleConns    int           `env:"DB_MAX_IDLE_CONNS" default:"5"`
	ConnMaxLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME" default:"5m"`
	ConnMaxIdleTime time.Duration `env:"DB_CONN_MAX_IDLE_TIME" default:"1m"`

	// SSL settings
	SSLMode string `env:"DB_SSL_MODE" default:"disable"`

	// Seeder settings
	SeederPath string `env:"DB_SEEDER_PATH" default:"./database/seeders"`

	// Backup settings
	BackupPath string `env:"DB_BACKUP_PATH" default:"./database/backups"`

	// Additional database connections
	RedisHost     string `env:"REDIS_HOST" default:"localhost"`
	RedisPort     int    `env:"REDIS_PORT" default:"6379"`
	RedisPassword string `env:"REDIS_PASSWORD" default:""`
	RedisDB       int    `env:"REDIS_DB" default:"0"`

	// MongoDB settings (if used)
	MongoURI      string `env:"MONGO_URI" default:""`
	MongoDatabase string `env:"MONGO_DATABASE" default:"mithril"`

	// MySQL settings (if used)
	MySQLCharset   string `env:"MYSQL_CHARSET" default:"utf8mb4"`
	MySQLParseTime bool   `env:"MYSQL_PARSE_TIME" default:"true"`
	MySQLLoc       string `env:"MYSQL_LOC" default:"Local"`

	// SQLite settings (if used)
	SQLitePath string `env:"SQLITE_PATH" default:"./database/sqlite.db"`
}

// GetDSN returns the database connection string
func (c *DatabaseConfig) GetDSN() string {
	switch c.Driver {
	case "postgres":
		return c.getPostgresDSN()
	case "mysql":
		return c.getMySQLDSN()
	case "sqlite":
		return c.getSQLiteDSN()
	case "mongodb":
		return c.getMongoDSN()
	default:
		return c.getPostgresDSN()
	}
}

// getPostgresDSN returns PostgreSQL connection string
func (c *DatabaseConfig) getPostgresDSN() string {
	dsn := "host=" + c.Host +
		" port=" + string(rune(c.Port)) +
		" user=" + c.Username +
		" password=" + c.Password +
		" dbname=" + c.Database +
		" sslmode=" + c.SSLMode

	return dsn
}

// getMySQLDSN returns MySQL connection string
func (c *DatabaseConfig) getMySQLDSN() string {
	dsn := c.Username + ":" + c.Password +
		"@tcp(" + c.Host + ":" + string(rune(c.Port)) + ")/" + c.Database +
		"?charset=" + c.MySQLCharset +
		"&parseTime=" + strconv.FormatBool(c.MySQLParseTime) +
		"&loc=" + c.MySQLLoc

	return dsn
}

// getSQLiteDSN returns SQLite connection string
func (c *DatabaseConfig) getSQLiteDSN() string {
	return c.SQLitePath
}

// getMongoDSN returns MongoDB connection string
func (c *DatabaseConfig) getMongoDSN() string {
	if c.MongoURI != "" {
		return c.MongoURI
	}
	return "mongodb://" + c.Host + ":" + string(rune(c.Port)) + "/" + c.MongoDatabase
}

// GetRedisAddr returns Redis address
func (c *DatabaseConfig) GetRedisAddr() string {
	return c.RedisHost + ":" + string(rune(c.RedisPort))
}

// GetRedisPassword returns Redis password
func (c *DatabaseConfig) GetRedisPassword() string {
	return c.RedisPassword
}

// GetRedisDB returns Redis database number
func (c *DatabaseConfig) GetRedisDB() int {
	return c.RedisDB
}

// IsPostgres returns true if using PostgreSQL
func (c *DatabaseConfig) IsPostgres() bool {
	return c.Driver == "postgres"
}

// IsMySQL returns true if using MySQL
func (c *DatabaseConfig) IsMySQL() bool {
	return c.Driver == "mysql"
}

// IsSQLite returns true if using SQLite
func (c *DatabaseConfig) IsSQLite() bool {
	return c.Driver == "sqlite"
}

// IsMongoDB returns true if using MongoDB
func (c *DatabaseConfig) IsMongoDB() bool {
	return c.Driver == "mongodb"
}

// GetSeederPath returns the seeder path
func (c *DatabaseConfig) GetSeederPath() string {
	if c.SeederPath == "" {
		return "./database/seeders"
	}
	return c.SeederPath
}

// GetBackupPath returns the backup path
func (c *DatabaseConfig) GetBackupPath() string {
	if c.BackupPath == "" {
		return "./database/backups"
	}
	return c.BackupPath
}

// NewDatabaseConfig creates a new DatabaseConfig from environment variables
func NewDatabaseConfig() *DatabaseConfig {
	port, _ := strconv.Atoi(getEnvOrDefault("DB_PORT", "5432"))
	maxOpenConns, _ := strconv.Atoi(getEnvOrDefault("DB_MAX_OPEN_CONNS", "100"))
	maxIdleConns, _ := strconv.Atoi(getEnvOrDefault("DB_MAX_IDLE_CONNS", "10"))

	return &DatabaseConfig{
		Driver:       getEnvOrDefault("DB_CONNECTION", "postgres"),
		Host:         getEnvOrDefault("DB_HOST", "localhost"),
		Port:         port,
		Database:     getEnvOrDefault("DB_DATABASE", "mithril"),
		Username:     getEnvOrDefault("DB_USERNAME", "postgres"),
		Password:     getEnvOrDefault("DB_PASSWORD", "postgres"),
		SSLMode:      getEnvOrDefault("DB_SSLMODE", "disable"),
		MaxOpenConns: maxOpenConns,
		MaxIdleConns: maxIdleConns,
	}
}
