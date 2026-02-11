package graphql

import (
	"fmt"
	"reflect"
	"strings"
)

// GraphQLConfig holds configuration for GraphQL
type GraphQLConfig struct {
	Path            string `json:"path"`             // GraphQL endpoint path
	Playground      bool   `json:"playground"`       // Enable GraphQL playground
	Introspection   bool   `json:"introspection"`    // Enable introspection
	ComplexityLimit int    `json:"complexity_limit"` // Query complexity limit
}

// DefaultGraphQLConfig returns the default GraphQL configuration
func DefaultGraphQLConfig() *GraphQLConfig {
	return &GraphQLConfig{
		Path:            "/graphql",
		Playground:      true,
		Introspection:   true,
		ComplexityLimit: 1000,
	}
}

// ModelInfo represents information about a model for GraphQL schema generation
type ModelInfo struct {
	Name       string                 `json:"name"`
	Fields     []FieldInfo            `json:"fields"`
	Relations  []RelationInfo         `json:"relations"`
	Directives map[string]interface{} `json:"directives"`
}

// FieldInfo represents information about a model field
type FieldInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	GraphQLType string `json:"graphql_type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
	Tags        string `json:"tags"`
}

// RelationInfo represents information about model relations
type RelationInfo struct {
	Name       string `json:"name"`
	Type       string `json:"type"`   // "one", "many", "belongs_to", "has_many"
	Target     string `json:"target"` // Target model name
	ForeignKey string `json:"foreign_key"`
}

// SchemaGenerator handles GraphQL schema generation
type SchemaGenerator struct {
	models map[string]ModelInfo
	config *GraphQLConfig
}

// NewSchemaGenerator creates a new SchemaGenerator
func NewSchemaGenerator(config *GraphQLConfig) *SchemaGenerator {
	if config == nil {
		config = DefaultGraphQLConfig()
	}

	return &SchemaGenerator{
		models: make(map[string]ModelInfo),
		config: config,
	}
}

// AddModel adds a model to the schema generator
func (sg *SchemaGenerator) AddModel(model interface{}) error {
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	if modelType.Kind() != reflect.Struct {
		return fmt.Errorf("model must be a struct")
	}

	modelInfo := ModelInfo{
		Name:       modelType.Name(),
		Fields:     []FieldInfo{},
		Relations:  []RelationInfo{},
		Directives: make(map[string]interface{}),
	}

	// Extract fields
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Skip fields with json:"-" tag
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		fieldInfo := FieldInfo{
			Name:        field.Name,
			Type:        field.Type.String(),
			GraphQLType: sg.goTypeToGraphQL(field.Type),
			Required:    sg.isRequired(field),
			Description: field.Tag.Get("description"),
			Tags:        string(field.Tag),
		}

		modelInfo.Fields = append(modelInfo.Fields, fieldInfo)
	}

	sg.models[modelInfo.Name] = modelInfo
	return nil
}

// GenerateSchema generates a GraphQL schema from registered models
func (sg *SchemaGenerator) GenerateSchema() (string, error) {
	var schema strings.Builder

	// Add scalar definitions
	schema.WriteString(sg.generateScalars())
	schema.WriteString("\n\n")

	// Add type definitions
	for _, model := range sg.models {
		schema.WriteString(sg.generateType(model))
		schema.WriteString("\n\n")
	}

	// Add input types
	for _, model := range sg.models {
		schema.WriteString(sg.generateInputType(model))
		schema.WriteString("\n\n")
	}

	// Add query type
	schema.WriteString(sg.generateQueryType())
	schema.WriteString("\n\n")

	// Add mutation type
	schema.WriteString(sg.generateMutationType())
	schema.WriteString("\n\n")

	return schema.String(), nil
}

// generateScalars generates scalar type definitions
func (sg *SchemaGenerator) generateScalars() string {
	return `scalar Time
scalar UUID`
}

// generateType generates a GraphQL type definition
func (sg *SchemaGenerator) generateType(model ModelInfo) string {
	var typeDef strings.Builder

	typeDef.WriteString(fmt.Sprintf("type %s {\n", model.Name))

	for _, field := range model.Fields {
		fieldType := field.GraphQLType
		if !field.Required {
			fieldType = fieldType + "!"
		}

		line := fmt.Sprintf("  %s: %s", strings.ToLower(field.Name), fieldType)
		if field.Description != "" {
			line += fmt.Sprintf(" # %s", field.Description)
		}
		line += "\n"

		typeDef.WriteString(line)
	}

	typeDef.WriteString("}")
	return typeDef.String()
}

// generateInputType generates a GraphQL input type definition
func (sg *SchemaGenerator) generateInputType(model ModelInfo) string {
	var inputDef strings.Builder

	inputDef.WriteString(fmt.Sprintf("input %sInput {\n", model.Name))

	for _, field := range model.Fields {
		// Skip ID fields for input types
		if strings.ToLower(field.Name) == "id" {
			continue
		}

		fieldType := field.GraphQLType
		if field.Required {
			fieldType = fieldType + "!"
		}

		line := fmt.Sprintf("  %s: %s", strings.ToLower(field.Name), fieldType)
		if field.Description != "" {
			line += fmt.Sprintf(" # %s", field.Description)
		}
		line += "\n"

		inputDef.WriteString(line)
	}

	inputDef.WriteString("}")
	return inputDef.String()
}

// generateQueryType generates the Query type
func (sg *SchemaGenerator) generateQueryType() string {
	var queryDef strings.Builder

	queryDef.WriteString("type Query {\n")

	for modelName := range sg.models {
		lowerName := strings.ToLower(modelName)

		// Add single item query
		queryDef.WriteString(fmt.Sprintf("  %s(id: ID!): %s\n", lowerName, modelName))

		// Add list query
		queryDef.WriteString(fmt.Sprintf("  %ss: [%s!]!\n", lowerName, modelName))
	}

	queryDef.WriteString("}")
	return queryDef.String()
}

// generateMutationType generates the Mutation type
func (sg *SchemaGenerator) generateMutationType() string {
	var mutationDef strings.Builder

	mutationDef.WriteString("type Mutation {\n")

	for modelName := range sg.models {
		lowerName := strings.ToLower(modelName)

		// Add create mutation
		mutationDef.WriteString(fmt.Sprintf("  create%s(input: %sInput!): %s!\n", modelName, modelName, modelName))

		// Add update mutation
		mutationDef.WriteString(fmt.Sprintf("  update%s(id: ID!, input: %sInput!): %s!\n", modelName, modelName, modelName))

		// Add delete mutation
		mutationDef.WriteString(fmt.Sprintf("  delete%s(id: ID!): Boolean!\n", lowerName))
	}

	mutationDef.WriteString("}")
	return mutationDef.String()
}

// goTypeToGraphQL converts Go types to GraphQL types
func (sg *SchemaGenerator) goTypeToGraphQL(goType reflect.Type) string {
	// Handle pointers
	if goType.Kind() == reflect.Ptr {
		goType = goType.Elem()
	}

	// Handle slices
	if goType.Kind() == reflect.Slice {
		elementType := sg.goTypeToGraphQL(goType.Elem())
		return "[" + elementType + "]"
	}

	// Handle maps
	if goType.Kind() == reflect.Map {
		return "JSON"
	}

	// Handle basic types
	switch goType.Kind() {
	case reflect.String:
		// Check for special types
		if goType.String() == "time.Time" {
			return "Time"
		}
		if goType.String() == "uuid.UUID" {
			return "UUID"
		}
		return "String"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "Int"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "Int"
	case reflect.Float32, reflect.Float64:
		return "Float"
	case reflect.Bool:
		return "Boolean"
	case reflect.Struct:
		// Check for special struct types
		if goType.String() == "time.Time" {
			return "Time"
		}
		if goType.String() == "uuid.UUID" {
			return "UUID"
		}
		// For other structs, return the type name
		return goType.Name()
	default:
		return "String"
	}
}

// isRequired checks if a field is required based on its tags
func (sg *SchemaGenerator) isRequired(field reflect.StructField) bool {
	// Check for required tag
	validateTag := field.Tag.Get("validate")
	if strings.Contains(validateTag, "required") {
		return true
	}

	// Check for gorm not null tag
	gormTag := field.Tag.Get("gorm")
	if strings.Contains(gormTag, "not null") {
		return true
	}

	// Check for json omitempty tag
	jsonTag := field.Tag.Get("json")
	return !strings.Contains(jsonTag, "omitempty")
}

// GetModels returns all registered models
func (sg *SchemaGenerator) GetModels() map[string]ModelInfo {
	return sg.models
}

// GetModel returns a specific model by name
func (sg *SchemaGenerator) GetModel(name string) (ModelInfo, bool) {
	model, exists := sg.models[name]
	return model, exists
}
