// Package openapi builds an OpenAPI 3.1 document from Go types at runtime.
//
// Schemas are derived by reflection from struct fields and tags, so the spec is deterministic,
// works offline and always matches the code:
//
//	json:"name,omitempty"   property name (omitempty = optional unless validate:"required")
//	validate:"required,min=3,max=50,email,oneof=a b"
//	doc:"Human description"   example:"value"   mithril:"readonly"
package openapi

import (
	"encoding/json"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Schema is a JSON Schema (OpenAPI 3.1 dialect) node.
type Schema struct {
	Ref                  string             `json:"$ref,omitempty"`
	Type                 any                `json:"type,omitempty"` // string or []string (nullable)
	Format               string             `json:"format,omitempty"`
	Description          string             `json:"description,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	Required             []string           `json:"required,omitempty"`
	Items                *Schema            `json:"items,omitempty"`
	AdditionalProperties *Schema            `json:"additionalProperties,omitempty"`
	Enum                 []any              `json:"enum,omitempty"`
	MinLength            *int               `json:"minLength,omitempty"`
	MaxLength            *int               `json:"maxLength,omitempty"`
	Minimum              *float64           `json:"minimum,omitempty"`
	Maximum              *float64           `json:"maximum,omitempty"`
	ReadOnly             bool               `json:"readOnly,omitempty"`
	Example              any                `json:"example,omitempty"`
}

// Parameter is a path or query parameter.
type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"` // path | query | header
	Required    bool    `json:"required,omitempty"`
	Description string  `json:"description,omitempty"`
	Schema      *Schema `json:"schema"`
}

// MediaType wraps a schema for a content type.
type MediaType struct {
	Schema *Schema `json:"schema"`
}

// RequestBody describes an operation's input body.
type RequestBody struct {
	Required bool                 `json:"required,omitempty"`
	Content  map[string]MediaType `json:"content"`
}

// Response describes one status code.
type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

// Operation is one method on one path.
type Operation struct {
	OperationID string                `json:"operationId,omitempty"`
	Summary     string                `json:"summary,omitempty"`
	Description string                `json:"description,omitempty"`
	Tags        []string              `json:"tags,omitempty"`
	Parameters  []Parameter           `json:"parameters,omitempty"`
	RequestBody *RequestBody          `json:"requestBody,omitempty"`
	Responses   map[string]Response   `json:"responses"`
	Security    []map[string][]string `json:"security,omitempty"`
}

// Document is an OpenAPI 3.1 document. Safe for concurrent use.
type Document struct {
	mu         sync.RWMutex
	cached     []byte                          // rendered JSON; reset by every mutation
	version    uint64                          // incremented by every mutation
	OpenAPI    string                          `json:"openapi"`
	Info       Info                            `json:"info"`
	Paths      map[string]map[string]Operation `json:"paths"`
	Components Components                      `json:"components"`
}

// Info is the document's metadata.
type Info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

// Components holds reusable schemas and security schemes.
type Components struct {
	Schemas         map[string]*Schema        `json:"schemas,omitempty"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty"`
}

// SecurityScheme describes an auth mechanism.
type SecurityScheme struct {
	Type         string `json:"type"`
	Scheme       string `json:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty"`
}

// BearerAuth is the security requirement for JWT-protected operations.
var BearerAuth = []map[string][]string{{"bearerAuth": {}}}

// New creates an empty document with a bearer JWT security scheme.
func New(title, version string) *Document {
	return &Document{
		OpenAPI: "3.1.0",
		Info:    Info{Title: title, Version: version},
		Paths:   map[string]map[string]Operation{},
		Components: Components{
			Schemas: map[string]*Schema{},
			SecuritySchemes: map[string]SecurityScheme{
				"bearerAuth": {Type: "http", Scheme: "bearer", BearerFormat: "JWT"},
			},
		},
	}
}

var fiberParam = regexp.MustCompile(`:([A-Za-z0-9_]+)\??`)

// PathFromFiber converts /users/:id to /users/{id}.
func PathFromFiber(p string) string { return fiberParam.ReplaceAllString(p, "{$1}") }

// AddOperation registers op for method and (Fiber-style) path. Path parameters missing from
// op.Parameters are added automatically as strings.
func (d *Document) AddOperation(method, path string, op Operation) {
	if op.Responses == nil {
		op.Responses = map[string]Response{"200": {Description: "OK"}}
	}
	for _, m := range fiberParam.FindAllStringSubmatch(path, -1) {
		if !hasParam(op.Parameters, m[1]) {
			op.Parameters = append(op.Parameters, Parameter{Name: m[1], In: "path", Required: true, Schema: &Schema{Type: "string"}})
		}
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cached = nil
	d.version++
	p := PathFromFiber(path)
	if d.Paths[p] == nil {
		d.Paths[p] = map[string]Operation{}
	}
	d.Paths[p][strings.ToLower(method)] = op
}

func hasParam(ps []Parameter, name string) bool {
	for _, p := range ps {
		if p.Name == name {
			return true
		}
	}
	return false
}

// MarshalJSON renders the document (holding the read lock).
func (d *Document) MarshalJSON() ([]byte, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	type plain struct {
		OpenAPI    string                          `json:"openapi"`
		Info       Info                            `json:"info"`
		Paths      map[string]map[string]Operation `json:"paths"`
		Components Components                      `json:"components"`
	}
	return json.Marshal(plain{d.OpenAPI, d.Info, d.Paths, d.Components})
}

// Operations returns "METHOD path" keys in sorted order (useful for tests and tooling).
func (d *Document) Operations() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var out []string
	for p, ops := range d.Paths {
		for m := range ops {
			out = append(out, strings.ToUpper(m)+" "+p)
		}
	}
	sort.Strings(out)
	return out
}

// SchemaRef returns a schema for v's type; named struct types are stored in components and
// referenced. Pass a zero value or a typed nil pointer, e.g. SchemaRef((*User)(nil)).
func (d *Document) SchemaRef(v any) *Schema {
	if v == nil {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cached = nil
	d.version++
	return d.schemaFor(reflect.TypeOf(v), map[reflect.Type]bool{})
}

// SchemaForType is SchemaRef for a reflect.Type.
func (d *Document) SchemaForType(t reflect.Type) *Schema {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cached = nil
	d.version++
	return d.schemaFor(t, map[reflect.Type]bool{})
}

var (
	timeType = reflect.TypeOf(time.Time{})
	uuidType = reflect.TypeOf(uuid.UUID{})
	rawType  = reflect.TypeOf(json.RawMessage{})
)

func (d *Document) schemaFor(t reflect.Type, seen map[reflect.Type]bool) *Schema {
	nullable := false
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
		nullable = true
	}
	s := d.baseSchema(t, seen)
	if nullable && s.Ref == "" {
		if typ, ok := s.Type.(string); ok {
			s.Type = []string{typ, "null"}
		}
	}
	return s
}

func (d *Document) baseSchema(t reflect.Type, seen map[reflect.Type]bool) *Schema {
	switch t {
	case timeType:
		return &Schema{Type: "string", Format: "date-time"}
	case uuidType:
		return &Schema{Type: "string", Format: "uuid"}
	case rawType:
		return &Schema{}
	}
	switch t.Kind() {
	case reflect.Bool:
		return &Schema{Type: "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return &Schema{Type: "integer", Format: "int32"}
	case reflect.Int64, reflect.Uint64:
		return &Schema{Type: "integer", Format: "int64"}
	case reflect.Float32, reflect.Float64:
		return &Schema{Type: "number"}
	case reflect.String:
		return &Schema{Type: "string"}
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			return &Schema{Type: "string", Format: "byte"}
		}
		return &Schema{Type: "array", Items: d.schemaFor(t.Elem(), seen)}
	case reflect.Map:
		return &Schema{Type: "object", AdditionalProperties: d.schemaFor(t.Elem(), seen)}
	case reflect.Interface:
		return &Schema{}
	case reflect.Struct:
		if t.Name() == "" {
			return d.structSchema(t, seen)
		}
		name := schemaName(t)
		if _, ok := d.Components.Schemas[name]; !ok && !seen[t] {
			seen[t] = true
			d.Components.Schemas[name] = d.structSchema(t, seen)
		}
		return &Schema{Ref: "#/components/schemas/" + name}
	}
	return &Schema{}
}

func schemaName(t reflect.Type) string {
	pkg := t.PkgPath()
	if i := strings.LastIndex(pkg, "/"); i >= 0 {
		pkg = pkg[i+1:]
	}
	name := t.Name()
	if pkg != "" && pkg != "main" {
		name = strings.ToUpper(pkg[:1]) + pkg[1:] + name
	}
	return strings.NewReplacer("[", "_", "]", "", "*", "", ".", "_", "/", "_").Replace(name)
}

// FieldName returns the JSON property name for a struct field ("" when skipped).
func FieldName(f reflect.StructField) (name string, omitempty bool) {
	if !f.IsExported() {
		return "", false
	}
	tag := f.Tag.Get("json")
	if tag == "-" {
		return "", false
	}
	parts := strings.Split(tag, ",")
	name = parts[0]
	if name == "" {
		name = f.Name
	}
	for _, p := range parts[1:] {
		if p == "omitempty" || p == "omitzero" {
			omitempty = true
		}
	}
	return name, omitempty
}

func (d *Document) structSchema(t reflect.Type, seen map[reflect.Type]bool) *Schema {
	s := &Schema{Type: "object", Properties: map[string]*Schema{}}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Anonymous && f.Type.Kind() == reflect.Struct && f.Tag.Get("json") == "" {
			emb := d.structSchema(f.Type, seen)
			for k, v := range emb.Properties {
				s.Properties[k] = v
			}
			s.Required = append(s.Required, emb.Required...)
			continue
		}
		name, omitempty := FieldName(f)
		if name == "" || f.Tag.Get("path") != "" || f.Tag.Get("query") != "" {
			continue // path/query inputs are documented as parameters, not body properties
		}
		ps := d.schemaFor(f.Type, seen)
		ApplyTags(ps, f)
		if ps.Ref != "" && (ps.Description != "" || ps.ReadOnly) {
			ps = &Schema{Ref: ps.Ref, Description: ps.Description, ReadOnly: ps.ReadOnly} // keep ref siblings minimal
		}
		s.Properties[name] = ps
		rules := ParseRules(f.Tag.Get("validate"))
		_ = omitempty // optional-ness is driven by validate:"required" so input and output schemas agree
		if _, ok := rules["required"]; ok {
			s.Required = append(s.Required, name)
		}
	}
	sort.Strings(s.Required)
	return s
}

// ApplyTags copies doc/example/validate/mithril tag information onto a property schema.
func ApplyTags(s *Schema, f reflect.StructField) {
	if v := f.Tag.Get("doc"); v != "" {
		s.Description = v
	}
	if v := f.Tag.Get("example"); v != "" {
		s.Example = v
	}
	for _, opt := range strings.Split(f.Tag.Get("mithril"), ",") {
		if opt == "readonly" || opt == "pk" || opt == "owner" {
			s.ReadOnly = true
		}
	}
	rules := ParseRules(f.Tag.Get("validate"))
	isString := s.Type == "string"
	for k, v := range rules {
		switch k {
		case "email":
			s.Format = "email"
		case "url":
			s.Format = "uri"
		case "uuid":
			s.Format = "uuid"
		case "min", "max":
			n, err := strconv.ParseFloat(v, 64)
			if err != nil {
				continue
			}
			if isString {
				i := int(n)
				if k == "min" {
					s.MinLength = &i
				} else {
					s.MaxLength = &i
				}
			} else if k == "min" {
				s.Minimum = &n
			} else {
				s.Maximum = &n
			}
		case "oneof":
			for _, o := range strings.Fields(v) {
				s.Enum = append(s.Enum, o)
			}
		}
	}
}

// ParseRules parses a validate tag like "required,min=3,oneof=a b" into a map.
func ParseRules(tag string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(tag, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		k, v, _ := strings.Cut(part, "=")
		out[k] = v
	}
	return out
}

// SetComponent stores s under components/schemas/name and returns a reference to it.
func (d *Document) SetComponent(name string, s *Schema) *Schema {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cached = nil
	d.version++
	d.Components.Schemas[name] = s
	return &Schema{Ref: "#/components/schemas/" + name}
}
