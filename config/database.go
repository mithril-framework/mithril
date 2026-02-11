package config

import (
	"strconv"
	"time"
)

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	// Primary database connection
	Driver   string `env:"DB_DRIVER" default:"postgres"`
	Host     string `env:"DB_HOST" default:"localhost"`
	Port     int    `env:"DB_PORT" default:"5432"`
	Username string `env:"DB_USERNAME" default:"mithril"`
	Password string `env:"DB_PASSWORD" default:"password"`
	Database string `env:"DB_NAME" default:"mithril"`

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
	case "sqlite", "sqlite3":
		return c.getSQLiteDSN()
	case "mongodb":
		return c.getMongoDSN()
	default:
		return c.getPostgresDSN()
	}
}

// getPostgresDSN returns PostgreSQL connection string
func (c *DatabaseConfig) getPostgresDSN() string {
	return "host=" + c.Host +
		" port=" + strconv.Itoa(c.Port) +
		" user=" + c.Username +
		" password=" + c.Password +
		" dbname=" + c.Database +
		" sslmode=" + c.SSLMode
}

// getMySQLDSN returns MySQL connection string
func (c *DatabaseConfig) getMySQLDSN() string {
	return c.Username + ":" + c.Password +
		"@tcp(" + c.Host + ":" + strconv.Itoa(c.Port) + ")/" + c.Database +
		"?charset=" + c.MySQLCharset +
		"&parseTime=" + strconv.FormatBool(c.MySQLParseTime) +
		"&loc=" + c.MySQLLoc
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
	return "mongodb://" + c.Host + ":" + strconv.Itoa(c.Port) + "/" + c.MongoDatabase
}

// GetRedisAddr returns Redis address
func (c *DatabaseConfig) GetRedisAddr() string {
	return c.RedisHost + ":" + strconv.Itoa(c.RedisPort)
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
	return c.Driver == "sqlite" || c.Driver == "sqlite3"
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

// LoadDatabaseConfig loads database configuration from environment variables
func LoadDatabaseConfig() *DatabaseConfig {
	port, _ := strconv.Atoi(getEnv("DB_PORT", "5432"))
	maxOpenConns, _ := strconv.Atoi(getEnv("DB_MAX_OPEN_CONNS", "25"))
	maxIdleConns, _ := strconv.Atoi(getEnv("DB_MAX_IDLE_CONNS", "5"))

	return &DatabaseConfig{
		Driver:       getEnv("DB_DRIVER", "postgres"),
		Host:         getEnv("DB_HOST", "localhost"),
		Port:         port,
		Database:     getEnv("DB_NAME", "mithril"),
		Username:     getEnv("DB_USERNAME", "mithril"),
		Password:     getEnv("DB_PASSWORD", "password"),
		SSLMode:      getEnv("DB_SSL_MODE", "disable"),
		MaxOpenConns: maxOpenConns,
		MaxIdleConns: maxIdleConns,
		SeederPath:   getEnv("DB_SEEDER_PATH", "./database/seeders"),
		BackupPath:   getEnv("DB_BACKUP_PATH", "./database/backups"),
		SQLitePath:   getEnv("SQLITE_PATH", "./database/sqlite.db"),
		MySQLCharset: getEnv("MYSQL_CHARSET", "utf8mb4"),
		MySQLLoc:     getEnv("MYSQL_LOC", "Local"),
	}
}
