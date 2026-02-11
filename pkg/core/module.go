package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// Module represents a Mithril module
type Module interface {
	RegisterRoutes()
	GetName() string
}

// ModuleManager manages all registered modules
type ModuleManager struct {
	modules map[string]Module
	app     *Application
}

// NewModuleManager creates a new module manager
func NewModuleManager(app *Application) *ModuleManager {
	return &ModuleManager{
		modules: make(map[string]Module),
		app:     app,
	}
}

// RegisterModule registers a module
func (mm *ModuleManager) RegisterModule(module Module) error {
	name := module.GetName()
	if _, exists := mm.modules[name]; exists {
		return fmt.Errorf("module '%s' is already registered", name)
	}

	mm.modules[name] = module
	module.RegisterRoutes()

	return nil
}

// GetModule returns a registered module by name
func (mm *ModuleManager) GetModule(name string) (Module, bool) {
	module, exists := mm.modules[name]
	return module, exists
}

// ListModules returns all registered modules
func (mm *ModuleManager) ListModules() map[string]Module {
	return mm.modules
}

// AutoLoadModules automatically loads modules from the app/modules directory
func (mm *ModuleManager) AutoLoadModules() error {
	modulesDir := "app/modules"

	// Check if modules directory exists
	if !mm.directoryExists(modulesDir) {
		return nil // No modules directory, nothing to load
	}

	// Walk through modules directory
	return filepath.Walk(modulesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a directory or if it's the modules directory itself
		if !info.IsDir() || path == modulesDir {
			return nil
		}

		// Get module name from directory name
		moduleName := filepath.Base(path)

		// Skip hidden directories
		if strings.HasPrefix(moduleName, ".") {
			return nil
		}

		// Try to load the module
		if err := mm.loadModule(moduleName, path); err != nil {
			fmt.Printf("Warning: Failed to load module '%s': %v\n", moduleName, err)
			// Continue loading other modules even if one fails
		}

		return nil
	})
}

// loadModule loads a specific module
func (mm *ModuleManager) loadModule(name, path string) error {
	// Check if module.go exists
	moduleFile := filepath.Join(path, "module.go")
	if !mm.fileExists(moduleFile) {
		return fmt.Errorf("module.go not found in %s", path)
	}

	// For now, we'll just register a placeholder
	// In a real implementation, you would use reflection or code generation
	// to dynamically load and instantiate the module
	module := &PlaceholderModule{
		name: name,
		path: path,
		app:  mm.app,
	}

	return mm.RegisterModule(module)
}

// directoryExists checks if a directory exists
func (mm *ModuleManager) directoryExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// fileExists checks if a file exists
func (mm *ModuleManager) fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// PlaceholderModule is a placeholder for dynamically loaded modules
type PlaceholderModule struct {
	name string
	path string
	app  *Application
}

// RegisterRoutes registers routes for the placeholder module
func (pm *PlaceholderModule) RegisterRoutes() {
	// Register a placeholder route
	pm.app.Get("/modules/"+pm.name, func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": fmt.Sprintf("Module '%s' is loaded but not fully implemented", pm.name),
			"path":    pm.path,
		})
	})
}

// GetName returns the module name
func (pm *PlaceholderModule) GetName() string {
	return pm.name
}

// ModuleRegistry provides a simple way to register modules
type ModuleRegistry struct {
	manager *ModuleManager
}

// NewModuleRegistry creates a new module registry
func NewModuleRegistry(manager *ModuleManager) *ModuleRegistry {
	return &ModuleRegistry{
		manager: manager,
	}
}

// Register registers a module
func (mr *ModuleRegistry) Register(module Module) error {
	return mr.manager.RegisterModule(module)
}

// RegisterAll registers multiple modules
func (mr *ModuleRegistry) RegisterAll(modules ...Module) error {
	for _, module := range modules {
		if err := mr.Register(module); err != nil {
			return err
		}
	}
	return nil
}
