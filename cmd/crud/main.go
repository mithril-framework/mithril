// Command crud generates CRUD repository, handlers, routes, and registers them from a model file.
// Usage: go run ./cmd/crud ModelName   (e.g. go run ./cmd/crud User)
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

func main() {
	dryRun := false
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "--dry-run" {
		dryRun = true
		args = args[1:]
	}
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: go run ./cmd/crud [--dry-run] <ModelName>")
		os.Exit(1)
	}
	modelName := strings.TrimSpace(args[0])
	if modelName == "" {
		fmt.Fprintln(os.Stderr, "Usage: go run ./cmd/crud [--dry-run] <ModelName>")
		os.Exit(1)
	}
	// PascalCase -> lowercase for file name
	lower := strings.ToLower(modelName)
	modelPath := filepath.Join("database", "models", lower+".go")
	if _, err := os.Stat(modelPath); err != nil {
		fmt.Fprintf(os.Stderr, "Model file not found: %s\n", modelPath)
		os.Exit(1)
	}

	info, err := parseModelFile(modelPath, modelName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse model: %v\n", err)
		os.Exit(1)
	}

	repoPath := filepath.Join("database", "repositories", info.SnakeName+"_repository.go")
	handlersPath := filepath.Join("internal", "crud", lower, "handlers.go")
	routesPath := filepath.Join("routes", "crud_"+lower+".go")
	if dryRun {
		fmt.Println("[dry-run] Would generate CRUD for", modelName)
		fmt.Println("  ", repoPath)
		fmt.Println("  ", handlersPath)
		fmt.Println("  ", routesPath)
		fmt.Println("[dry-run] Would update routes/register.go with MountCrud" + info.Name + "Routes")
		return
	}

	if _, err := os.Stat(repoPath); err == nil {
		fmt.Fprintf(os.Stderr, "Repository already exists: %s (skip)\n", repoPath)
	} else {
		if err := generateRepository(repoPath, info); err != nil {
			fmt.Fprintf(os.Stderr, "Generate repository: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Generated", repoPath)
	}

	handlersDir := filepath.Join("internal", "crud", lower)
	handlersPath = filepath.Join(handlersDir, "handlers.go")
	if err := os.MkdirAll(handlersDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Mkdir handlers: %v\n", err)
		os.Exit(1)
	}
	if err := generateHandlers(handlersPath, info); err != nil {
		fmt.Fprintf(os.Stderr, "Generate handlers: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Generated", handlersPath)

	routesPath = filepath.Join("routes", "crud_"+lower+".go")
	if err := generateRoutes(routesPath, info); err != nil {
		fmt.Fprintf(os.Stderr, "Generate routes: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Generated", routesPath)

	if err := appendRegister(info); err != nil {
		fmt.Fprintf(os.Stderr, "Append register: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Updated routes/register.go")
}

type modelInfo struct {
	Name       string   // PascalCase, e.g. User
	Lower      string   // lowercase, e.g. user
	SnakeName  string   // for repo file, e.g. user
	TableName  string   // plural lowercase, e.g. users
	Plural     string   // for URL, e.g. users
	IDType     string   // uuid.UUID or int64
	IDPkg      string   // e.g. github.com/google/uuid
	Fields     []fieldInfo
	HasUUID    bool
	HasTime    bool
}

type fieldInfo struct {
	Name      string // Go name, e.g. Email
	Snake     string // snake_case, e.g. email
	GoType    string // e.g. string, uuid.UUID
	SQLType   string // e.g. TEXT, UUID
	OmitID    bool   // skip in insert (auto)
	OmitTimes bool   // skip in insert (auto)
}

func parseModelFile(path, structName string) (*modelInfo, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var fields []fieldInfo
	var idType string
	var idPkg string
	hasUUID := false
	hasTime := false

	ast.Inspect(f, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok || typeSpec.Name.Name != structName {
			return true
		}
		st, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}
		for _, fl := range st.Fields.List {
			typ := typeString(fl.Type)
			for _, name := range fl.Names {
				if name.Name == "" {
					continue
				}
				snake := toSnake(name.Name)
				sqlType := goTypeToSQL(typ)
				fi := fieldInfo{
					Name:   name.Name,
					Snake:  snake,
					GoType: typ,
					SQLType: sqlType,
				}
				if name.Name == "ID" {
					idType = typ
					if strings.Contains(typ, "uuid") {
						idPkg = "github.com/google/uuid"
						hasUUID = true
					}
					fi.OmitID = true
				}
				if name.Name == "CreatedAt" || name.Name == "UpdatedAt" {
					fi.OmitTimes = true
					hasTime = true
				}
				fields = append(fields, fi)
			}
		}
		return false
	})

	if len(fields) == 0 {
		return nil, fmt.Errorf("struct %s not found in %s", structName, path)
	}
	if idType == "" {
		idType = "uuid.UUID"
		idPkg = "github.com/google/uuid"
		hasUUID = true
	}

	lower := strings.ToLower(structName)
	tableName := lower + "s"
	if strings.HasSuffix(lower, "s") {
		tableName = lower + "es"
	} else if strings.HasSuffix(lower, "y") {
		tableName = strings.TrimSuffix(lower, "y") + "ies"
	}

	return &modelInfo{
		Name:      structName,
		Lower:     lower,
		SnakeName: toSnake(structName),
		TableName: tableName,
		Plural:   tableName,
		IDType:   idType,
		IDPkg:    idPkg,
		Fields:   fields,
		HasUUID:  hasUUID,
		HasTime:  hasTime,
	}, nil
}

func typeString(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok {
			return x.Name + "." + t.Sel.Name
		}
	}
	return ""
}

func toSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}

func goTypeToSQL(goType string) string {
	switch goType {
	case "string":
		return "TEXT"
	case "bool", "boolean":
		return "BOOLEAN"
	case "int", "int32":
		return "INTEGER"
	case "int64":
		return "BIGINT"
	case "float64":
		return "DOUBLE PRECISION"
	case "time.Time":
		return "TIMESTAMPTZ"
	}
	if strings.Contains(goType, "uuid") {
		return "UUID"
	}
	return "TEXT"
}

func generateRepository(path string, info *modelInfo) error {
	tpl := `package repositories

import (
	"context"

	"github.com/mithril-framework/mithril/database/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// {{.Name}}Repository provides {{.Lower}} persistence.
type {{.Name}}Repository struct {
	db *pgxpool.Pool
}

// New{{.Name}}Repository returns a new {{.Name}}Repository.
func New{{.Name}}Repository(db *pgxpool.Pool) *{{.Name}}Repository {
	return &{{.Name}}Repository{db: db}
}

// Create inserts a {{.Lower}}.
func (r *{{.Name}}Repository) Create(ctx context.Context, m *models.{{.Name}}) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	query := ` + "`" + `
		INSERT INTO {{.TableName}} ({{.InsertColumns}})
		VALUES ({{.InsertPlaceholders}}){{if .HasReturning}}
		RETURNING {{.ReturningColumns}}{{end}}
	` + "`" + `
	{{if .HasReturning}}return r.db.QueryRow(ctx, query, {{.InsertArgs}}).Scan({{.ScanArgs}}){{else}}_, err := r.db.Exec(ctx, query, {{.InsertArgs}})
	return err{{end}}
}

// GetByID returns a {{.Lower}} by id.
func (r *{{.Name}}Repository) GetByID(ctx context.Context, id uuid.UUID) (*models.{{.Name}}, error) {
	query := ` + "`" + `SELECT {{.SelectColumns}} FROM {{.TableName}} WHERE id = $1` + "`" + `
	var m models.{{.Name}}
	err := r.db.QueryRow(ctx, query, id).Scan({{.ScanRefs}})
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// Update updates a {{.Lower}} by id.
func (r *{{.Name}}Repository) Update(ctx context.Context, m *models.{{.Name}}) error {
	query := ` + "`" + `UPDATE {{.TableName}} SET {{.UpdateSet}} WHERE id = ${{.UpdateWhereParam}}` + "`" + `
	_, err := r.db.Exec(ctx, query, {{.UpdateArgs}})
	return err
}

// Delete deletes a {{.Lower}} by id.
func (r *{{.Name}}Repository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, ` + "`DELETE FROM {{.TableName}} WHERE id = $1`" + `, id)
	return err
}

// List returns {{.Plural}} with limit and offset.
func (r *{{.Name}}Repository) List(ctx context.Context, limit, offset int) ([]*models.{{.Name}}, error) {
	query := ` + "`" + `SELECT {{.SelectColumns}} FROM {{.TableName}} ORDER BY {{.OrderBy}} LIMIT $1 OFFSET $2` + "`" + `
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.{{.Name}}
	for rows.Next() {
		var m models.{{.Name}}
		if err := rows.Scan({{.ScanRefs}}); err != nil {
			return nil, err
		}
		list = append(list, &m)
	}
	return list, rows.Err()
}
`
	var insertCols, insertPh, returningCols, insertArgs, scanArgs, selectCols, scanRefs, updateSet, updateArgs []string
	param := 1
	for _, f := range info.Fields {
		if f.OmitTimes {
			returningCols = append(returningCols, f.Snake)
			scanArgs = append(scanArgs, "&m."+f.Name)
			continue
		}
		insertCols = append(insertCols, f.Snake)
		insertPh = append(insertPh, fmt.Sprintf("$%d", param))
		param++
		insertArgs = append(insertArgs, "m."+f.Name)
	}
	for _, f := range info.Fields {
		selectCols = append(selectCols, f.Snake)
		scanRefs = append(scanRefs, "&m."+f.Name)
	}
	updateParam := 1
	for _, f := range info.Fields {
		if f.Name == "ID" {
			continue
		}
		updateSet = append(updateSet, f.Snake+" = $"+fmt.Sprint(updateParam))
		updateParam++
		updateArgs = append(updateArgs, "m."+f.Name)
	}
	updateArgs = append(updateArgs, "m.ID")

	hasReturning := len(returningCols) > 0
	orderBy := "id"
	for _, f := range info.Fields {
		if f.Name == "CreatedAt" {
			orderBy = "created_at DESC"
			break
		}
	}
	data := map[string]interface{}{
		"Name":               info.Name,
		"OrderBy":            orderBy,
		"Lower":              info.Lower,
		"TableName":          info.TableName,
		"Plural":             info.Plural,
		"InsertColumns":      strings.Join(insertCols, ", "),
		"InsertPlaceholders": strings.Join(insertPh, ", "),
		"ReturningColumns":   strings.Join(returningCols, ", "),
		"InsertArgs":         strings.Join(insertArgs, ", "),
		"ScanArgs":            strings.Join(scanArgs, ", "),
		"SelectColumns":      strings.Join(selectCols, ", "),
		"ScanRefs":           strings.Join(scanRefs, ", "),
		"UpdateSet":          strings.Join(updateSet, ", "),
		"UpdateWhereParam":   updateParam,
		"UpdateArgs":         strings.Join(updateArgs, ", "),
		"HasReturning":       hasReturning,
	}
	t, _ := template.New("").Parse(tpl)
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}

func generateHandlers(path string, info *modelInfo) error {
	tpl := `package {{.Lower}}

import (
	"net/http"

	"github.com/mithril-framework/mithril/database/models"
	"github.com/mithril-framework/mithril/database/repositories"
	"github.com/mithril-framework/mithril/internal/acl"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Handlers holds {{.Lower}} CRUD handlers.
type Handlers struct {
	repo *repositories.{{.Name}}Repository
	acl  *acl.Service
}

// NewHandlers returns {{.Name}} CRUD handlers. acl may be nil.
func NewHandlers(repo *repositories.{{.Name}}Repository, aclSvc *acl.Service) *Handlers {
	return &Handlers{repo: repo, acl: aclSvc}
}

// List returns paginated {{.Plural}}. Query: limit (default 20), offset (default 0).
func (h *Handlers) List(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	list, err := h.repo.List(c.Context(), limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(list)
}

// Get returns one {{.Lower}} by id.
func (h *Handlers) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	m, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	return c.JSON(m)
}

// Create creates a {{.Lower}}. Request body: JSON matching models.{{.Name}}.
func (h *Handlers) Create(c *fiber.Ctx) error {
	var m models.{{.Name}}
	if err := c.BodyParser(&m); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := h.repo.Create(c.Context(), &m); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(http.StatusCreated).JSON(m)
}

// Update updates a {{.Lower}} by id.
func (h *Handlers) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var m models.{{.Name}}
	if err := c.BodyParser(&m); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	m.ID = id
	if err := h.repo.Update(c.Context(), &m); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(m)
}

// Delete deletes a {{.Lower}} by id.
func (h *Handlers) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.repo.Delete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
`
	t, _ := template.New("").Parse(tpl)
	var buf bytes.Buffer
	if err := t.Execute(&buf, info); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}

func generateRoutes(path string, info *modelInfo) error {
	tpl := `package routes

import (
	"github.com/mithril-framework/mithril/database/repositories"
	"github.com/mithril-framework/mithril/internal/acl"
	crudhandlers "github.com/mithril-framework/mithril/internal/crud/{{.Lower}}"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MountCrud{{.Name}}Routes registers {{.Plural}} CRUD on an existing /api group (middleware already applied).
func MountCrud{{.Name}}Routes(api fiber.Router, pool *pgxpool.Pool, aclSvc *acl.Service) {
	if pool == nil {
		return
	}
	repo := repositories.New{{.Name}}Repository(pool)
	h := crudhandlers.NewHandlers(repo, aclSvc)
	api.Get("/{{.Plural}}", acl.RequirePermission(aclSvc, "{{.Plural}}.view"), h.List)
	api.Get("/{{.Plural}}/:id", h.Get)
	api.Post("/{{.Plural}}", acl.RequirePermission(aclSvc, "{{.Plural}}.add"), h.Create)
	api.Put("/{{.Plural}}/:id", h.Update)
	api.Delete("/{{.Plural}}/:id", acl.RequirePermission(aclSvc, "{{.Plural}}.delete"), h.Delete)
}
`
	t, _ := template.New("").Parse(tpl)
	var buf bytes.Buffer
	if err := t.Execute(&buf, info); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}

func appendRegister(info *modelInfo) error {
	registerPath := "routes/register.go"
	content, err := os.ReadFile(registerPath)
	if err != nil {
		return err
	}
	line := "\t\tMountCrud" + info.Name + "Routes(api, pool, aclSvc)\n"
	if strings.Contains(string(content), "MountCrud"+info.Name+"Routes") {
		return nil
	}
	anchor := "\t\tMountCrudUserRoutes(api, pool, aclSvc)\n"
	newContent := bytes.Replace(content, []byte(anchor), []byte(anchor+line), 1)
	if bytes.Equal(content, newContent) {
		return fmt.Errorf("could not find %q in register.go", strings.TrimSpace(anchor))
	}
	return os.WriteFile(registerPath, newContent, 0644)
}
