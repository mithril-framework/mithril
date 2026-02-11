package graphql

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/google/uuid"
)

// Resolver interface defines methods for GraphQL resolvers
type Resolver interface {
	// Query resolvers
	GetByID(ctx context.Context, id string) (interface{}, error)
	List(ctx context.Context, limit, offset int) ([]interface{}, error)

	// Mutation resolvers
	Create(ctx context.Context, input map[string]interface{}) (interface{}, error)
	Update(ctx context.Context, id string, input map[string]interface{}) (interface{}, error)
	Delete(ctx context.Context, id string) (bool, error)
}

// BaseResolver provides a base implementation for resolvers
type BaseResolver struct {
	modelType reflect.Type
	resolver  Resolver
}

// NewBaseResolver creates a new base resolver
func NewBaseResolver(modelType reflect.Type, resolver Resolver) *BaseResolver {
	return &BaseResolver{
		modelType: modelType,
		resolver:  resolver,
	}
}

// GetByID resolves single item queries
func (br *BaseResolver) GetByID(ctx context.Context, id string) (interface{}, error) {
	if br.resolver != nil {
		return br.resolver.GetByID(ctx, id)
	}

	// Default implementation - return error
	return nil, fmt.Errorf("resolver not implemented for %s", br.modelType.Name())
}

// List resolves list queries
func (br *BaseResolver) List(ctx context.Context, limit, offset int) ([]interface{}, error) {
	if br.resolver != nil {
		return br.resolver.List(ctx, limit, offset)
	}

	// Default implementation - return empty list
	return []interface{}{}, nil
}

// Create resolves create mutations
func (br *BaseResolver) Create(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	if br.resolver != nil {
		return br.resolver.Create(ctx, input)
	}

	// Default implementation - return error
	return nil, fmt.Errorf("create resolver not implemented for %s", br.modelType.Name())
}

// Update resolves update mutations
func (br *BaseResolver) Update(ctx context.Context, id string, input map[string]interface{}) (interface{}, error) {
	if br.resolver != nil {
		return br.resolver.Update(ctx, id, input)
	}

	// Default implementation - return error
	return nil, fmt.Errorf("update resolver not implemented for %s", br.modelType.Name())
}

// Delete resolves delete mutations
func (br *BaseResolver) Delete(ctx context.Context, id string) (bool, error) {
	if br.resolver != nil {
		return br.resolver.Delete(ctx, id)
	}

	// Default implementation - return error
	return false, fmt.Errorf("delete resolver not implemented for %s", br.modelType.Name())
}

// ResolverRegistry manages GraphQL resolvers
type ResolverRegistry struct {
	resolvers map[string]*BaseResolver
}

// NewResolverRegistry creates a new resolver registry
func NewResolverRegistry() *ResolverRegistry {
	return &ResolverRegistry{
		resolvers: make(map[string]*BaseResolver),
	}
}

// RegisterResolver registers a resolver for a model
func (rr *ResolverRegistry) RegisterResolver(modelName string, resolver *BaseResolver) {
	rr.resolvers[modelName] = resolver
}

// GetResolver returns a resolver for a model
func (rr *ResolverRegistry) GetResolver(modelName string) (*BaseResolver, bool) {
	resolver, exists := rr.resolvers[modelName]
	return resolver, exists
}

// GetAllResolvers returns all registered resolvers
func (rr *ResolverRegistry) GetAllResolvers() map[string]*BaseResolver {
	return rr.resolvers
}

// ModelToMap converts a model to a map for GraphQL
func ModelToMap(model interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	modelValue := reflect.ValueOf(model)
	if modelValue.Kind() == reflect.Ptr {
		modelValue = modelValue.Elem()
	}

	if modelValue.Kind() != reflect.Struct {
		return result
	}

	modelType := modelValue.Type()

	for i := 0; i < modelValue.NumField(); i++ {
		field := modelType.Field(i)
		fieldValue := modelValue.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get field name from JSON tag or use field name
		fieldName := getFieldName(field)

		// Convert field value to interface{}
		var value interface{}
		if fieldValue.IsValid() && fieldValue.CanInterface() {
			value = fieldValue.Interface()
		}

		// Handle special types
		value = convertSpecialTypes(value)

		result[fieldName] = value
	}

	return result
}

// MapToModel converts a map to a model
func MapToModel(data map[string]interface{}, modelType reflect.Type) (interface{}, error) {
	// Create new instance of the model
	modelValue := reflect.New(modelType)
	modelElem := modelValue.Elem()

	// Set fields from map
	for key, value := range data {
		field := findFieldByName(modelType, key)
		if field == nil {
			continue
		}

		fieldValue := modelElem.FieldByIndex(field.Index)
		if !fieldValue.CanSet() {
			continue
		}

		// Convert value to field type
		convertedValue, err := convertValue(value, field.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to convert field %s: %v", key, err)
		}

		fieldValue.Set(convertedValue)
	}

	return modelValue.Interface(), nil
}

// getFieldName gets the field name from JSON tag or field name
func getFieldName(field reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	if jsonTag != "" && jsonTag != "-" {
		// Remove omitempty and other options
		parts := strings.Split(jsonTag, ",")
		return parts[0]
	}

	// Convert to snake_case
	return toSnakeCase(field.Name)
}

// findFieldByName finds a field by name (case-insensitive)
func findFieldByName(modelType reflect.Type, name string) *reflect.StructField {
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)

		// Check field name
		if strings.EqualFold(field.Name, name) {
			return &field
		}

		// Check JSON tag
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" && jsonTag != "-" {
			parts := strings.Split(jsonTag, ",")
			if strings.EqualFold(parts[0], name) {
				return &field
			}
		}

		// Check snake_case version
		if strings.EqualFold(toSnakeCase(field.Name), name) {
			return &field
		}
	}

	return nil
}

// convertSpecialTypes converts special types for GraphQL
func convertSpecialTypes(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case uuid.UUID:
		return v.String()
	case *uuid.UUID:
		if v != nil {
			return v.String()
		}
		return nil
	default:
		return value
	}
}

// convertValue converts a value to the target type
func convertValue(value interface{}, targetType reflect.Type) (reflect.Value, error) {
	if value == nil {
		return reflect.Zero(targetType), nil
	}

	valueType := reflect.TypeOf(value)

	// If types match, direct conversion
	if valueType.AssignableTo(targetType) {
		return reflect.ValueOf(value), nil
	}

	// Handle string to UUID conversion
	if targetType.String() == "uuid.UUID" {
		if str, ok := value.(string); ok {
			uuidValue, err := uuid.Parse(str)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(uuidValue), nil
		}
	}

	// Handle string to *UUID conversion
	if targetType.String() == "*uuid.UUID" {
		if str, ok := value.(string); ok {
			uuidValue, err := uuid.Parse(str)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(&uuidValue), nil
		}
	}

	// Try direct conversion
	if valueType.ConvertibleTo(targetType) {
		return reflect.ValueOf(value).Convert(targetType), nil
	}

	return reflect.Value{}, fmt.Errorf("cannot convert %T to %s", value, targetType)
}

// toSnakeCase converts a string to snake_case
func toSnakeCase(str string) string {
	var result []rune
	for i, r := range str {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}

// GraphQLContext represents context for GraphQL operations
type GraphQLContext struct {
	UserID    string
	UserRole  string
	RequestID string
	Data      map[string]interface{}
}

// GetUserID returns the user ID from context
func (ctx *GraphQLContext) GetUserID() string {
	return ctx.UserID
}

// GetUserRole returns the user role from context
func (ctx *GraphQLContext) GetUserRole() string {
	return ctx.UserRole
}

// GetRequestID returns the request ID from context
func (ctx *GraphQLContext) GetRequestID() string {
	return ctx.RequestID
}

// GetData returns custom data from context
func (ctx *GraphQLContext) GetData(key string) interface{} {
	if ctx.Data == nil {
		return nil
	}
	return ctx.Data[key]
}

// SetData sets custom data in context
func (ctx *GraphQLContext) SetData(key string, value interface{}) {
	if ctx.Data == nil {
		ctx.Data = make(map[string]interface{})
	}
	ctx.Data[key] = value
}
