// Package migration provides database migration functionality.
//
// DEPRECATED: The custom migration system (Migrator, MigrationInterface, etc.) is deprecated.
// Use GORM's AutoMigrate instead. Register models using RegisterModel() and call AutoMigrateRegistered().
package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/gorm"
)

// Migration represents a single migration
type Migration struct {
	ID        uint   `gorm:"primaryKey"`
	Migration string `gorm:"uniqueIndex;not null"`
	Batch     int    `gorm:"not null"`
	CreatedAt time.Time
}

// Migrator handles database migrations
type Migrator struct {
	db         *gorm.DB
	tableName  string
	migrations map[string]MigrationInterface
}

// MigrationInterface defines the interface for migrations
// DEPRECATED: Use GORM AutoMigrate instead
type MigrationInterface interface {
	Up() error
	Down() error
	Signature() string
}

// NewMigrator creates a new migrator instance
// DEPRECATED: Use RegisterModel() and AutoMigrateRegistered() instead
func NewMigrator(db *gorm.DB, tableName string) *Migrator {
	return &Migrator{
		db:         db,
		tableName:  tableName,
		migrations: make(map[string]MigrationInterface),
	}
}

// RegisterMigration registers a migration
func (m *Migrator) RegisterMigration(migration MigrationInterface) {
	m.migrations[migration.Signature()] = migration
}

// CreateMigrationsTable creates the migrations table
func (m *Migrator) CreateMigrationsTable() error {
	return m.db.Table(m.tableName).AutoMigrate(&Migration{})
}

// Run runs all pending migrations
func (m *Migrator) Run() error {
	// Create migrations table if it doesn't exist
	if err := m.CreateMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get ran migrations
	ranMigrations, err := m.getRanMigrations()
	if err != nil {
		return fmt.Errorf("failed to get ran migrations: %w", err)
	}

	// Get pending migrations
	pendingMigrations := m.getPendingMigrations(ranMigrations)

	if len(pendingMigrations) == 0 {
		fmt.Println("No pending migrations")
		return nil
	}

	// Get next batch number
	batch, err := m.getNextBatchNumber()
	if err != nil {
		return fmt.Errorf("failed to get next batch number: %w", err)
	}

	// Run pending migrations
	for _, migration := range pendingMigrations {
		if err := m.runMigration(migration, batch); err != nil {
			return fmt.Errorf("failed to run migration %s: %w", migration.Signature(), err)
		}
		fmt.Printf("Migrated: %s\n", migration.Signature())
	}

	return nil
}

// Rollback rolls back the last batch of migrations
func (m *Migrator) Rollback(steps int) error {
	// Get migrations to rollback
	migrationsToRollback, err := m.getMigrationsToRollback(steps)
	if err != nil {
		return fmt.Errorf("failed to get migrations to rollback: %w", err)
	}

	if len(migrationsToRollback) == 0 {
		fmt.Println("Nothing to rollback")
		return nil
	}

	// Rollback migrations
	for _, migration := range migrationsToRollback {
		if err := m.rollbackMigration(migration); err != nil {
			return fmt.Errorf("failed to rollback migration %s: %w", migration.Signature(), err)
		}
		fmt.Printf("Rolled back: %s\n", migration.Signature())
	}

	return nil
}

// Status returns the status of all migrations
func (m *Migrator) Status() ([]MigrationStatus, error) {
	// Get ran migrations
	ranMigrations, err := m.getRanMigrations()
	if err != nil {
		return nil, fmt.Errorf("failed to get ran migrations: %w", err)
	}

	// Create status map
	statusMap := make(map[string]MigrationStatus)
	for _, migration := range ranMigrations {
		statusMap[migration.Migration] = MigrationStatus{
			Migration: migration.Migration,
			Batch:     migration.Batch,
			Status:    "Ran",
		}
	}

	// Add pending migrations
	for signature := range m.migrations {
		if _, exists := statusMap[signature]; !exists {
			statusMap[signature] = MigrationStatus{
				Migration: signature,
				Batch:     0,
				Status:    "Pending",
			}
		}
	}

	// Convert to slice and sort
	var statuses []MigrationStatus
	for _, status := range statusMap {
		statuses = append(statuses, status)
	}

	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].Migration < statuses[j].Migration
	})

	return statuses, nil
}

// MigrationStatus represents the status of a migration
type MigrationStatus struct {
	Migration string
	Batch     int
	Status    string
}

// getRanMigrations retrieves all ran migrations
func (m *Migrator) getRanMigrations() ([]Migration, error) {
	var migrations []Migration
	err := m.db.Table(m.tableName).Order("batch ASC, migration ASC").Find(&migrations).Error
	return migrations, err
}

// getPendingMigrations returns migrations that haven't been run yet
func (m *Migrator) getPendingMigrations(ranMigrations []Migration) []MigrationInterface {
	ranMap := make(map[string]bool)
	for _, migration := range ranMigrations {
		ranMap[migration.Migration] = true
	}

	var pending []MigrationInterface
	for signature, migration := range m.migrations {
		if !ranMap[signature] {
			pending = append(pending, migration)
		}
	}

	// Sort by signature
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Signature() < pending[j].Signature()
	})

	return pending
}

// getNextBatchNumber returns the next batch number
func (m *Migrator) getNextBatchNumber() (int, error) {
	var maxBatch int
	err := m.db.Table(m.tableName).Select("COALESCE(MAX(batch), 0)").Scan(&maxBatch).Error
	return maxBatch + 1, err
}

// runMigration runs a single migration
func (m *Migrator) runMigration(migration MigrationInterface, batch int) error {
	// Start transaction
	tx := m.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Run migration
	if err := migration.Up(); err != nil {
		tx.Rollback()
		return err
	}

	// Record migration
	record := Migration{
		Migration: migration.Signature(),
		Batch:     batch,
		CreatedAt: time.Now(),
	}

	if err := tx.Table(m.tableName).Create(&record).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Commit transaction
	return tx.Commit().Error
}

// getMigrationsToRollback returns migrations to rollback
func (m *Migrator) getMigrationsToRollback(steps int) ([]MigrationInterface, error) {
	var migrations []Migration
	err := m.db.Table(m.tableName).Order("batch DESC, migration DESC").Limit(steps).Find(&migrations).Error
	if err != nil {
		return nil, err
	}

	var result []MigrationInterface
	for _, migration := range migrations {
		if m, exists := m.migrations[migration.Migration]; exists {
			result = append(result, m)
		}
	}

	return result, nil
}

// rollbackMigration rolls back a single migration
func (m *Migrator) rollbackMigration(migration MigrationInterface) error {
	// Start transaction
	tx := m.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Run rollback
	if err := migration.Down(); err != nil {
		tx.Rollback()
		return err
	}

	// Remove migration record
	if err := tx.Table(m.tableName).Where("migration = ?", migration.Signature()).Delete(&Migration{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Commit transaction
	return tx.Commit().Error
}

// GenerateMigrationFile generates a new migration file
func (m *Migrator) GenerateMigrationFile(name string) error {
	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("%s_%s.go", timestamp, name)

	// Create migrations directory if it doesn't exist
	migrationsDir := "database/migrations"
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return err
	}

	filepath := filepath.Join(migrationsDir, filename)

	// Generate migration content
	content := fmt.Sprintf(`package migrations

import (
	"gorm.io/gorm"
)

type %s struct{}

func (m *%s) Up() error {
	// TODO: Implement migration
	return nil
}

func (m *%s) Down() error {
	// TODO: Implement rollback
	return nil
}

func (m *%s) Signature() string {
	return "%s_%s"
}
`,
		cases.Title(language.English).String(name),
		cases.Title(language.English).String(name),
		cases.Title(language.English).String(name),
		cases.Title(language.English).String(name),
		timestamp,
		name,
	)

	return os.WriteFile(filepath, []byte(content), 0644)
}
