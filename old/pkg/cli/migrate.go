package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mithril-framework/mithril/pkg/database/connection"
	"github.com/mithril-framework/mithril/pkg/database/migration"
	"gorm.io/gorm"
)

// MigrateCommand handles migration commands
type MigrateCommand struct {
	manager *connection.Manager
}

// NewMigrateCommand creates a new migrate command
func NewMigrateCommand(manager *connection.Manager) *MigrateCommand {
	return &MigrateCommand{
		manager: manager,
	}
}

// Run runs the migrate command
func (mc *MigrateCommand) Run(args []string) error {
	if len(args) == 0 {
		return mc.runMigrations()
	}

	switch args[0] {
	case "rollback":
		steps := 1
		if len(args) > 1 {
			if s, err := strconv.Atoi(args[1]); err == nil {
				steps = s
			}
		}
		return mc.rollbackMigrations(steps)
	case "fresh":
		return mc.freshMigrations()
	case "status":
		return mc.migrationStatus()
	case "make":
		if len(args) < 2 {
			return fmt.Errorf("migration name is required")
		}
		return mc.makeMigration(args[1])
	default:
		return fmt.Errorf("unknown migrate command: %s", args[0])
	}
}

// runMigrations runs AutoMigrate on all registered models
func (mc *MigrateCommand) runMigrations() error {
	db, err := mc.getDefaultConnection()
	if err != nil {
		return err
	}

	// Get registered models
	registeredModels := migration.GetRegisteredModels()
	if len(registeredModels) == 0 {
		return fmt.Errorf("no models registered for migration. Register models using migration.RegisterModel() in your model files")
	}

	fmt.Printf("Running AutoMigrate on %d model(s)...\n", len(registeredModels))
	if err := migration.AutoMigrateRegistered(db); err != nil {
		return fmt.Errorf("failed to run AutoMigrate: %w", err)
	}

	fmt.Println("Migrations completed successfully")
	return nil
}

// rollbackMigrations is deprecated - GORM AutoMigrate doesn't support rollback
func (mc *MigrateCommand) rollbackMigrations(steps int) error {
	return fmt.Errorf("rollback is not supported with GORM AutoMigrate. AutoMigrate automatically updates schema based on model definitions")
}

// freshMigrations drops all tables and re-runs AutoMigrate
func (mc *MigrateCommand) freshMigrations() error {
	db, err := mc.getDefaultConnection()
	if err != nil {
		return err
	}

	// Drop all tables
	if err := mc.dropAllTables(db); err != nil {
		return fmt.Errorf("failed to drop tables: %w", err)
	}

	// Run migrations
	return mc.runMigrations()
}

// migrationStatus shows registered models
func (mc *MigrateCommand) migrationStatus() error {
	registeredModels := migration.GetRegisteredModels()
	modelNames, _ := migration.DiscoverModelNames("app/models")

	fmt.Println("Migration Status:")
	fmt.Println("=================")
	fmt.Printf("Registered models: %d\n", len(registeredModels))
	if len(modelNames) > 0 {
		fmt.Printf("Discovered model names: %s\n", strings.Join(modelNames, ", "))
	}
	
	if len(registeredModels) == 0 {
		fmt.Println("\nNo models registered. Register models using migration.RegisterModel() in your model files.")
	}

	return nil
}

// makeMigration is deprecated - migrations are no longer needed with GORM AutoMigrate
func (mc *MigrateCommand) makeMigration(name string) error {
	return fmt.Errorf("make:migration is deprecated. With GORM AutoMigrate, you don't need migration files. Just update your models and run migrate")
}

// getDefaultConnection gets the default database connection
func (mc *MigrateCommand) getDefaultConnection() (*gorm.DB, error) {
	// TODO: Load database configuration from environment/config
	// For now, use a simple connection string
	dsn := mc.getDatabaseDSN()

	db, err := mc.manager.Connect("default", "postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}

// getDatabaseDSN gets the database DSN from environment
func (mc *MigrateCommand) getDatabaseDSN() string {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "mithril")
	password := getEnv("DB_PASSWORD", "password")
	database := getEnv("DB_NAME", "mithril")
	sslmode := getEnv("DB_SSLMODE", "disable")

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, database, sslmode)
}

// dropAllTables drops all tables in the database
func (mc *MigrateCommand) dropAllTables(db *gorm.DB) error {
	// Get all table names
	var tables []string
	if err := db.Raw("SELECT tablename FROM pg_tables WHERE schemaname = 'public'").Scan(&tables).Error; err != nil {
		return err
	}

	// Drop each table
	for _, table := range tables {
		if err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table)).Error; err != nil {
			return err
		}
	}

	return nil
}


// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
