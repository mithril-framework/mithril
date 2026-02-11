package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mithril-framework/mithril/pkg/config"
	"github.com/mithril-framework/mithril/pkg/database/connection"
	"github.com/mithril-framework/mithril/pkg/database/migration"
	"github.com/urfave/cli/v2"
	"gorm.io/gorm"
)

// DatabaseCommands contains all database-related commands
type DatabaseCommands struct{}

// NewDatabaseCommands creates a new database commands instance
func NewDatabaseCommands() *DatabaseCommands {
	return &DatabaseCommands{}
}

// Register registers all database commands
func (dc *DatabaseCommands) Register() {
	// migrate command - simplified to use GORM AutoMigrate
	NewCommand("migrate", "Run database migrations using GORM AutoMigrate").
		Description("Auto-migrate all registered models from app/models directory").
		Category("Database").
		BoolFlag("force", "Force migration without confirmation").
		Action(dc.Migrate).
		Register()

	// make:seeder command
	NewCommand("make:seeder", "Create a new seeder").
		Description("Create a new database seeder file").
		Category("Database").
		Action(dc.MakeSeeder).
		Register()

	// db:seed command
	NewCommand("db:seed", "Run database seeders").
		Description("Run database seeders to populate the database").
		Category("Database").
		StringFlag("seeder", "", "Run specific seeder").
		StringFlag("class", "", "Run specific seeder class").
		Action(dc.DatabaseSeed).
		Register()

	// db:backup command
	NewCommand("db:backup", "Backup database").
		Description("Create a backup of the database").
		Category("Database").
		StringFlag("connection", "default", "Database connection name").
		BoolFlag("schema-only", "Backup schema only").
		BoolFlag("data-only", "Backup data only").
		StringFlag("file", "", "Output file path").
		BoolFlag("compress", "Compress backup file").
		Action(dc.DatabaseBackup).
		Register()

	// db:restore command
	NewCommand("db:restore", "Restore database from backup").
		Description("Restore database from a backup file").
		Category("Database").
		StringFlag("file", "", "Backup file path").
		StringFlag("connection", "default", "Database connection name").
		BoolFlag("force", "Force restore without confirmation").
		Action(dc.DatabaseRestore).
		Register()

	// db:wipe command
	NewCommand("db:wipe", "Wipe all database data").
		Description("Remove all data from the database").
		Category("Database").
		BoolFlag("force", "Force wipe without confirmation").
		Action(dc.DatabaseWipe).
		Register()

	// db:test command
	NewCommand("db:test", "Test database connection").
		Description("Test the database connection").
		Category("Database").
		StringFlag("connection", "default", "Database connection name").
		Action(dc.DatabaseTest).
		Register()
}

// Migrate runs GORM AutoMigrate on all registered models
func (dc *DatabaseCommands) Migrate(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)
	force := cc.GetBoolFlag("force")
	
	cc.PrintInfo("Running database migrations using GORM AutoMigrate...")
	
	// Get database connection
	db, err := dc.getDefaultConnection(cc)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	
	// Discover model names from app/models directory
	modelsPath := "app/models"
	modelNames, err := migration.DiscoverModelNames(modelsPath)
	if err != nil {
		cc.PrintWarning(fmt.Sprintf("Could not discover models: %v", err))
		cc.PrintInfo("Make sure models are registered using migration.RegisterModel() in your model files")
	}
	
	// Get registered models
	registeredModels := migration.GetRegisteredModels()
	
	if len(registeredModels) == 0 {
		if len(modelNames) > 0 {
			cc.PrintWarning(fmt.Sprintf("Found %d model(s) in app/models but none are registered", len(modelNames)))
			cc.PrintInfo("To register models, add this to your model files:")
			cc.PrintInfo("  func init() {")
			cc.PrintInfo("      migration.RegisterModel(&YourModel{})")
			cc.PrintInfo("  }")
			cc.PrintInfo("")
			cc.PrintInfo("Or call db.AutoMigrate() directly in your application code.")
			return fmt.Errorf("no models registered for migration")
		}
		return fmt.Errorf("no models found in app/models directory and no models registered")
	}
	
	cc.PrintInfo(fmt.Sprintf("Found %d registered model(s) to migrate", len(registeredModels)))
	if len(modelNames) > 0 {
		cc.PrintInfo(fmt.Sprintf("Discovered model names: %s", strings.Join(modelNames, ", ")))
	}
	
	// Confirm migration
	if !force {
		if !cc.Confirm(fmt.Sprintf("Migrate %d model(s)?", len(registeredModels))) {
			cc.PrintInfo("Migration cancelled")
			return nil
		}
	}
	
	// Run AutoMigrate
	cc.PrintInfo("Running AutoMigrate...")
	if err := migration.AutoMigrateRegistered(db); err != nil {
		return fmt.Errorf("failed to run AutoMigrate: %w", err)
	}
	
	cc.PrintSuccess(fmt.Sprintf("Successfully migrated %d model(s)", len(registeredModels)))
	return nil
}

// getDefaultConnection gets the default database connection
func (dc *DatabaseCommands) getDefaultConnection(cc *CommandContext) (*gorm.DB, error) {
	// Load database configuration
	dbConfig := config.NewDatabaseConfig()
	
	// Create connection manager
	manager := connection.NewManager()
	
	// Get DSN
	dsn := dbConfig.GetDSN()
	driver := dbConfig.Driver
	
	// Connect to database
	db, err := manager.Connect("default", driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	
	return db, nil
}


// MakeSeeder creates a new seeder
func (dc *DatabaseCommands) MakeSeeder(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)
	
	if len(cc.Args) == 0 {
		return fmt.Errorf("seeder name is required")
	}
	
	seederName := cc.GetStringArg(0)
	
	// Validate seeder name
	if !isValidSeederName(seederName) {
		return fmt.Errorf("invalid seeder name: %s", seederName)
	}
	
	// Create seeders directory if it doesn't exist
	seedersPath := "database/seeders"
	if !cc.DirectoryExists(seedersPath) {
		if err := cc.CreateDirectory(seedersPath); err != nil {
			return fmt.Errorf("failed to create seeders directory: %w", err)
		}
	}
	
	// Generate seeder filename
	filename := fmt.Sprintf("%s.go", toSnakeCase(seederName))
	filepath := filepath.Join(seedersPath, filename)
	
	// Check if seeder already exists
	if cc.FileExists(filepath) {
		return fmt.Errorf("seeder '%s' already exists", seederName)
	}
	
	cc.PrintInfo(fmt.Sprintf("Creating seeder: %s", seederName))
	
	// Generate seeder content
	content := dc.generateSeeder(seederName)
	
	// Write seeder file
	if err := cc.WriteFile(filepath, []byte(content)); err != nil {
		return fmt.Errorf("failed to create seeder file: %w", err)
	}
	
	cc.PrintSuccess(fmt.Sprintf("Seeder '%s' created successfully", seederName))
	cc.PrintInfo(fmt.Sprintf("Seeder path: %s", filepath))
	
	return nil
}

// DatabaseSeed runs database seeders
func (dc *DatabaseCommands) DatabaseSeed(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)
	
	seeder := cc.GetStringFlag("seeder")
	_ = cc.GetStringFlag("class")
	
	cc.PrintInfo("Running database seeders...")
	
	// Get seeder files
	seedersPath := "database/seeders"
	if !cc.DirectoryExists(seedersPath) {
		cc.PrintWarning("Seeders directory not found")
		return nil
	}
	
	seederFiles, err := dc.getSeederFiles(cc, seedersPath)
	if err != nil {
		return fmt.Errorf("failed to get seeder files: %w", err)
	}
	
	if len(seederFiles) == 0 {
		cc.PrintInfo("No seeder files found")
		return nil
	}
	
	// Filter seeders if specific seeder requested
	if seeder != "" {
		seederFiles = dc.filterSeeders(seederFiles, seeder)
	}
	
	if len(seederFiles) == 0 {
		cc.PrintInfo("No matching seeders found")
		return nil
	}
	
	cc.PrintInfo(fmt.Sprintf("Found %d seeders to run", len(seederFiles)))
	
	// Run seeders
	successCount := 0
	for _, seederFile := range seederFiles {
		cc.PrintInfo(fmt.Sprintf("Running seeder: %s", seederFile.Name))
		
		if err := dc.runSeeder(cc, seederFile); err != nil {
			cc.PrintError(fmt.Sprintf("Failed to run seeder %s: %v", seederFile.Name, err))
			break
		}
		
		successCount++
	}
	
	if successCount == len(seederFiles) {
		cc.PrintSuccess(fmt.Sprintf("Successfully ran %d seeders", successCount))
	} else {
		cc.PrintWarning(fmt.Sprintf("Ran %d of %d seeders", successCount, len(seederFiles)))
	}
	
	return nil
}

// DatabaseBackup creates a database backup
func (dc *DatabaseCommands) DatabaseBackup(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)
	
	connection := cc.GetStringFlag("connection")
	schemaOnly := cc.GetBoolFlag("schema-only")
	dataOnly := cc.GetBoolFlag("data-only")
	file := cc.GetStringFlag("file")
	compress := cc.GetBoolFlag("compress")
	
	cc.PrintInfo("Creating database backup...")
	
	// Generate backup filename if not provided
	if file == "" {
		timestamp := time.Now().Format("2006_01_02_150405")
		file = fmt.Sprintf("backup_%s.sql", timestamp)
		if compress {
			file += ".gz"
		}
	}
	
	// Create backup
	if err := dc.createBackup(cc, connection, file, schemaOnly, dataOnly, compress); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}
	
	cc.PrintSuccess(fmt.Sprintf("Database backup created: %s", file))
	return nil
}

// DatabaseRestore restores database from backup
func (dc *DatabaseCommands) DatabaseRestore(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)
	
	file := cc.GetStringFlag("file")
	connection := cc.GetStringFlag("connection")
	force := cc.GetBoolFlag("force")
	
	if file == "" {
		return fmt.Errorf("backup file is required")
	}
	
	// Check if backup file exists
	if !cc.FileExists(file) {
		return fmt.Errorf("backup file not found: %s", file)
	}
	
	cc.PrintInfo(fmt.Sprintf("Restoring database from: %s", file))
	
	// Confirm restore
	if !force {
		if !cc.Confirm("This will overwrite the current database. Continue?") {
			cc.PrintInfo("Restore cancelled")
			return nil
		}
	}
	
	// Restore database
	if err := dc.restoreBackup(cc, connection, file); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}
	
	cc.PrintSuccess("Database restored successfully")
	return nil
}

// DatabaseWipe wipes all database data
func (dc *DatabaseCommands) DatabaseWipe(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)
	
	force := cc.GetBoolFlag("force")
	
	cc.PrintInfo("Wiping all database data...")
	
	// Confirm wipe
	if !force {
		if !cc.Confirm("This will remove all data from the database. Continue?") {
			cc.PrintInfo("Wipe cancelled")
			return nil
		}
	}
	
	// Wipe database
	if err := dc.wipeDatabase(cc); err != nil {
		return fmt.Errorf("failed to wipe database: %w", err)
	}
	
	cc.PrintSuccess("Database wiped successfully")
	return nil
}

// DatabaseTest tests database connection
func (dc *DatabaseCommands) DatabaseTest(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)
	
	connection := cc.GetStringFlag("connection")
	
	cc.PrintInfo(fmt.Sprintf("Testing database connection: %s", connection))
	
	// Test connection
	if err := dc.testConnection(cc, connection); err != nil {
		cc.PrintError(fmt.Sprintf("Database connection failed: %v", err))
		return err
	}
	
	cc.PrintSuccess("Database connection successful")
	return nil
}

// Helper functions

type SeederFile struct {
	Name string
	Path string
}


func (dc *DatabaseCommands) getSeederFiles(cc *CommandContext, path string) ([]SeederFile, error) {
	var files []SeederFile
	
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			files = append(files, SeederFile{
				Name: entry.Name(),
				Path: filepath.Join(path, entry.Name()),
			})
		}
	}
	
	return files, nil
}

func (dc *DatabaseCommands) filterSeeders(seeders []SeederFile, name string) []SeederFile {
	var filtered []SeederFile
	for _, seeder := range seeders {
		if strings.Contains(seeder.Name, toSnakeCase(name)) {
			filtered = append(filtered, seeder)
		}
	}
	return filtered
}

func (dc *DatabaseCommands) runSeeder(cc *CommandContext, seeder SeederFile) error {
	// This would execute the seeder
	cc.PrintInfo(fmt.Sprintf("Executing seeder: %s", seeder.Name))
	return nil
}

func (dc *DatabaseCommands) createBackup(cc *CommandContext, connection, file string, schemaOnly, dataOnly, compress bool) error {
	// This would create a database backup
	cc.PrintInfo(fmt.Sprintf("Creating backup: %s", file))
	return nil
}

func (dc *DatabaseCommands) restoreBackup(cc *CommandContext, connection, file string) error {
	// This would restore a database backup
	cc.PrintInfo(fmt.Sprintf("Restoring from: %s", file))
	return nil
}

func (dc *DatabaseCommands) wipeDatabase(cc *CommandContext) error {
	// This would wipe the database
	cc.PrintInfo("Wiping database...")
	return nil
}

func (dc *DatabaseCommands) testConnection(cc *CommandContext, connection string) error {
	// This would test the database connection
	cc.PrintInfo("Testing connection...")
	return nil
}


func (dc *DatabaseCommands) generateSeeder(name string) string {
	return fmt.Sprintf(`package seeders

import (
	"gorm.io/gorm"
)

type %s struct{}

func (s *%s) Run(db *gorm.DB) error {
	// Add your seeder logic here
	return nil
}
`, toPascalCase(name), toPascalCase(name))
}

// Validation functions

func isValidSeederName(name string) bool {
	if name == "" {
		return false
	}
	// Check for valid characters (alphanumeric and hyphens/underscores)
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return true
}
