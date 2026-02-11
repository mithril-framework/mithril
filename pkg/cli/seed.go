package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mithril-framework/mithril/pkg/database/connection"
	"github.com/mithril-framework/mithril/pkg/database/seeder"
	"gorm.io/gorm"
)

// SeedCommand handles seeder commands
type SeedCommand struct {
	manager *connection.Manager
}

// NewSeedCommand creates a new seed command
func NewSeedCommand(manager *connection.Manager) *SeedCommand {
	return &SeedCommand{
		manager: manager,
	}
}

// Run runs the seed command
func (sc *SeedCommand) Run(args []string) error {
	if len(args) == 0 {
		return sc.runAllSeeders()
	}

	switch args[0] {
	case "status":
		return sc.seederStatus()
	case "reset":
		return sc.resetSeeders()
	case "make":
		if len(args) < 2 {
			return fmt.Errorf("seeder name is required")
		}
		return sc.makeSeeder(args[1])
	default:
		// Run specific seeder
		return sc.runSpecificSeeder(args[0])
	}
}

// runAllSeeders runs all pending seeders
func (sc *SeedCommand) runAllSeeders() error {
	db, err := sc.getDefaultConnection()
	if err != nil {
		return err
	}

	seederManager := seeder.NewManager(db, "seeders")
	if err := seederManager.Run(); err != nil {
		return fmt.Errorf("failed to run seeders: %w", err)
	}

	fmt.Println("Seeders completed successfully")
	return nil
}

// runSpecificSeeder runs a specific seeder
func (sc *SeedCommand) runSpecificSeeder(seederName string) error {
	db, err := sc.getDefaultConnection()
	if err != nil {
		return err
	}

	seederManager := seeder.NewManager(db, "seeders")
	if err := seederManager.RunSpecific(seederName); err != nil {
		return fmt.Errorf("failed to run seeder %s: %w", seederName, err)
	}

	fmt.Printf("Seeder %s completed successfully\n", seederName)
	return nil
}

// seederStatus shows seeder status
func (sc *SeedCommand) seederStatus() error {
	db, err := sc.getDefaultConnection()
	if err != nil {
		return err
	}

	seederManager := seeder.NewManager(db, "seeders")
	statuses, err := seederManager.Status()
	if err != nil {
		return fmt.Errorf("failed to get seeder status: %w", err)
	}

	fmt.Println("Seeder Status:")
	fmt.Println("==============")
	for _, status := range statuses {
		fmt.Printf("%-50s %s\n", status.Seeder, status.Status)
	}

	return nil
}

// resetSeeders resets all seeders
func (sc *SeedCommand) resetSeeders() error {
	db, err := sc.getDefaultConnection()
	if err != nil {
		return err
	}

	seederManager := seeder.NewManager(db, "seeders")
	if err := seederManager.Reset(); err != nil {
		return fmt.Errorf("failed to reset seeders: %w", err)
	}

	fmt.Println("All seeders reset successfully")
	return nil
}

// makeSeeder creates a new seeder file
func (sc *SeedCommand) makeSeeder(name string) error {
	// Create seeders directory if it doesn't exist
	seedersDir := "database/seeders"
	if err := os.MkdirAll(seedersDir, 0755); err != nil {
		return err
	}

	// Generate seeder file
	filename := fmt.Sprintf("%s_seeder.go", strings.ToLower(name))
	filepath := filepath.Join(seedersDir, filename)

	// Generate seeder content
	content := sc.generateSeederContent(name)

	if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create seeder file: %w", err)
	}

	fmt.Printf("Created seeder: %s\n", filename)
	return nil
}

// getDefaultConnection gets the default database connection
func (sc *SeedCommand) getDefaultConnection() (*gorm.DB, error) {
	// TODO: Load database configuration from environment/config
	// For now, use a simple connection string
	dsn := sc.getDatabaseDSN()

	db, err := sc.manager.Connect("default", "postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}

// getDatabaseDSN gets the database DSN from environment
func (sc *SeedCommand) getDatabaseDSN() string {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "mithril")
	password := getEnv("DB_PASSWORD", "password")
	database := getEnv("DB_NAME", "mithril")
	sslmode := getEnv("DB_SSLMODE", "disable")

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, database, sslmode)
}

// generateSeederContent generates the content for a seeder file
func (sc *SeedCommand) generateSeederContent(name string) string {
	className := strings.Title(name)

	return fmt.Sprintf(`package seeders

import (
	"gorm.io/gorm"
)

type %sSeeder struct {
	db *gorm.DB
}

func New%sSeeder(db *gorm.DB) *%sSeeder {
	return &%sSeeder{db: db}
}

func (s *%sSeeder) Run() error {
	// TODO: Implement seeder logic
	return nil
}

func (s *%sSeeder) Signature() string {
	return "%sSeeder"
}
`,
		className,
		className,
		className,
		className,
		className,
		className,
		className,
	)
}
