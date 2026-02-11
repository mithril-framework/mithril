package graphql

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// GraphQLIntegration handles GraphQL integration with Mithril
type GraphQLIntegration struct {
	config           *GraphQLConfig
	schemaGenerator  *SchemaGenerator
	resolverRegistry *ResolverRegistry
	server           *SimpleGraphQLServer
}

// NewGraphQLIntegration creates a new GraphQL integration
func NewGraphQLIntegration(config *GraphQLConfig) *GraphQLIntegration {
	if config == nil {
		config = DefaultGraphQLConfig()
	}

	return &GraphQLIntegration{
		config:           config,
		schemaGenerator:  NewSchemaGenerator(config),
		resolverRegistry: NewResolverRegistry(),
	}
}

// AddModel adds a model to the GraphQL schema
func (gi *GraphQLIntegration) AddModel(model interface{}) error {
	return gi.schemaGenerator.AddModel(model)
}

// RegisterResolver registers a resolver for a model
func (gi *GraphQLIntegration) RegisterResolver(modelName string, resolver Resolver) {
	// For now, just register the resolver without type checking
	// In a real implementation, you would maintain a registry of model types
	baseResolver := &BaseResolver{
		resolver: resolver,
	}
	gi.resolverRegistry.RegisterResolver(modelName, baseResolver)
}

// GenerateSchema generates the GraphQL schema
func (gi *GraphQLIntegration) GenerateSchema() (string, error) {
	return gi.schemaGenerator.GenerateSchema()
}

// GenerateSchemaFile generates the GraphQL schema and saves it to a file
func (gi *GraphQLIntegration) GenerateSchemaFile(filename string) error {
	schema, err := gi.GenerateSchema()
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write schema to file
	return os.WriteFile(filename, []byte(schema), 0644)
}

// GenerateGQLGenConfig generates gqlgen configuration
func (gi *GraphQLIntegration) GenerateGQLGenConfig(outputDir string) error {
	config := gi.buildGQLGenConfig(outputDir)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	// Write config to file
	configFile := filepath.Join(outputDir, "gqlgen.yml")
	return os.WriteFile(configFile, []byte(config), 0644)
}

// buildGQLGenConfig builds the gqlgen configuration
func (gi *GraphQLIntegration) buildGQLGenConfig(outputDir string) string {
	var config strings.Builder

	config.WriteString("schema:\n")
	config.WriteString("  - \"schema.graphql\"\n\n")

	config.WriteString("exec:\n")
	config.WriteString("  filename: generated.go\n")
	config.WriteString("  package: graphql\n\n")

	config.WriteString("model:\n")
	config.WriteString("  filename: models_gen.go\n")
	config.WriteString("  package: graphql\n\n")

	config.WriteString("resolver:\n")
	config.WriteString("  layout: follow-schema\n")
	config.WriteString("  dir: resolvers\n")
	config.WriteString("  package: resolvers\n\n")

	config.WriteString("autobind:\n")
	config.WriteString("  - \"github.com/mithril-framework/mithril/app/models\"\n\n")

	config.WriteString("models:\n")
	for modelName := range gi.schemaGenerator.GetModels() {
		config.WriteString(fmt.Sprintf("  %s:\n", modelName))
		config.WriteString("    model:\n")
		config.WriteString(fmt.Sprintf("      - github.com/mithril-framework/mithril/app/models.%s\n", modelName))
		config.WriteString("\n")
	}

	return config.String()
}

// SetupServer sets up the GraphQL server
func (gi *GraphQLIntegration) SetupServer() error {
	// Generate schema
	schema, err := gi.GenerateSchema()
	if err != nil {
		return err
	}

	gi.server = NewSimpleGraphQLServer(gi.config, schema)

	return nil
}

// RegisterRoutes registers GraphQL routes with Fiber
func (gi *GraphQLIntegration) RegisterRoutes(app *fiber.App) error {
	if gi.server == nil {
		if err := gi.SetupServer(); err != nil {
			return err
		}
	}

	gi.server.RegisterRoutes(app)
	return nil
}

// GetServer returns the GraphQL server
func (gi *GraphQLIntegration) GetServer() *SimpleGraphQLServer {
	return gi.server
}

// GetConfig returns the GraphQL configuration
func (gi *GraphQLIntegration) GetConfig() *GraphQLConfig {
	return gi.config
}

// GetSchemaGenerator returns the schema generator
func (gi *GraphQLIntegration) GetSchemaGenerator() *SchemaGenerator {
	return gi.schemaGenerator
}

// GetResolverRegistry returns the resolver registry
func (gi *GraphQLIntegration) GetResolverRegistry() *ResolverRegistry {
	return gi.resolverRegistry
}

// Removed unused method getModelType

// SimpleGraphQLIntegration provides a simplified GraphQL integration
type SimpleGraphQLIntegration struct {
	config          *GraphQLConfig
	schemaGenerator *SchemaGenerator
	server          *SimpleGraphQLServer
}

// NewSimpleGraphQLIntegration creates a new simple GraphQL integration
func NewSimpleGraphQLIntegration(config *GraphQLConfig) *SimpleGraphQLIntegration {
	if config == nil {
		config = DefaultGraphQLConfig()
	}

	return &SimpleGraphQLIntegration{
		config:          config,
		schemaGenerator: NewSchemaGenerator(config),
	}
}

// AddModel adds a model to the GraphQL schema
func (sgi *SimpleGraphQLIntegration) AddModel(model interface{}) error {
	return sgi.schemaGenerator.AddModel(model)
}

// GenerateSchema generates the GraphQL schema
func (sgi *SimpleGraphQLIntegration) GenerateSchema() (string, error) {
	return sgi.schemaGenerator.GenerateSchema()
}

// SetupServer sets up the GraphQL server
func (sgi *SimpleGraphQLIntegration) SetupServer() error {
	// Generate schema
	schema, err := sgi.GenerateSchema()
	if err != nil {
		return err
	}

	sgi.server = NewSimpleGraphQLServer(sgi.config, schema)

	return nil
}

// RegisterRoutes registers GraphQL routes with Fiber
func (sgi *SimpleGraphQLIntegration) RegisterRoutes(app *fiber.App) error {
	if sgi.server == nil {
		if err := sgi.SetupServer(); err != nil {
			return err
		}
	}

	sgi.server.RegisterRoutes(app)
	return nil
}

// GetServer returns the GraphQL server
func (sgi *SimpleGraphQLIntegration) GetServer() *SimpleGraphQLServer {
	return sgi.server
}

// GetConfig returns the GraphQL configuration
func (sgi *SimpleGraphQLIntegration) GetConfig() *GraphQLConfig {
	return sgi.config
}

// AddModelFromStruct adds a model from a struct type
func (gi *GraphQLIntegration) AddModelFromStruct(model interface{}) error {
	return gi.AddModel(model)
}

// AddModelFromName adds a model by name (for dynamic model loading)
func (gi *GraphQLIntegration) AddModelFromName(modelName string, modelType interface{}) error {
	// Register the model type
	gi.registerModelType(modelName, modelType)

	// Add to schema generator
	return gi.schemaGenerator.AddModel(modelType)
}

// registerModelType registers a model type
func (gi *GraphQLIntegration) registerModelType(modelName string, modelType interface{}) {
	// This is a simplified implementation
	// In a real implementation, you would maintain a registry of model types
}

// GenerateCompleteGraphQL generates complete GraphQL setup
func (gi *GraphQLIntegration) GenerateCompleteGraphQL(outputDir string) error {
	// Generate schema file
	schemaFile := filepath.Join(outputDir, "schema.graphql")
	if err := gi.GenerateSchemaFile(schemaFile); err != nil {
		return fmt.Errorf("failed to generate schema file: %v", err)
	}

	// Generate gqlgen config
	if err := gi.GenerateGQLGenConfig(outputDir); err != nil {
		return fmt.Errorf("failed to generate gqlgen config: %v", err)
	}

	// Generate resolvers directory structure
	resolversDir := filepath.Join(outputDir, "resolvers")
	if err := os.MkdirAll(resolversDir, 0755); err != nil {
		return fmt.Errorf("failed to create resolvers directory: %v", err)
	}

	// Generate base resolver
	baseResolverFile := filepath.Join(resolversDir, "resolver.go")
	if err := gi.generateBaseResolver(baseResolverFile); err != nil {
		return fmt.Errorf("failed to generate base resolver: %v", err)
	}

	return nil
}

// generateBaseResolver generates a base resolver file
func (gi *GraphQLIntegration) generateBaseResolver(filename string) error {
	var resolver strings.Builder

	resolver.WriteString("package resolvers\n\n")
	resolver.WriteString("import (\n")
	resolver.WriteString("\t\"context\"\n")
	resolver.WriteString("\t\"github.com/99designs/gqlgen/graphql\"\n")
	resolver.WriteString(")\n\n")

	resolver.WriteString("// This file will not be overwritten automatically.\n")
	resolver.WriteString("// It serves as the base for your GraphQL resolvers.\n\n")

	resolver.WriteString("// Resolver is the root resolver\n")
	resolver.WriteString("type Resolver struct{}\n\n")

	// Generate resolver methods for each model
	for modelName := range gi.schemaGenerator.GetModels() {

		// Query resolvers
		resolver.WriteString(fmt.Sprintf("// %s returns a single %s by ID\n", modelName, modelName))
		resolver.WriteString(fmt.Sprintf("func (r *Resolver) %s(ctx context.Context, id string) (*%s, error) {\n", modelName, modelName))
		resolver.WriteString("\t// TODO: Implement resolver\n")
		resolver.WriteString("\treturn nil, fmt.Errorf(\"not implemented\")\n")
		resolver.WriteString("}\n\n")

		// List resolver
		resolver.WriteString(fmt.Sprintf("// %ss returns a list of %ss\n", modelName, modelName))
		resolver.WriteString(fmt.Sprintf("func (r *Resolver) %ss(ctx context.Context) ([]*%s, error) {\n", modelName, modelName))
		resolver.WriteString("\t// TODO: Implement resolver\n")
		resolver.WriteString(fmt.Sprintf("\treturn []*%s{}, nil\n", modelName))
		resolver.WriteString("}\n\n")

		// Mutation resolvers
		resolver.WriteString(fmt.Sprintf("// Create%s creates a new %s\n", modelName, modelName))
		resolver.WriteString(fmt.Sprintf("func (r *Resolver) Create%s(ctx context.Context, input %sInput) (*%s, error) {\n", modelName, modelName, modelName))
		resolver.WriteString("\t// TODO: Implement resolver\n")
		resolver.WriteString("\treturn nil, fmt.Errorf(\"not implemented\")\n")
		resolver.WriteString("}\n\n")

		resolver.WriteString(fmt.Sprintf("// Update%s updates an existing %s\n", modelName, modelName))
		resolver.WriteString(fmt.Sprintf("func (r *Resolver) Update%s(ctx context.Context, id string, input %sInput) (*%s, error) {\n", modelName, modelName, modelName))
		resolver.WriteString("\t// TODO: Implement resolver\n")
		resolver.WriteString("\treturn nil, fmt.Errorf(\"not implemented\")\n")
		resolver.WriteString("}\n\n")

	resolver.WriteString(fmt.Sprintf("// Delete%s deletes a %s\n", modelName, modelName))
	resolver.WriteString(fmt.Sprintf("func (r *Resolver) Delete%s(ctx context.Context, id string) (bool, error) {\n", modelName))
		resolver.WriteString("\t// TODO: Implement resolver\n")
		resolver.WriteString("\treturn false, fmt.Errorf(\"not implemented\")\n")
		resolver.WriteString("}\n\n")
	}

	return os.WriteFile(filename, []byte(resolver.String()), 0644)
}
