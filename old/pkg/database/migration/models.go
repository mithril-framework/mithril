package migration

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/gorm"
)

// ModelRegistry is a global registry for models that should be migrated
// Models can register themselves using RegisterModel() in their init() functions
var globalRegistry = NewModelRegistry()

// ModelRegistry is a registry for models that should be migrated
type ModelRegistry struct {
	models []interface{}
}

// NewModelRegistry creates a new model registry
func NewModelRegistry() *ModelRegistry {
	return &ModelRegistry{
		models: make([]interface{}, 0),
	}
}

// Register adds a model to the registry
func (r *ModelRegistry) Register(model interface{}) {
	r.models = append(r.models, model)
}

// GetModels returns all registered models
func (r *ModelRegistry) GetModels() []interface{} {
	return r.models
}

// AutoMigrate runs AutoMigrate on all registered models
func (r *ModelRegistry) AutoMigrate(db *gorm.DB) error {
	if len(r.models) == 0 {
		return fmt.Errorf("no models registered")
	}

	return db.AutoMigrate(r.models...)
}

// RegisterModel registers a model with the global registry
// This should be called in init() functions of model files
func RegisterModel(model interface{}) {
	globalRegistry.Register(model)
}

// GetRegisteredModels returns all models registered with the global registry
func GetRegisteredModels() []interface{} {
	return globalRegistry.GetModels()
}

// AutoMigrateRegistered runs AutoMigrate on all registered models
func AutoMigrateRegistered(db *gorm.DB) error {
	return globalRegistry.AutoMigrate(db)
}

// DiscoverModelNames discovers model type names from the app/models directory
// by parsing Go files. Returns a list of exported struct names that have gorm tags.
// This is useful for documentation and validation, but actual model instances
// must be provided via RegisterModel() in init() functions.
func DiscoverModelNames(modelsPath string) ([]string, error) {
	if modelsPath == "" {
		modelsPath = "app/models"
	}

	// Check if directory exists
	if _, err := os.Stat(modelsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("models directory not found: %s", modelsPath)
	}

	var modelNames []string
	seen := make(map[string]bool)

	// Read all .go files in the models directory
	err := filepath.Walk(modelsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip test files
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Parse the Go file
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			// Skip files that can't be parsed (might be build tags, etc.)
			return nil
		}

		// Find struct types that look like GORM models
		ast.Inspect(node, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}

			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				return true
			}

			// Check if struct has gorm tags (indicating it's a model)
			hasGormTags := false
			for _, field := range st.Fields.List {
				if field.Tag != nil {
					tag := field.Tag.Value
					if strings.Contains(tag, "gorm:") {
						hasGormTags = true
						break
					}
				}
			}

			// If it has gorm tags and is exported, it's likely a model
			if hasGormTags && ts.Name != nil && ts.Name.IsExported() {
				typeName := ts.Name.Name
				if !seen[typeName] {
					modelNames = append(modelNames, typeName)
					seen[typeName] = true
				}
			}

			return true
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk models directory: %w", err)
	}

	return modelNames, nil
}

