package swagger

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/mithril-framework/mithril/pkg/validation"
)

// OpenAPISpec represents the OpenAPI 3.0 specification
type OpenAPISpec struct {
	OpenAPI    string              `json:"openapi"`
	Info       Info                `json:"info"`
	Servers    []Server            `json:"servers,omitempty"`
	Paths      map[string]PathItem `json:"paths"`
	Components Components          `json:"components,omitempty"`
	Tags       []Tag               `json:"tags,omitempty"`
}

// Info contains metadata about the API
type Info struct {
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Version     string   `json:"version"`
	Contact     *Contact `json:"contact,omitempty"`
	License     *License `json:"license,omitempty"`
}

// Contact information
type Contact struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

// License information
type License struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// Server information
type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// PathItem represents a path in the API
type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
	Patch  *Operation `json:"patch,omitempty"`
}

// Operation represents an API operation
type Operation struct {
	Tags        []string              `json:"tags,omitempty"`
	Summary     string                `json:"summary,omitempty"`
	Description string                `json:"description,omitempty"`
	OperationID string                `json:"operationId,omitempty"`
	Parameters  []Parameter           `json:"parameters,omitempty"`
	RequestBody *RequestBody          `json:"requestBody,omitempty"`
	Responses   map[string]Response   `json:"responses"`
	Security    []map[string][]string `json:"security,omitempty"`
}

// Parameter represents a parameter
type Parameter struct {
	Name        string      `json:"name"`
	In          string      `json:"in"`
	Description string      `json:"description,omitempty"`
	Required    bool        `json:"required,omitempty"`
	Schema      interface{} `json:"schema,omitempty"`
	Example     interface{} `json:"example,omitempty"`
}

// RequestBody represents a request body
type RequestBody struct {
	Description string               `json:"description,omitempty"`
	Content     map[string]MediaType `json:"content"`
	Required    bool                 `json:"required,omitempty"`
}

// Response represents a response
type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
	Headers     map[string]Header    `json:"headers,omitempty"`
}

// MediaType represents a media type
type MediaType struct {
	Schema interface{} `json:"schema,omitempty"`
}

// Header represents a header
type Header struct {
	Description string      `json:"description,omitempty"`
	Schema      interface{} `json:"schema,omitempty"`
}

// Components holds reusable components
type Components struct {
	Schemas map[string]interface{} `json:"schemas,omitempty"`
}

// Tag represents a tag
type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Generator generates OpenAPI specifications
type Generator struct {
	spec      *OpenAPISpec
	validator *validation.Validator
	routes    []RouteInfo
}

// RouteInfo holds information about a route
type RouteInfo struct {
	Method       string
	Path         string
	Handler      string
	Summary      string
	Description  string
	Tags         []string
	RequestType  interface{}
	ResponseType interface{}
	Parameters   []Parameter
}

// NewGenerator creates a new Swagger generator
func NewGenerator(title, version, description string) *Generator {
	return &Generator{
		spec: &OpenAPISpec{
			OpenAPI: "3.0.0",
			Info: Info{
				Title:       title,
				Version:     version,
				Description: description,
			},
			Paths: make(map[string]PathItem),
			Components: Components{
				Schemas: make(map[string]interface{}),
			},
		},
		validator: validation.NewValidator(),
		routes:    make([]RouteInfo, 0),
	}
}

// AddRoute adds a route to the specification
func (g *Generator) AddRoute(route RouteInfo) {
	g.routes = append(g.routes, route)
	g.generatePathItem(route)
}

// AddServer adds a server to the specification
func (g *Generator) AddServer(url, description string) {
	g.spec.Servers = append(g.spec.Servers, Server{
		URL:         url,
		Description: description,
	})
}

// AddTag adds a tag to the specification
func (g *Generator) AddTag(name, description string) {
	g.spec.Tags = append(g.spec.Tags, Tag{
		Name:        name,
		Description: description,
	})
}

// Generate generates the OpenAPI specification
func (g *Generator) Generate() (*OpenAPISpec, error) {
	// Generate schemas for all request/response types
	for _, route := range g.routes {
		if route.RequestType != nil {
			schema, err := g.validator.GenerateSchema(route.RequestType)
			if err == nil {
				g.spec.Components.Schemas[g.getTypeName(route.RequestType)] = schema
			}
		}

		if route.ResponseType != nil {
			schema, err := g.validator.GenerateSchema(route.ResponseType)
			if err == nil {
				g.spec.Components.Schemas[g.getTypeName(route.ResponseType)] = schema
			}
		}
	}

	return g.spec, nil
}

// GenerateJSON generates the OpenAPI specification as JSON
func (g *Generator) GenerateJSON() ([]byte, error) {
	spec, err := g.Generate()
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(spec, "", "  ")
}

// generatePathItem generates a path item for a route
func (g *Generator) generatePathItem(route RouteInfo) {
	pathItem, exists := g.spec.Paths[route.Path]
	if !exists {
		pathItem = PathItem{}
	}

	operation := &Operation{
		Tags:        route.Tags,
		Summary:     route.Summary,
		Description: route.Description,
		OperationID: g.generateOperationID(route.Method, route.Path),
		Parameters:  route.Parameters,
		Responses:   g.generateResponses(route),
	}

	// Add request body if present
	if route.RequestType != nil {
		operation.RequestBody = g.generateRequestBody(route)
	}

	// Set the appropriate method
	switch strings.ToUpper(route.Method) {
	case "GET":
		pathItem.Get = operation
	case "POST":
		pathItem.Post = operation
	case "PUT":
		pathItem.Put = operation
	case "DELETE":
		pathItem.Delete = operation
	case "PATCH":
		pathItem.Patch = operation
	}

	g.spec.Paths[route.Path] = pathItem
}

// generateResponses generates responses for a route
func (g *Generator) generateResponses(route RouteInfo) map[string]Response {
	responses := make(map[string]Response)

	// Default success response
	successResponse := Response{
		Description: "Success",
	}

	if route.ResponseType != nil {
		successResponse.Content = map[string]MediaType{
			"application/json": {
				Schema: map[string]interface{}{
					"$ref": "#/components/schemas/" + g.getTypeName(route.ResponseType),
				},
			},
		}
	} else {
		successResponse.Content = map[string]MediaType{
			"application/json": {
				Schema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"message": map[string]string{"type": "string"},
						"data":    map[string]string{"type": "object"},
					},
				},
			},
		}
	}

	responses["200"] = successResponse

	// Common error responses
	responses["400"] = Response{
		Description: "Bad Request",
		Content: map[string]MediaType{
			"application/json": {
				Schema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"error":   map[string]string{"type": "string"},
						"message": map[string]string{"type": "string"},
					},
				},
			},
		},
	}

	responses["422"] = Response{
		Description: "Validation Error",
		Content: map[string]MediaType{
			"application/json": {
				Schema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"error":   map[string]string{"type": "string"},
						"message": map[string]string{"type": "string"},
					},
				},
			},
		},
	}

	responses["500"] = Response{
		Description: "Internal Server Error",
		Content: map[string]MediaType{
			"application/json": {
				Schema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"error":   map[string]string{"type": "string"},
						"message": map[string]string{"type": "string"},
					},
				},
			},
		},
	}

	return responses
}

// generateRequestBody generates a request body for a route
func (g *Generator) generateRequestBody(route RouteInfo) *RequestBody {
	if route.RequestType == nil {
		return nil
	}

	return &RequestBody{
		Description: "Request body",
		Required:    true,
		Content: map[string]MediaType{
			"application/json": {
				Schema: map[string]interface{}{
					"$ref": "#/components/schemas/" + g.getTypeName(route.RequestType),
				},
			},
		},
	}
}

// generateOperationID generates an operation ID
func (g *Generator) generateOperationID(method, path string) string {
	// Convert path parameters to camelCase
	operationID := strings.ToLower(method) + strings.ReplaceAll(path, "/", "_")
	operationID = strings.ReplaceAll(operationID, "{", "")
	operationID = strings.ReplaceAll(operationID, "}", "")
	operationID = strings.ReplaceAll(operationID, "__", "_")
	operationID = strings.Trim(operationID, "_")

	return operationID
}

// getTypeName gets the type name for a Go type
func (g *Generator) getTypeName(t interface{}) string {
	rt := reflect.TypeOf(t)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	return rt.Name()
}

// Middleware for serving Swagger UI
func SwaggerUI(spec *OpenAPISpec) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Path() == "/docs" {
			return c.SendString(generateSwaggerHTML(spec))
		}
		return c.Next()
	}
}

// generateSwaggerHTML generates the Swagger UI HTML
func generateSwaggerHTML(spec *OpenAPISpec) string {
	specJSON, _ := json.Marshal(spec)

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>%s API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui.css" />
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
            ]
        });
    </script>
</body>
</html>`, spec.Info.Title, string(specJSON))
}
