// Package resource turns a Go struct into a full REST resource with one call:
//
//	type Blog struct {
//	    ID        uuid.UUID `json:"id"`
//	    Title     string    `json:"title" validate:"required,max=200"`
//	    Content   string    `json:"content" mithril:"text"`
//	    AuthorID  uuid.UUID `json:"author_id" mithril:"owner"`
//	    CreatedAt time.Time `json:"created_at"`
//	    UpdatedAt time.Time `json:"updated_at"`
//	}
//
//	app.Register(resource.Define[Blog](resource.Options{Name: "blogs"}))
//
// This generates GET/POST /api/blogs and GET/PATCH/PUT/DELETE /api/blogs/:id with validation,
// the OpenAPI operations, the permission codenames, and an admin panel section.
//
// Permissions (superusers bypass all checks):
//
//	<name>.view        list/read every row (without it, owners see only their own rows)
//	<name>.add         create
//	<name>.change      update (with an owner field: update own rows only)
//	<name>.change_any  update any row            (only with an owner field)
//	<name>.delete      delete (with an owner field: delete own rows only)
//	<name>.delete_any  delete any row            (only with an owner field)
package resource

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mithril-framework/mithril/acl"
	"github.com/mithril-framework/mithril/api"
)

// Options configures a resource.
type Options struct {
	Name        string // URL segment and permission prefix, e.g. "blogs" (required)
	Table       string // database table (default: Name)
	Label       string // human label for the admin panel (default: Name)
	Description string // OpenAPI tag description
	OwnerField  string // Go field or column holding the owning user's id (alternative to mithril:"owner")
	OrderBy     string // column for default ordering, newest first (default: created_at if present, else the pk)
	MaxLimit    int    // maximum page size (default 100)
}

// Permission is a codename the resource uses.
type Permission struct {
	Codename    string `json:"codename"`
	Description string `json:"description"`
}

// FieldMeta describes a field for the admin UI and tooling.
type FieldMeta struct {
	Name     string `json:"name"`
	Column   string `json:"column"`
	Kind     string `json:"kind"` // string | text | email | integer | number | boolean | datetime | uuid | json
	ReadOnly bool   `json:"readonly"`
	Required bool   `json:"required"`
}

// Meta describes a resource for the admin UI, MCP tools and code generators.
type Meta struct {
	Name        string       `json:"name"`
	Label       string       `json:"label"`
	Table       string       `json:"table"`
	PK          string       `json:"pk"`
	Owner       string       `json:"owner,omitempty"`
	Path        string       `json:"path"`
	Fields      []FieldMeta  `json:"fields"`
	Permissions []Permission `json:"permissions"`
}

// Deps are runtime dependencies supplied by the app.
type Deps struct {
	DB  *pgxpool.Pool
	ACL *acl.Service
}

// Resource is the type-erased view the app, admin panel and MCP server use.
type Resource interface {
	Meta() Meta
	Permissions() []Permission
	// Mount registers the REST routes on r (an /api router with auth middleware applied).
	Mount(r *api.Router, deps func() Deps)
	// Admin returns untyped operations for the admin panel.
	Admin(deps func() Deps) AdminOps
}

// AdminOps are JSON-in/JSON-out operations used by the admin API. Permission checks use the
// "any row" codenames (view, add, change/change_any, delete/delete_any).
type AdminOps interface {
	List(c fiber.Ctx, limit, offset int) (any, error)
	Get(c fiber.Ctx, id string) (any, error)
	Create(c fiber.Ctx, body []byte) (any, error)
	Update(c fiber.Ctx, id string, body []byte) (any, error)
	Delete(c fiber.Ctx, id string) error
}

// R is a resource over the struct type T.
type R[T any] struct {
	opts    Options
	typ     reflect.Type
	fields  []*field
	pk      *field
	owner   *field
	orderBy *field
	byJSON  map[string]*field
	updated *field // updated_at column, maintained on update
}

// Define builds a resource for T. It panics on invalid definitions (they are programming errors
// that must fail at startup, not at request time).
func Define[T any](opts Options) *R[T] {
	r, err := New[T](opts)
	if err != nil {
		panic(err)
	}
	return r
}

// New is Define returning an error.
func New[T any](opts Options) (*R[T], error) {
	if opts.Name == "" {
		return nil, errors.New("resource: Options.Name is required")
	}
	if strings.ContainsAny(opts.Name, "/ .:") {
		return nil, fmt.Errorf("resource: invalid name %q", opts.Name)
	}
	if opts.Table == "" {
		opts.Table = opts.Name
	}
	if opts.Label == "" {
		opts.Label = strings.ToUpper(opts.Name[:1]) + opts.Name[1:]
	}
	if opts.MaxLimit <= 0 {
		opts.MaxLimit = 100
	}
	t := reflect.TypeOf((*T)(nil)).Elem()
	fields, err := parseFields(t, opts.OwnerField)
	if err != nil {
		return nil, fmt.Errorf("resource %s: %w", opts.Name, err)
	}
	r := &R[T]{opts: opts, typ: t, fields: fields, byJSON: map[string]*field{}}
	for _, f := range fields {
		if f.PK {
			r.pk = f
		}
		if f.Owner {
			r.owner = f
		}
		if f.Column == "updated_at" {
			r.updated = f
		}
		if f.JSON != "" {
			r.byJSON[f.JSON] = f
		}
	}
	order := opts.OrderBy
	if order == "" {
		order = "created_at"
	}
	for _, f := range fields {
		if f.Column == order || f.GoName == order {
			r.orderBy = f
		}
	}
	if r.orderBy == nil {
		if opts.OrderBy != "" {
			return nil, fmt.Errorf("resource %s: OrderBy column %q not found", opts.Name, opts.OrderBy)
		}
		r.orderBy = r.pk
	}
	return r, nil
}

// Name returns the resource name.
func (r *R[T]) Name() string { return r.opts.Name }

func (r *R[T]) perm(action string) string { return r.opts.Name + "." + action }

// Permissions lists the codenames this resource checks.
func (r *R[T]) Permissions() []Permission {
	label := r.opts.Label
	ps := []Permission{
		{r.perm("view"), "List and view all " + label},
		{r.perm("add"), "Create " + label},
	}
	if r.owner != nil {
		ps = append(ps,
			Permission{r.perm("change"), "Update own " + label},
			Permission{r.perm("change_any"), "Update any " + label},
			Permission{r.perm("delete"), "Delete own " + label},
			Permission{r.perm("delete_any"), "Delete any " + label},
		)
	} else {
		ps = append(ps,
			Permission{r.perm("change"), "Update " + label},
			Permission{r.perm("delete"), "Delete " + label},
		)
	}
	return ps
}

// Meta describes the resource.
func (r *R[T]) Meta() Meta {
	m := Meta{Name: r.opts.Name, Label: r.opts.Label, Table: r.opts.Table, PK: r.pk.JSON,
		Path: "/api/" + r.opts.Name, Permissions: r.Permissions()}
	if r.owner != nil {
		m.Owner = r.owner.JSON
	}
	for _, f := range r.fields {
		if f.JSON == "" {
			continue
		}
		m.Fields = append(m.Fields, FieldMeta{Name: f.JSON, Column: f.Column, Kind: f.kind(),
			ReadOnly: !f.writable(), Required: hasKey(f.Rules, "required")})
	}
	return m
}

// ---- SQL --------------------------------------------------------------------------------------

func (r *R[T]) table() string { return pgx.Identifier{r.opts.Table}.Sanitize() }

func (r *R[T]) columns() string {
	cols := make([]string, len(r.fields))
	for i, f := range r.fields {
		cols[i] = f.quoted()
	}
	return strings.Join(cols, ", ")
}

func (r *R[T]) scanTargets(v *T) []any {
	rv := reflect.ValueOf(v).Elem()
	dest := make([]any, len(r.fields))
	for i, f := range r.fields {
		dest[i] = rv.FieldByIndex(f.Index).Addr().Interface()
	}
	return dest
}

func (r *R[T]) collect(rows pgx.Rows) ([]T, error) {
	defer rows.Close()
	out := []T{}
	for rows.Next() {
		var v T
		if err := rows.Scan(r.scanTargets(&v)...); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// Page is a paginated list response.
type Page[T any] struct {
	Items  []T   `json:"items"`
	Total  int64 `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
}

func (r *R[T]) list(ctx context.Context, db *pgxpool.Pool, owner *uuid.UUID, limit, offset int) (*Page[T], error) {
	where, args := "", []any{}
	if owner != nil {
		where = " WHERE " + r.owner.quoted() + " = $1"
		args = append(args, *owner)
	}
	var total int64
	if err := db.QueryRow(ctx, "SELECT count(*) FROM "+r.table()+where, args...).Scan(&total); err != nil {
		return nil, err
	}
	q := fmt.Sprintf("SELECT %s FROM %s%s ORDER BY %s DESC, %s DESC LIMIT $%d OFFSET $%d",
		r.columns(), r.table(), where, r.orderBy.quoted(), r.pk.quoted(), len(args)+1, len(args)+2)
	rows, err := db.Query(ctx, q, append(args, limit, offset)...)
	if err != nil {
		return nil, err
	}
	items, err := r.collect(rows)
	if err != nil {
		return nil, err
	}
	return &Page[T]{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}

func (r *R[T]) get(ctx context.Context, db *pgxpool.Pool, id any) (*T, error) {
	var v T
	err := db.QueryRow(ctx, "SELECT "+r.columns()+" FROM "+r.table()+" WHERE "+r.pk.quoted()+" = $1", id).
		Scan(r.scanTargets(&v)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, api.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *R[T]) insert(ctx context.Context, db *pgxpool.Pool, v *T, present map[string]bool, ownerID *uuid.UUID) (*T, error) {
	rv := reflect.ValueOf(v).Elem()
	var cols, ph []string
	var args []any
	add := func(f *field, val any) {
		cols = append(cols, f.quoted())
		args = append(args, val)
		ph = append(ph, "$"+strconv.Itoa(len(args)))
	}
	if isUUID(r.pk.Type) {
		add(r.pk, uuid.New())
	}
	if r.owner != nil && ownerID != nil {
		add(r.owner, *ownerID)
	}
	for _, f := range r.fields {
		if f.writable() && present[f.JSON] {
			add(f, rv.FieldByIndex(f.Index).Interface())
		}
	}
	q := "INSERT INTO " + r.table()
	if len(cols) == 0 {
		q += " DEFAULT VALUES"
	} else {
		q += " (" + strings.Join(cols, ", ") + ") VALUES (" + strings.Join(ph, ", ") + ")"
	}
	q += " RETURNING " + r.columns()
	var out T
	if err := db.QueryRow(ctx, q, args...).Scan(r.scanTargets(&out)...); err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *R[T]) update(ctx context.Context, db *pgxpool.Pool, id any, v *T, present map[string]bool) (*T, error) {
	rv := reflect.ValueOf(v).Elem()
	var sets []string
	var args []any
	for _, f := range r.fields {
		if f.writable() && present[f.JSON] {
			args = append(args, rv.FieldByIndex(f.Index).Interface())
			sets = append(sets, f.quoted()+" = $"+strconv.Itoa(len(args)))
		}
	}
	if len(sets) == 0 {
		return r.get(ctx, db, id)
	}
	if r.updated != nil {
		sets = append(sets, r.updated.quoted()+" = NOW()")
	}
	args = append(args, id)
	q := fmt.Sprintf("UPDATE %s SET %s WHERE %s = $%d RETURNING %s",
		r.table(), strings.Join(sets, ", "), r.pk.quoted(), len(args), r.columns())
	var out T
	err := db.QueryRow(ctx, q, args...).Scan(r.scanTargets(&out)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, api.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *R[T]) delete(ctx context.Context, db *pgxpool.Pool, id any) error {
	tag, err := db.Exec(ctx, "DELETE FROM "+r.table()+" WHERE "+r.pk.quoted()+" = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return api.ErrNotFound
	}
	return nil
}

// ---- input --------------------------------------------------------------------------------------

func (r *R[T]) parseID(s string) (any, error) {
	if isUUID(r.pk.Type) {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, api.BadRequest("invalid id")
		}
		return id, nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil, api.BadRequest("invalid id")
	}
	return n, nil
}

// decode reads only writable fields from a JSON object (read-only keys such as id, owner or
// timestamps are ignored, so clients cannot mass-assign them) and validates them.
func (r *R[T]) decode(body []byte, partial bool) (*T, map[string]bool, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, nil, api.BadRequest("request body must be a JSON object")
	}
	v := new(T)
	rv := reflect.ValueOf(v).Elem()
	present := map[string]bool{}
	var fieldErrs []api.FieldError
	for key, msg := range raw {
		f, ok := r.byJSON[key]
		if !ok || !f.writable() {
			continue
		}
		target := rv.FieldByIndex(f.Index).Addr().Interface()
		if err := json.Unmarshal(msg, target); err != nil {
			fieldErrs = append(fieldErrs, api.FieldError{Field: key, Rule: "type", Message: key + " has the wrong type"})
			continue
		}
		present[key] = true
	}
	only := map[string]bool{}
	for _, f := range r.fields {
		if f.writable() && (!partial || present[f.JSON]) {
			only[f.JSON] = true
		}
	}
	fieldErrs = append(fieldErrs, api.Validate(v, only)...)
	if len(fieldErrs) > 0 {
		return nil, nil, &api.Error{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "validation failed", Fields: fieldErrs}
	}
	return v, present, nil
}

// dbError maps constraint violations to client errors.
func dbError(err error) error {
	var pg *pgconn.PgError
	if !errors.As(err, &pg) {
		return err
	}
	switch pg.Code {
	case "23505":
		return &api.Error{Status: http.StatusConflict, Code: "conflict", Message: "a record with this value already exists"}
	case "23503":
		return &api.Error{Status: http.StatusBadRequest, Code: "invalid_reference", Message: "referenced record does not exist"}
	case "23502":
		return &api.Error{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: pg.ColumnName + " is required",
			Fields: []api.FieldError{{Field: pg.ColumnName, Rule: "required", Message: pg.ColumnName + " is required"}}}
	case "23514", "22001", "22P02":
		return &api.Error{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "invalid value"}
	}
	return err
}
