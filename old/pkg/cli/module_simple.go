package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ModuleFlags represents the flags for module generation
type ModuleFlags struct {
	Full    bool // Generate full CRUD (web + API)
	APIOnly bool // Generate API routes only
	WebOnly bool // Generate web routes only
}

// SimpleModuleGenerator handles the creation of new modules (simplified version)
type SimpleModuleGenerator struct {
	ModuleName string
	Flags      ModuleFlags
}

// NewSimpleModuleGenerator creates a new SimpleModuleGenerator instance
func NewSimpleModuleGenerator(moduleName string, flags ModuleFlags) *SimpleModuleGenerator {
	return &SimpleModuleGenerator{
		ModuleName: moduleName,
		Flags:      flags,
	}
}

// Generate creates a new module
func (mg *SimpleModuleGenerator) Generate() error {
	// Determine module type based on flags
	moduleType := mg.determineModuleType()

	// Create module directory structure
	if err := mg.createModuleStructure(); err != nil {
		return err
	}

	// Generate basic module files
	if err := mg.generateBasicFiles(moduleType); err != nil {
		return err
	}

	fmt.Printf("✅ Module '%s' created successfully!\n", mg.ModuleName)
	fmt.Printf("📁 Directory: app/modules/%s\n", mg.ModuleName)
	fmt.Println("🚀 Next steps:")
	fmt.Println("   1. Register the module in main.go")
	fmt.Println("   2. Run migrations if models were created")
	fmt.Println("   3. Update routes in routes/api.go or routes/web.go")

	return nil
}

// determineModuleType determines the module type based on flags
func (mg *SimpleModuleGenerator) determineModuleType() string {
	if mg.Flags.APIOnly {
		return "api"
	} else if mg.Flags.WebOnly {
		return "web"
	}
	return "full"
}

// createModuleStructure creates the module directory structure
func (mg *SimpleModuleGenerator) createModuleStructure() error {
	moduleDir := filepath.Join("app", "modules", mg.ModuleName)

	directories := []string{
		moduleDir,
		filepath.Join(moduleDir, "controllers"),
		filepath.Join(moduleDir, "schemas"),
		filepath.Join(moduleDir, "models"),
		filepath.Join(moduleDir, "middleware"),
		filepath.Join(moduleDir, "services"),
	}

	for _, dir := range directories {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// generateBasicFiles generates basic module files
func (mg *SimpleModuleGenerator) generateBasicFiles(moduleType string) error {
	moduleDir := filepath.Join("app", "modules", mg.ModuleName)

	// Generate module.go
	if err := mg.generateModuleGo(moduleDir, moduleType); err != nil {
		return err
	}

	// Generate service.go
	if err := mg.generateServiceGo(moduleDir); err != nil {
		return err
	}

	// Generate model.go
	if err := mg.generateModelGo(moduleDir); err != nil {
		return err
	}

	// Generate README
	if err := mg.generateREADME(moduleDir, moduleType); err != nil {
		return err
	}

	return nil
}

// generateModuleGo generates the main module file
func (mg *SimpleModuleGenerator) generateModuleGo(moduleDir, moduleType string) error {
	content := fmt.Sprintf(`package %s

import (
	"github.com/gofiber/fiber/v2"
	"github.com/mithril-framework/mithril/pkg/core"
)

// Module represents the %s module
type Module struct {
	app *core.Application
}

// New creates a new %s module
func New(app *core.Application) *Module {
	return &Module{
		app: app,
	}
}

// RegisterRoutes registers all module routes
func (m *Module) RegisterRoutes() {
`, mg.ModuleName, mg.ModuleName, mg.ModuleName)

	if moduleType == "api" || moduleType == "full" {
		content += fmt.Sprintf(`
	// API routes
	api := m.app.Group("/api/v1/%s")
	m.registerAPIRoutes(api)
`, mg.ModuleName)
	}

	if moduleType == "web" || moduleType == "full" {
		content += fmt.Sprintf(`
	// Web routes
	web := m.app.Group("/%s")
	m.registerWebRoutes(web)
`, mg.ModuleName)
	}

	content += `}

// GetName returns the module name
func (m *Module) GetName() string {
	return "` + mg.ModuleName + `"
}
`

	if moduleType == "api" || moduleType == "full" {
		content += `
// registerAPIRoutes registers API routes
func (m *Module) registerAPIRoutes(api fiber.Router) {
	// TODO: Register API routes here
	// Example:
	// api.Get("/", m.List)
	// api.Post("/", m.Create)
	// api.Get("/:id", m.Show)
	// api.Put("/:id", m.Update)
	// api.Delete("/:id", m.Delete)
}
`
	}

	if moduleType == "web" || moduleType == "full" {
		content += `
// registerWebRoutes registers web routes
func (m *Module) registerWebRoutes(web fiber.Router) {
	// TODO: Register web routes here
	// Example:
	// web.Get("/", m.Index)
	// web.Get("/create", m.CreateForm)
	// web.Post("/", m.Store)
	// web.Get("/:id", m.Show)
	// web.Get("/:id/edit", m.EditForm)
	// web.Put("/:id", m.Update)
	// web.Delete("/:id", m.Delete)
}
`
	}

	return os.WriteFile(filepath.Join(moduleDir, "module.go"), []byte(content), 0644)
}

// generateServiceGo generates the service file
func (mg *SimpleModuleGenerator) generateServiceGo(moduleDir string) error {
	content := fmt.Sprintf(`package %s

import (
	"gorm.io/gorm"
)

// Service handles business logic for %s
type Service struct {
	db *gorm.DB
}

// NewService creates a new service instance
func NewService(db *gorm.DB) *Service {
	return &Service{
		db: db,
	}
}

// TODO: Implement service methods here
// Example:
// func (s *Service) Create(data CreateRequest) (*Model, error) { ... }
// func (s *Service) GetByID(id string) (*Model, error) { ... }
// func (s *Service) Update(id string, data UpdateRequest) (*Model, error) { ... }
// func (s *Service) Delete(id string) error { ... }
// func (s *Service) List(filters ListFilters) ([]*Model, int64, error) { ... }
`, mg.ModuleName, mg.ModuleName)

	return os.WriteFile(filepath.Join(moduleDir, "services", "service.go"), []byte(content), 0644)
}

// generateModelGo generates the model file
func (mg *SimpleModuleGenerator) generateModelGo(moduleDir string) error {
	modelName := strings.Title(mg.ModuleName)
	tableName := mg.ModuleName + "s"

	content := fmt.Sprintf(`package %s

import (
	"time"
	"gorm.io/gorm"
)

// %s represents a %s model
type %s struct {
	ID        string         `+"`"+`gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`+"`"+`
	Name      string         `+"`"+`gorm:"not null" json:"name" validate:"required,min=2,max=100"`+"`"+`
	CreatedAt time.Time      `+"`"+`json:"created_at"`+"`"+`
	UpdatedAt time.Time      `+"`"+`json:"updated_at"`+"`"+`
	DeletedAt gorm.DeletedAt `+"`"+`gorm:"index" json:"-"`+"`"+`
}

// TableName returns the table name for %s
func (%s) TableName() string {
	return "%s"
}

// BeforeCreate is called before creating a record
func (m *%s) BeforeCreate(tx *gorm.DB) error {
	// TODO: Add any pre-creation logic here
	return nil
}

// BeforeUpdate is called before updating a record
func (m *%s) BeforeUpdate(tx *gorm.DB) error {
	// TODO: Add any pre-update logic here
	return nil
}

// BeforeDelete is called before deleting a record
func (m *%s) BeforeDelete(tx *gorm.DB) error {
	// TODO: Add any pre-deletion logic here
	return nil
}
`, mg.ModuleName, modelName, mg.ModuleName, modelName, modelName, modelName, tableName, modelName, modelName, modelName)

	return os.WriteFile(filepath.Join(moduleDir, "models", "model.go"), []byte(content), 0644)
}

// generateREADME generates a README file for the module
func (mg *SimpleModuleGenerator) generateREADME(moduleDir, moduleType string) error {
	content := fmt.Sprintf(`# %s Module

This module provides %s functionality for %s.

## Generated Structure

- controllers/          # Request handlers
- models/              # Database models
- schemas/             # Request/response schemas
- services/            # Business logic
- middleware/          # Module-specific middleware
- module.go           # Main module file
- README.md           # This file

## Features

`, strings.Title(mg.ModuleName), moduleType, mg.ModuleName)

	if moduleType == "api" || moduleType == "full" {
		content += fmt.Sprintf(`### API Endpoints

- GET /api/v1/%s - List %s
- POST /api/v1/%s - Create %s
- GET /api/v1/%s/:id - Show %s
- PUT /api/v1/%s/:id - Update %s
- DELETE /api/v1/%s/:id - Delete %s

`, mg.ModuleName, mg.ModuleName, mg.ModuleName, mg.ModuleName, mg.ModuleName, mg.ModuleName, mg.ModuleName, mg.ModuleName, mg.ModuleName, mg.ModuleName)
	}

	if moduleType == "web" || moduleType == "full" {
		content += fmt.Sprintf(`### Web Routes

- GET /%s - List %s
- GET /%s/create - Create form
- POST /%s - Store %s
- GET /%s/:id - Show %s
- GET /%s/:id/edit - Edit form
- PUT /%s/:id - Update %s
- DELETE /%s/:id - Delete %s

`, mg.ModuleName, mg.ModuleName, mg.ModuleName, mg.ModuleName, mg.ModuleName, mg.ModuleName, mg.ModuleName, mg.ModuleName, mg.ModuleName, mg.ModuleName, mg.ModuleName, mg.ModuleName)
	}

	content += fmt.Sprintf(`## Usage

1. Register the module in your main.go:
`+"```go"+`
import "your-project/app/modules/%s"

// In your main function
%sModule := %s.New(app)
%sModule.RegisterRoutes()
`+"```"+`

2. Run migrations if models were created:
`+"```bash"+`
./artisan migrate
`+"```"+`

3. Update routes in routes/api.go or routes/web.go if needed.

## TODO

- [ ] Implement service methods
- [ ] Add validation
- [ ] Add error handling
- [ ] Add tests
- [ ] Add documentation
- [ ] Customize templates (for web routes)
`, mg.ModuleName, mg.ModuleName, mg.ModuleName, mg.ModuleName)

	return os.WriteFile(filepath.Join(moduleDir, "README.md"), []byte(content), 0644)
}
