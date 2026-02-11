package swagger

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// SwaggerMiddleware handles automatic Swagger generation
type SwaggerMiddleware struct {
	generator *Generator
	enabled   bool
}

// NewSwaggerMiddleware creates a new Swagger middleware
func NewSwaggerMiddleware(generator *Generator, enabled bool) *SwaggerMiddleware {
	return &SwaggerMiddleware{
		generator: generator,
		enabled:   enabled,
	}
}

// RegisterRoutes registers routes with Swagger
func (sm *SwaggerMiddleware) RegisterRoutes(app *fiber.App) {
	if !sm.enabled {
		return
	}

	// Add Swagger endpoints
	app.Get("/docs", sm.serveSwaggerUI)
	app.Get("/swagger.json", sm.serveSwaggerJSON)
	app.Get("/openapi.json", sm.serveSwaggerJSON)
}

// serveSwaggerUI serves the Swagger UI
func (sm *SwaggerMiddleware) serveSwaggerUI(c *fiber.Ctx) error {
	spec, err := sm.generator.Generate()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to generate Swagger spec",
		})
	}

	html := sm.generateSwaggerHTML(spec)
	c.Set("Content-Type", "text/html")
	return c.SendString(html)
}

// serveSwaggerJSON serves the Swagger JSON
func (sm *SwaggerMiddleware) serveSwaggerJSON(c *fiber.Ctx) error {
	spec, err := sm.generator.Generate()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to generate Swagger spec",
		})
	}

	return c.JSON(spec)
}

// generateSwaggerHTML generates the Swagger UI HTML
func (sm *SwaggerMiddleware) generateSwaggerHTML(spec *OpenAPISpec) string {
	specJSON, _ := json.Marshal(spec)

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>%s API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui.css" />
    <style>
        .swagger-ui .topbar { display: none; }
        .swagger-ui .info { margin: 20px 0; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-bundle.js"></script>
    <script>
        const spec = %s;
        SwaggerUIBundle({
            spec: spec,
            dom_id: '#swagger-ui',
            presets: [
                SwaggerUIBundle.presets.apis,
                SwaggerUIBundle.presets.standalone
            ],
            layout: "StandaloneLayout"
        });
    </script>
</body>
</html>`, spec.Info.Title, string(specJSON))
}

// RouteDecorator is a function type for decorating routes
type RouteDecorator func(route RouteInfo) RouteInfo

// WithValidation decorates a route with validation information
func WithValidation(requestType, responseType interface{}) RouteDecorator {
	return func(route RouteInfo) RouteInfo {
		route.RequestType = requestType
		route.ResponseType = responseType
		return route
	}
}

// WithSummary decorates a route with a summary
func WithSummary(summary string) RouteDecorator {
	return func(route RouteInfo) RouteInfo {
		route.Summary = summary
		return route
	}
}

// WithDescription decorates a route with a description
func WithDescription(description string) RouteDecorator {
	return func(route RouteInfo) RouteInfo {
		route.Description = description
		return route
	}
}

// WithTags decorates a route with tags
func WithTags(tags ...string) RouteDecorator {
	return func(route RouteInfo) RouteInfo {
		route.Tags = tags
		return route
	}
}

// WithParameters decorates a route with parameters
func WithParameters(parameters ...Parameter) RouteDecorator {
	return func(route RouteInfo) RouteInfo {
		route.Parameters = parameters
		return route
	}
}

// AutoRegister automatically registers routes from Fiber app
func (sm *SwaggerMiddleware) AutoRegister(app *fiber.App) {
	if !sm.enabled {
		return
	}

	// This is a simplified version - in a real implementation,
	// you would need to parse the Fiber app's route tree
	// For now, we'll provide a manual registration method
}

// RegisterRoute manually registers a route
func (sm *SwaggerMiddleware) RegisterRoute(method, path string, decorators ...RouteDecorator) {
	route := RouteInfo{
		Method: method,
		Path:   path,
	}

	// Apply decorators
	for _, decorator := range decorators {
		route = decorator(route)
	}

	sm.generator.AddRoute(route)
}

// RegisterGroup registers a group of routes
func (sm *SwaggerMiddleware) RegisterGroup(prefix string, routes []RouteInfo) {
	for _, route := range routes {
		route.Path = prefix + route.Path
		sm.generator.AddRoute(route)
	}
}

// Helper functions for creating common parameters

// PathParameter creates a path parameter
func PathParameter(name, description string, required bool) Parameter {
	return Parameter{
		Name:        name,
		In:          "path",
		Description: description,
		Required:    required,
		Schema: map[string]interface{}{
			"type": "string",
		},
	}
}

// QueryParameter creates a query parameter
func QueryParameter(name, description string, required bool) Parameter {
	return Parameter{
		Name:        name,
		In:          "query",
		Description: description,
		Required:    required,
		Schema: map[string]interface{}{
			"type": "string",
		},
	}
}

// HeaderParameter creates a header parameter
func HeaderParameter(name, description string, required bool) Parameter {
	return Parameter{
		Name:        name,
		In:          "header",
		Description: description,
		Required:    required,
		Schema: map[string]interface{}{
			"type": "string",
		},
	}
}

// GetTypeName gets the type name for a Go type
func GetTypeName(t interface{}) string {
	rt := reflect.TypeOf(t)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	name := rt.Name()
	if name == "" {
		// Handle anonymous types
		name = strings.ReplaceAll(rt.String(), "*", "")
		name = strings.ReplaceAll(name, "[]", "")
		name = strings.ReplaceAll(name, "map[string]", "")
	}

	return name
}

// GenerateExample generates an example from a struct
func GenerateExample(t interface{}) interface{} {
	rt := reflect.TypeOf(t)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	if rt.Kind() != reflect.Struct {
		return nil
	}

	example := make(map[string]interface{})

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)

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

		// Generate example value based on type
		example[fieldName] = generateExampleValue(fieldType, field)
	}

	return example
}

// generateExampleValue generates an example value for a field
func generateExampleValue(fieldType reflect.Type, field reflect.StructField) interface{} {
	// Check for example tag first
	if example := field.Tag.Get("example"); example != "" {
		return example
	}

	switch fieldType.Kind() {
	case reflect.String:
		if field.Name == "Email" {
			return "user@example.com"
		}
		if field.Name == "Password" {
			return "password123"
		}
		if field.Name == "Name" {
			return "John Doe"
		}
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return 1
	case reflect.Float32, reflect.Float64:
		return 1.0
	case reflect.Bool:
		return true
	case reflect.Slice:
		return []interface{}{}
	case reflect.Map:
		return map[string]interface{}{}
	default:
		return nil
	}
}
