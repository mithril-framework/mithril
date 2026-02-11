package validation

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Tag     string
	Value   interface{}
	Message string
}

// Error implements the error interface
func (e ValidationError) Error() string {
	return e.Message
}

// Validator holds the validation instance
type Validator struct {
	rules map[string]func(interface{}) error
}

// NewValidator creates a new validator instance
func NewValidator() *Validator {
	v := &Validator{
		rules: make(map[string]func(interface{}) error),
	}

	// Register built-in validators
	v.registerBuiltInValidators()

	return v
}

// registerBuiltInValidators registers built-in validation rules
func (v *Validator) registerBuiltInValidators() {
	v.rules["required"] = func(value interface{}) error {
		if value == nil || value == "" {
			return fmt.Errorf("is required")
		}
		return nil
	}

	v.rules["email"] = func(value interface{}) error {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("must be a string")
		}
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(str) {
			return fmt.Errorf("must be a valid email address")
		}
		return nil
	}

	v.rules["min"] = func(value interface{}) error {
		// This is a simplified version - in practice, you'd parse the tag parameter
		return nil
	}

	v.rules["max"] = func(value interface{}) error {
		// This is a simplified version - in practice, you'd parse the tag parameter
		return nil
	}

	v.rules["uuid"] = func(value interface{}) error {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("must be a string")
		}
		uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
		if !uuidRegex.MatchString(strings.ToLower(str)) {
			return fmt.Errorf("must be a valid UUID")
		}
		return nil
	}

	v.rules["phone"] = func(value interface{}) error {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("must be a string")
		}
		// Remove all non-digit characters
		digits := regexp.MustCompile(`\D`).ReplaceAllString(str, "")
		if len(digits) < 10 || len(digits) > 15 {
			return fmt.Errorf("must be a valid phone number")
		}
		return nil
	}

	v.rules["password"] = func(value interface{}) error {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("must be a string")
		}
		if len(str) < 8 {
			return fmt.Errorf("must be at least 8 characters long")
		}

		var hasUpper, hasLower, hasNumber, hasSpecial bool
		for _, char := range str {
			switch {
			case char >= 'A' && char <= 'Z':
				hasUpper = true
			case char >= 'a' && char <= 'z':
				hasLower = true
			case char >= '0' && char <= '9':
				hasNumber = true
			case char >= 33 && char <= 126 && (char < 48 || char > 57) && (char < 65 || char > 90) && (char < 97 || char > 122):
				hasSpecial = true
			}
		}

		if !hasUpper || !hasLower || !hasNumber || !hasSpecial {
			return fmt.Errorf("must contain uppercase, lowercase, number and special character")
		}
		return nil
	}
}

// ValidateStruct validates a struct using struct tags
func (v *Validator) ValidateStruct(s interface{}) error {
	val := reflect.ValueOf(s)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, got %s", val.Kind())
	}

	typ := val.Type()
	var errors []string

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}

		validateTag := fieldType.Tag.Get("validate")
		if validateTag == "" {
			continue
		}

		fieldName := fieldType.Name
		if jsonTag := fieldType.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
			fieldName = strings.Split(jsonTag, ",")[0]
		}

		// Parse validation tags
		tags := strings.Split(validateTag, ",")
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}

			// Handle tag with parameters (e.g., "min=5")
			tagName := tag
			if strings.Contains(tag, "=") {
				tagName = strings.Split(tag, "=")[0]
			}

			if validator, exists := v.rules[tagName]; exists {
				if err := validator(field.Interface()); err != nil {
					errors = append(errors, fmt.Sprintf("%s %s", fieldName, err.Error()))
				}
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}

	return nil
}

// ValidateRequest validates a request body and binds it to a struct
func (v *Validator) ValidateRequest(c *fiber.Ctx, dest interface{}) error {
	// Parse JSON body
	if err := c.BodyParser(dest); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid JSON format")
	}

	// Validate struct
	if err := v.ValidateStruct(dest); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
	}

	return nil
}

// ValidateQuery validates query parameters
func (v *Validator) ValidateQuery(c *fiber.Ctx, dest interface{}) error {
	// Parse query parameters
	if err := c.QueryParser(dest); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid query parameters")
	}

	// Validate struct
	if err := v.ValidateStruct(dest); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
	}

	return nil
}

// Schema represents a validation schema with metadata
type Schema struct {
	Type        string                 `json:"type"`
	Properties  map[string]interface{} `json:"properties"`
	Required    []string               `json:"required,omitempty"`
	Example     interface{}            `json:"example,omitempty"`
	Description string                 `json:"description,omitempty"`
}

// GenerateSchema generates a JSON schema from a Go struct
func (v *Validator) GenerateSchema(s interface{}) (*Schema, error) {
	t := reflect.TypeOf(s)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %s", t.Kind())
	}

	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]interface{}),
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		fieldName := strings.Split(jsonTag, ",")[0]
		fieldType := field.Type

		// Handle pointers
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}

		// Generate field schema
		fieldSchema := v.generateFieldSchema(fieldType, field)
		schema.Properties[fieldName] = fieldSchema

		// Check if required
		validateTag := field.Tag.Get("validate")
		if strings.Contains(validateTag, "required") {
			schema.Required = append(schema.Required, fieldName)
		}
	}

	return schema, nil
}

// generateFieldSchema generates schema for a field
func (v *Validator) generateFieldSchema(fieldType reflect.Type, field reflect.StructField) map[string]interface{} {
	schema := make(map[string]interface{})

	// Set type
	switch fieldType.Kind() {
	case reflect.String:
		schema["type"] = "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema["type"] = "integer"
	case reflect.Float32, reflect.Float64:
		schema["type"] = "number"
	case reflect.Bool:
		schema["type"] = "boolean"
	case reflect.Slice, reflect.Array:
		schema["type"] = "array"
		if fieldType.Elem().Kind() == reflect.String {
			schema["items"] = map[string]string{"type": "string"}
		}
	default:
		schema["type"] = "string"
	}

	// Add validation constraints
	validateTag := field.Tag.Get("validate")
	if validateTag != "" {
		constraints := v.parseValidationConstraints(validateTag)
		for key, value := range constraints {
			schema[key] = value
		}
	}

	// Add example from struct tag
	if example := field.Tag.Get("example"); example != "" {
		schema["example"] = example
	}

	// Add description from struct tag
	if description := field.Tag.Get("description"); description != "" {
		schema["description"] = description
	}

	return schema
}

// parseValidationConstraints parses validation tags into OpenAPI constraints
func (v *Validator) parseValidationConstraints(validateTag string) map[string]interface{} {
	constraints := make(map[string]interface{})

	parts := strings.Split(validateTag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)

		if strings.HasPrefix(part, "min=") {
			constraints["minLength"] = part[4:]
		} else if strings.HasPrefix(part, "max=") {
			constraints["maxLength"] = part[4:]
		} else if strings.HasPrefix(part, "len=") {
			constraints["minLength"] = part[4:]
			constraints["maxLength"] = part[4:]
		} else if part == "email" {
			constraints["format"] = "email"
		} else if part == "uuid" {
			constraints["format"] = "uuid"
		} else if part == "phone" {
			constraints["pattern"] = "^[+]?[0-9\\-\\s\\(\\)]{10,15}$"
		}
	}

	return constraints
}

// Middleware function for request validation
func ValidateRequest[T any](validator *Validator) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var request T

		if err := validator.ValidateRequest(c, &request); err != nil {
			return err
		}

		// Store validated data in context
		c.Locals("validated_data", request)

		return c.Next()
	}
}

// GetValidatedData retrieves validated data from context
func GetValidatedData[T any](c *fiber.Ctx) (T, error) {
	var data T

	if validatedData := c.Locals("validated_data"); validatedData != nil {
		if data, ok := validatedData.(T); ok {
			return data, nil
		}
	}

	return data, fiber.NewError(fiber.StatusInternalServerError, "No validated data found")
}
