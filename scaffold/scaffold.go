// Package scaffold generates code for Mithril projects (used by `mithril make:resource` and the
// MCP server). Generators are deterministic and print exactly which files they touch, so humans
// and AI agents can review the result.
package scaffold

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

// Field is one requested field, parsed from "name:type[:required]".
type Field struct {
	Name     string // snake_case column / JSON name
	Type     string // string | text | email | int | float | bool | time | uuid | json
	Required bool
}

// FieldTypes lists the supported field types.
var FieldTypes = []string{"string", "text", "email", "url", "int", "float", "bool", "time", "uuid", "json"}

// ResourceSpec describes a resource to generate.
type ResourceSpec struct {
	Model  string  // Go type name, e.g. "Post" (singular)
	Name   string  // resource / table name, e.g. "posts" (default: plural snake_case of Model)
	Fields []Field // user fields (id, timestamps and owner are added automatically)
	Owner  bool    // add owner_id referencing users (rows owned by their creator)
}

// File is a generated or modified file.
type File struct {
	Path    string `json:"path"`
	Action  string `json:"action"` // create | modify
	Content string `json:"content"`
}

var identRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

// ParseFields parses ["title:string:required", "body:text"].
func ParseFields(args []string) ([]Field, error) {
	var out []Field
	seen := map[string]bool{}
	for _, a := range args {
		parts := strings.Split(a, ":")
		if len(parts) < 2 || len(parts) > 3 {
			return nil, fmt.Errorf("field %q: want name:type[:required]", a)
		}
		f := Field{Name: SnakeCase(parts[0]), Type: strings.ToLower(parts[1])}
		if !identRe.MatchString(parts[0]) {
			return nil, fmt.Errorf("field %q: invalid name", a)
		}
		if !contains(FieldTypes, f.Type) {
			return nil, fmt.Errorf("field %q: unknown type %q (one of %s)", a, f.Type, strings.Join(FieldTypes, ", "))
		}
		if len(parts) == 3 {
			if parts[2] != "required" {
				return nil, fmt.Errorf("field %q: unknown modifier %q", a, parts[2])
			}
			f.Required = true
		}
		switch f.Name {
		case "id", "created_at", "updated_at", "owner_id":
			return nil, fmt.Errorf("field %q is added automatically", f.Name)
		}
		if seen[f.Name] {
			return nil, fmt.Errorf("duplicate field %q", f.Name)
		}
		seen[f.Name] = true
		out = append(out, f)
	}
	return out, nil
}

// Project describes the Mithril project being modified.
type Project struct {
	Dir    string // project root (contains go.mod)
	Module string // module path from go.mod
}

// LoadProject reads go.mod in dir.
func LoadProject(dir string) (*Project, error) {
	b, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return nil, fmt.Errorf("not a Go module (no go.mod in %s)", dir)
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "module ") {
			return &Project{Dir: dir, Module: strings.TrimSpace(strings.TrimPrefix(line, "module "))}, nil
		}
	}
	return nil, errors.New("go.mod has no module line")
}

// PlanResource computes the files for spec without writing anything.
func (p *Project) PlanResource(spec ResourceSpec, now time.Time) ([]File, error) {
	if !identRe.MatchString(spec.Model) {
		return nil, fmt.Errorf("invalid model name %q", spec.Model)
	}
	spec.Model = Pascal(spec.Model)
	if spec.Name == "" {
		spec.Name = Plural(SnakeCase(spec.Model))
	}
	if !regexp.MustCompile(`^[a-z][a-z0-9_]*$`).MatchString(spec.Name) {
		return nil, fmt.Errorf("invalid resource name %q (lowercase snake_case)", spec.Name)
	}
	modelPath := filepath.Join("models", SnakeCase(spec.Model)+".go")
	if _, err := os.Stat(filepath.Join(p.Dir, modelPath)); err == nil {
		return nil, fmt.Errorf("%s already exists", modelPath)
	}
	model, err := modelSource(spec)
	if err != nil {
		return nil, err
	}
	files := []File{
		{Path: modelPath, Action: "create", Content: model},
		{Path: filepath.Join("migrations", now.UTC().Format("20060102150405")+"_create_"+spec.Name+".sql"), Action: "create", Content: migrationSource(spec)},
	}
	reg, err := p.registration(spec)
	if err != nil {
		return nil, err
	}
	return append(files, reg), nil
}

// Apply writes planned files.
func (p *Project) Apply(files []File) error {
	for _, f := range files {
		full := filepath.Join(p.Dir, f.Path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(full, []byte(f.Content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

const marker = "// mithril:resources"

func (p *Project) registration(spec ResourceSpec) (File, error) {
	path := "resources.go"
	b, err := os.ReadFile(filepath.Join(p.Dir, path))
	if err != nil {
		return File{}, fmt.Errorf("%s not found (projects created by `mithril new` have one): %w", path, err)
	}
	src := string(b)
	if !strings.Contains(src, marker) {
		return File{}, fmt.Errorf("%s has no %q marker line", path, marker)
	}
	line := fmt.Sprintf("resource.Define[models.%s](resource.Options{Name: %q}),\n\t\t%s", spec.Model, spec.Name, marker)
	src = strings.Replace(src, marker, line, 1)
	imp := `"` + p.Module + `/models"`
	if !strings.Contains(src, imp) {
		src = strings.Replace(src, "import (", "import (\n\t"+imp, 1)
	}
	formatted, err := format.Source([]byte(src))
	if err != nil {
		return File{}, fmt.Errorf("format %s: %w", path, err)
	}
	return File{Path: path, Action: "modify", Content: string(formatted)}, nil
}

type goField struct{ name, typ, tags string }

func modelSource(spec ResourceSpec) (string, error) {
	imports := map[string]bool{"github.com/google/uuid": true, "time": true}
	fields := []goField{{"ID", "uuid.UUID", `json:"id"`}}
	for _, f := range spec.Fields {
		typ, extra := "", ""
		var rules []string
		if f.Required {
			rules = append(rules, "required")
		}
		switch f.Type {
		case "string":
			typ = "string"
			rules = append(rules, "max=255")
		case "email":
			typ = "string"
			rules = append(rules, "email")
		case "url":
			typ = "string"
			rules = append(rules, "url")
		case "text":
			typ, extra = "string", ` mithril:"text"`
		case "int":
			typ = "int64"
		case "float":
			typ = "float64"
		case "bool":
			typ = "bool"
		case "time":
			typ = "*time.Time"
		case "uuid":
			typ = "*uuid.UUID"
		case "json":
			typ = "json.RawMessage"
			imports["encoding/json"] = true
		}
		tags := fmt.Sprintf(`json:"%s"`, f.Name)
		if len(rules) > 0 {
			tags += fmt.Sprintf(` validate:"%s"`, strings.Join(rules, ","))
		}
		fields = append(fields, goField{Pascal(f.Name), typ, tags + extra})
	}
	if spec.Owner {
		fields = append(fields, goField{"OwnerID", "uuid.UUID", `json:"owner_id" mithril:"owner"`})
	}
	fields = append(fields,
		goField{"CreatedAt", "time.Time", `json:"created_at"`},
		goField{"UpdatedAt", "time.Time", `json:"updated_at"`})

	var b bytes.Buffer
	fmt.Fprintf(&b, "package models\n\nimport (\n")
	var std, ext []string
	for k := range imports {
		if strings.Contains(strings.Split(k, "/")[0], ".") {
			ext = append(ext, k)
		} else {
			std = append(std, k)
		}
	}
	sort.Strings(std)
	sort.Strings(ext)
	for _, k := range std {
		fmt.Fprintf(&b, "\t%q\n", k)
	}
	if len(std) > 0 && len(ext) > 0 {
		b.WriteString("\n")
	}
	for _, k := range ext {
		fmt.Fprintf(&b, "\t%q\n", k)
	}
	fmt.Fprintf(&b, ")\n\n// %s is the %s resource (table %s). Generated by `mithril make:resource`.\ntype %s struct {\n",
		spec.Model, spec.Name, spec.Name, spec.Model)
	for _, f := range fields {
		fmt.Fprintf(&b, "\t%s %s `%s`\n", f.name, f.typ, f.tags)
	}
	b.WriteString("}\n")
	out, err := format.Source(b.Bytes())
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func migrationSource(spec ResourceSpec) string {
	var cols []string
	cols = append(cols, "    id UUID PRIMARY KEY DEFAULT gen_random_uuid()")
	for _, f := range spec.Fields {
		var c string
		switch f.Type {
		case "string", "text", "email", "url":
			c = "TEXT NOT NULL DEFAULT ''"
		case "int":
			c = "BIGINT NOT NULL DEFAULT 0"
		case "float":
			c = "DOUBLE PRECISION NOT NULL DEFAULT 0"
		case "bool":
			c = "BOOLEAN NOT NULL DEFAULT false"
		case "time":
			c = "TIMESTAMPTZ"
		case "uuid":
			c = "UUID"
		case "json":
			c = "JSONB NOT NULL DEFAULT '{}'"
		}
		cols = append(cols, fmt.Sprintf("    %s %s", f.Name, c))
	}
	if spec.Owner {
		cols = append(cols, "    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE")
	}
	cols = append(cols,
		"    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()",
		"    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()")
	var b strings.Builder
	fmt.Fprintf(&b, "-- +goose Up\nCREATE TABLE %s (\n%s\n);\nCREATE INDEX idx_%s_created_at ON %s (created_at DESC);\n",
		spec.Name, strings.Join(cols, ",\n"), spec.Name, spec.Name)
	if spec.Owner {
		fmt.Fprintf(&b, "CREATE INDEX idx_%s_owner_id ON %s (owner_id);\n", spec.Name, spec.Name)
	}
	fmt.Fprintf(&b, "\n-- +goose Down\nDROP TABLE IF EXISTS %s;\n", spec.Name)
	return b.String()
}

// SnakeCase converts "BlogPost" or "blogPost" to "blog_post".
func SnakeCase(s string) string {
	var b strings.Builder
	r := []rune(s)
	for i, c := range r {
		if unicode.IsUpper(c) {
			if i > 0 && r[i-1] != '_' && (unicode.IsLower(r[i-1]) || (i+1 < len(r) && unicode.IsLower(r[i+1]))) {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(c))
		} else {
			b.WriteRune(c)
		}
	}
	return b.String()
}

// Pascal converts "blog_post" or "blogPost" to "BlogPost" (keeping common initialisms upper-case).
func Pascal(s string) string {
	var b strings.Builder
	for _, part := range strings.Split(SnakeCase(s), "_") {
		if part == "" {
			continue
		}
		switch part {
		case "id", "url", "uuid", "api", "http", "json", "sql", "ip":
			b.WriteString(strings.ToUpper(part))
		default:
			b.WriteString(strings.ToUpper(part[:1]) + part[1:])
		}
	}
	return b.String()
}

// Plural is a small English pluralizer for table names.
func Plural(s string) string {
	switch {
	case strings.HasSuffix(s, "y") && len(s) > 1 && !strings.ContainsRune("aeiou", rune(s[len(s)-2])):
		return s[:len(s)-1] + "ies"
	case strings.HasSuffix(s, "s"), strings.HasSuffix(s, "x"), strings.HasSuffix(s, "ch"), strings.HasSuffix(s, "sh"):
		return s + "es"
	}
	return s + "s"
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
