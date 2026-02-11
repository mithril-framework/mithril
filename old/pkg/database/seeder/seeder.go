package seeder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/gorm"
)

// SeederInterface defines the interface for seeders
type SeederInterface interface {
	Run() error
	Signature() string
}

// Manager manages database seeders
type Manager struct {
	db        *gorm.DB
	seeders   map[string]SeederInterface
	tableName string
}

// SeederRecord represents a seeder record in the database
type SeederRecord struct {
	ID        uint   `gorm:"primaryKey"`
	Seeder    string `gorm:"uniqueIndex;not null"`
	CreatedAt time.Time
}

// NewManager creates a new seeder manager
func NewManager(db *gorm.DB, tableName string) *Manager {
	return &Manager{
		db:        db,
		seeders:   make(map[string]SeederInterface),
		tableName: tableName,
	}
}

// RegisterSeeder registers a seeder
func (m *Manager) RegisterSeeder(seeder SeederInterface) {
	m.seeders[seeder.Signature()] = seeder
}

// CreateSeedersTable creates the seeders table
func (m *Manager) CreateSeedersTable() error {
	return m.db.Table(m.tableName).AutoMigrate(&SeederRecord{})
}

// Run runs all seeders
func (m *Manager) Run() error {
	// Create seeders table if it doesn't exist
	if err := m.CreateSeedersTable(); err != nil {
		return fmt.Errorf("failed to create seeders table: %w", err)
	}

	// Get ran seeders
	ranSeeders, err := m.getRanSeeders()
	if err != nil {
		return fmt.Errorf("failed to get ran seeders: %w", err)
	}

	// Get pending seeders
	pendingSeeders := m.getPendingSeeders(ranSeeders)

	if len(pendingSeeders) == 0 {
		fmt.Println("No pending seeders")
		return nil
	}

	// Run pending seeders
	for _, seeder := range pendingSeeders {
		if err := m.runSeeder(seeder); err != nil {
			return fmt.Errorf("failed to run seeder %s: %w", seeder.Signature(), err)
		}
		fmt.Printf("Seeded: %s\n", seeder.Signature())
	}

	return nil
}

// RunSpecific runs a specific seeder
func (m *Manager) RunSpecific(seederName string) error {
	seeder, exists := m.seeders[seederName]
	if !exists {
		return fmt.Errorf("seeder %s not found", seederName)
	}

	// Check if already run
	ran, err := m.isSeederRan(seederName)
	if err != nil {
		return fmt.Errorf("failed to check if seeder is ran: %w", err)
	}

	if ran {
		fmt.Printf("Seeder %s already run\n", seederName)
		return nil
	}

	// Run seeder
	if err := m.runSeeder(seeder); err != nil {
		return fmt.Errorf("failed to run seeder %s: %w", seeder.Signature(), err)
	}

	fmt.Printf("Seeded: %s\n", seeder.Signature())
	return nil
}

// Reset resets all seeders
func (m *Manager) Reset() error {
	// Create seeders table if it doesn't exist
	if err := m.CreateSeedersTable(); err != nil {
		return fmt.Errorf("failed to create seeders table: %w", err)
	}

	// Clear seeders table
	if err := m.db.Table(m.tableName).Where("1 = 1").Delete(&SeederRecord{}).Error; err != nil {
		return fmt.Errorf("failed to clear seeders table: %w", err)
	}

	fmt.Println("All seeders reset")
	return nil
}

// Status returns the status of all seeders
func (m *Manager) Status() ([]SeederStatus, error) {
	// Get ran seeders
	ranSeeders, err := m.getRanSeeders()
	if err != nil {
		return nil, fmt.Errorf("failed to get ran seeders: %w", err)
	}

	// Create status map
	statusMap := make(map[string]SeederStatus)
	for _, seeder := range ranSeeders {
		statusMap[seeder.Seeder] = SeederStatus{
			Seeder: seeder.Seeder,
			Status: "Ran",
		}
	}

	// Add pending seeders
	for signature := range m.seeders {
		if _, exists := statusMap[signature]; !exists {
			statusMap[signature] = SeederStatus{
				Seeder: signature,
				Status: "Pending",
			}
		}
	}

	// Convert to slice
	var statuses []SeederStatus
	for _, status := range statusMap {
		statuses = append(statuses, status)
	}

	return statuses, nil
}

// SeederStatus represents the status of a seeder
type SeederStatus struct {
	Seeder string
	Status string
}

// getRanSeeders retrieves all ran seeders
func (m *Manager) getRanSeeders() ([]SeederRecord, error) {
	var seeders []SeederRecord
	err := m.db.Table(m.tableName).Order("created_at ASC").Find(&seeders).Error
	return seeders, err
}

// getPendingSeeders returns seeders that haven't been run yet
func (m *Manager) getPendingSeeders(ranSeeders []SeederRecord) []SeederInterface {
	ranMap := make(map[string]bool)
	for _, record := range ranSeeders {
		ranMap[record.Seeder] = true
	}

	var pending []SeederInterface
	for signature, seeder := range m.seeders {
		if !ranMap[signature] {
			pending = append(pending, seeder)
		}
	}

	return pending
}

// isSeederRan checks if a seeder has been run
func (m *Manager) isSeederRan(seederName string) (bool, error) {
	var count int64
	err := m.db.Table(m.tableName).Where("seeder = ?", seederName).Count(&count).Error
	return count > 0, err
}

// runSeeder runs a single seeder
func (m *Manager) runSeeder(seeder SeederInterface) error {
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

	// Run seeder
	if err := seeder.Run(); err != nil {
		tx.Rollback()
		return err
	}

	// Record seeder
	record := SeederRecord{
		Seeder:    seeder.Signature(),
		CreatedAt: time.Now(),
	}

	if err := tx.Table(m.tableName).Create(&record).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Commit transaction
	return tx.Commit().Error
}

// GenerateSeederFile generates a new seeder file
func (m *Manager) GenerateSeederFile(name string) error {
	filename := fmt.Sprintf("%s_seeder.go", strings.ToLower(name))

	// Create seeders directory if it doesn't exist
	seedersDir := "database/seeders"
	if err := os.MkdirAll(seedersDir, 0755); err != nil {
		return err
	}

	filepath := filepath.Join(seedersDir, filename)

	// Generate seeder content
	content := fmt.Sprintf(`package seeders

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
		cases.Title(language.English).String(name),
		cases.Title(language.English).String(name),
		cases.Title(language.English).String(name),
		cases.Title(language.English).String(name),
		cases.Title(language.English).String(name),
		cases.Title(language.English).String(name),
		cases.Title(language.English).String(name),
	)

	return os.WriteFile(filepath, []byte(content), 0644)
}
