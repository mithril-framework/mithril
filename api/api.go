// Package api provides typed HTTP handlers with automatic binding, validation and OpenAPI docs.
//
//	type GetBlogIn struct {
//	    ID uuid.UUID `path:"id"`
//	}
//	api.Get(r, "/blogs/:id", func(c fiber.Ctx, in *GetBlogIn) (*Blog, error) { ... },
//	    api.Op{Summary: "Get a blog"})
//
// Input structs combine path:"name" fields, query:"name" fields and JSON body fields; validate
// tags are enforced before the handler runs and errors come back as 422 with per-field details.
package api

import (
	"encoding"
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/mithril-framework/mithril/openapi"
)

// Error is an HTTP error with a stable machine-readable code.
type Error struct {
	Status  int          `json:"-"`
	Code    string       `json:"error"`
	Message string       `json:"message"`
	Fields  []FieldError `json:"fields,omitempty"`
}

func (e *Error) Error() string { return e.Message }

// Errorf builds an Error.
func Errorf(status int, code, format string, args ...any) *Error {
	return &Error{Status: status, Code: code, Message: fmt.Sprintf(format, args...)}
}

// Common errors.
var (
	ErrUnauthorized = &Error{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "authentication required"}
	ErrForbidden    = &Error{Status: http.StatusForbidden, Code: "forbidden", Message: "permission denied"}
	ErrNotFound     = &Error{Status: http.StatusNotFound, Code: "not_found", Message: "not found"}
)

// BadRequest returns a 400 error.
func BadRequest(msg string) *Error {
	return &Error{Status: http.StatusBadRequest, Code: "bad_request", Message: msg}
}

// WriteError renders err as JSON. Unknown errors become a generic 500 and are logged, so internal
// details (SQL, stack traces) never reach clients.
func WriteError(c fiber.Ctx, err error) error {
	var ae *Error
	if errors.As(err, &ae) {
		return c.Status(ae.Status).JSON(ae)
	}
	var fe *fiber.Error
	if errors.As(err, &fe) {
		return c.Status(fe.Code).JSON(&Error{Code: strings.ToLower(strings.ReplaceAll(http.StatusText(fe.Code), " ", "_")), Message: fe.Message})
	}
	rid, _ := c.Locals("requestid").(string)
	log.Printf("api: %s %s request_id=%s: %v", c.Method(), c.Path(), rid, err)
	return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error", "message": "internal server error", "request_id": rid})
}

// Op documents an operation.
type Op struct {
	Summary     string
	Description string
	Tags        []string
	ID          string // operationId; defaults to method+path
	Public      bool   // no bearer auth in the spec
	Status      int    // success status; default 200 (201 for POST, 204 for nil output)
}

// Router registers typed handlers on a Fiber router and documents them in Doc.
type Router struct {
	fiber  fiber.Router
	Doc    *openapi.Document
	prefix string
	tags   []string
	public bool
}

// NewRouter wraps r. prefix is the full path prefix of r (used for documentation, e.g. "/api").
func NewRouter(r fiber.Router, doc *openapi.Document, prefix string) *Router {
	return &Router{fiber: r, Doc: doc, prefix: strings.TrimRight(prefix, "/")}
}

// Fiber returns the underlying router for untyped routes.
func (r *Router) Fiber() fiber.Router { return r.fiber }

// Group returns a sub-router with a path prefix and middleware.
func (r *Router) Group(prefix string, handlers ...any) *Router {
	return &Router{fiber: r.fiber.Group(prefix, handlers...), Doc: r.Doc, prefix: r.prefix + prefix, tags: r.tags, public: r.public}
}

// WithTags returns a router whose operations default to tags.
func (r *Router) WithTags(tags ...string) *Router {
	cp := *r
	cp.tags = tags
	return &cp
}

// Public returns a router whose operations are documented without auth.
func (r *Router) Public() *Router {
	cp := *r
	cp.public = true
	return &cp
}

// Handler is a typed handler.
type Handler[In, Out any] func(c fiber.Ctx, in *In) (*Out, error)

// Get registers a typed GET route.
func Get[In, Out any](r *Router, path string, h Handler[In, Out], op Op, mw ...fiber.Handler) {
	Handle(r, http.MethodGet, path, h, op, mw...)
}

// Post registers a typed POST route.
func Post[In, Out any](r *Router, path string, h Handler[In, Out], op Op, mw ...fiber.Handler) {
	Handle(r, http.MethodPost, path, h, op, mw...)
}

// Put registers a typed PUT route.
func Put[In, Out any](r *Router, path string, h Handler[In, Out], op Op, mw ...fiber.Handler) {
	Handle(r, http.MethodPut, path, h, op, mw...)
}

// Patch registers a typed PATCH route.
func Patch[In, Out any](r *Router, path string, h Handler[In, Out], op Op, mw ...fiber.Handler) {
	Handle(r, http.MethodPatch, path, h, op, mw...)
}

// Delete registers a typed DELETE route.
func Delete[In, Out any](r *Router, path string, h Handler[In, Out], op Op, mw ...fiber.Handler) {
	Handle(r, http.MethodDelete, path, h, op, mw...)
}

// Empty is used as In or Out when there is nothing to bind or return.
type Empty struct{}

// Handle registers h for method and path and adds the OpenAPI operation.
func Handle[In, Out any](r *Router, method, path string, h Handler[In, Out], op Op, mw ...fiber.Handler) {
	inType := reflect.TypeOf((*In)(nil)).Elem()
	outType := reflect.TypeOf((*Out)(nil)).Elem()
	status := op.Status
	if status == 0 {
		switch {
		case outType == reflect.TypeOf(Empty{}):
			status = http.StatusNoContent
		case method == http.MethodPost:
			status = http.StatusCreated
		default:
			status = http.StatusOK
		}
	}
	hasBody := method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch

	handler := func(c fiber.Ctx) error {
		in := new(In)
		if err := bind(c, in, hasBody); err != nil {
			return WriteError(c, err)
		}
		if errs := Validate(in, nil); len(errs) > 0 {
			return WriteError(c, &Error{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "validation failed", Fields: errs})
		}
		out, err := h(c, in)
		if err != nil {
			return WriteError(c, err)
		}
		if status == http.StatusNoContent || out == nil {
			return c.SendStatus(status)
		}
		return c.Status(status).JSON(out)
	}
	// Fiber runs handlers in the order given: middleware first, the typed handler last.
	chain := make([]any, 0, len(mw)+1)
	for _, m := range mw {
		chain = append(chain, m)
	}
	chain = append(chain, fiber.Handler(handler))
	r.fiber.Add([]string{method}, path, chain[0], chain[1:]...)
	if r.Doc != nil {
		r.Doc.AddOperation(method, r.prefix+path, r.operation(method, path, inType, outType, status, hasBody, op))
	}
}

func (r *Router) operation(method, path string, inType, outType reflect.Type, status int, hasBody bool, op Op) openapi.Operation {
	o := openapi.Operation{
		OperationID: op.ID,
		Summary:     op.Summary,
		Description: op.Description,
		Tags:        op.Tags,
		Responses:   map[string]openapi.Response{},
	}
	if o.OperationID == "" {
		o.OperationID = operationID(method, r.prefix+path)
	}
	if len(o.Tags) == 0 {
		o.Tags = r.tags
	}
	if !op.Public && !r.public {
		o.Security = openapi.BearerAuth
	}
	if inType.Kind() == reflect.Struct {
		bodyFields := false
		for i := 0; i < inType.NumField(); i++ {
			f := inType.Field(i)
			if !f.IsExported() {
				continue
			}
			if name := f.Tag.Get("path"); name != "" {
				ps := r.Doc.SchemaForType(f.Type)
				openapi.ApplyTags(ps, f)
				o.Parameters = append(o.Parameters, openapi.Parameter{Name: name, In: "path", Required: true, Schema: ps, Description: f.Tag.Get("doc")})
				continue
			}
			if q := f.Tag.Get("query"); q != "" {
				name := strings.Split(q, ",")[0]
				ps := r.Doc.SchemaForType(f.Type)
				openapi.ApplyTags(ps, f)
				_, req := openapi.ParseRules(f.Tag.Get("validate"))["required"]
				o.Parameters = append(o.Parameters, openapi.Parameter{Name: name, In: "query", Required: req, Schema: ps, Description: f.Tag.Get("doc")})
				continue
			}
			if n, _ := openapi.FieldName(f); n != "" {
				bodyFields = true
			}
		}
		if hasBody && bodyFields {
			o.RequestBody = &openapi.RequestBody{Required: true, Content: map[string]openapi.MediaType{
				"application/json": {Schema: r.Doc.SchemaForType(inType)},
			}}
		}
	}
	resp := openapi.Response{Description: http.StatusText(status)}
	if status != http.StatusNoContent && outType != reflect.TypeOf(Empty{}) {
		resp.Content = map[string]openapi.MediaType{"application/json": {Schema: r.Doc.SchemaForType(outType)}}
	}
	o.Responses[strconv.Itoa(status)] = resp
	errSchema := r.Doc.SchemaForType(reflect.TypeOf(Error{}))
	errResp := func(desc string) openapi.Response {
		return openapi.Response{Description: desc, Content: map[string]openapi.MediaType{"application/json": {Schema: errSchema}}}
	}
	if hasBody || len(o.Parameters) > 0 {
		o.Responses["422"] = errResp("Validation error")
	}
	if o.Security != nil {
		o.Responses["401"] = errResp("Unauthorized")
		o.Responses["403"] = errResp("Forbidden")
	}
	return o
}

func operationID(method, path string) string {
	var b strings.Builder
	b.WriteString(strings.ToLower(method))
	for _, seg := range strings.Split(path, "/") {
		seg = strings.Trim(seg, ":{}?")
		if seg == "" {
			continue
		}
		b.WriteString(strings.ToUpper(seg[:1]))
		b.WriteString(seg[1:])
	}
	return strings.NewReplacer("-", "", "_", "", ".", "").Replace(b.String())
}

func bind(c fiber.Ctx, in any, hasBody bool) error {
	rv := reflect.ValueOf(in).Elem()
	if rv.Kind() != reflect.Struct {
		if hasBody && len(c.Body()) > 0 {
			if err := c.Bind().JSON(in); err != nil {
				return BadRequest("invalid JSON body")
			}
		}
		return nil
	}
	rt := rv.Type()
	hasQuery := false
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if name := f.Tag.Get("path"); name != "" {
			if err := setFromString(rv.Field(i), c.Params(name)); err != nil {
				return BadRequest("invalid path parameter " + name)
			}
		}
		if f.Tag.Get("query") != "" {
			hasQuery = true
		}
	}
	if hasBody && len(c.Body()) > 0 {
		// Decode into a copy so the body cannot override path parameters.
		saved := reflect.New(rt).Elem()
		saved.Set(rv)
		if err := c.Bind().JSON(in); err != nil {
			return BadRequest("invalid JSON body")
		}
		for i := 0; i < rt.NumField(); i++ {
			if rt.Field(i).Tag.Get("path") != "" || rt.Field(i).Tag.Get("query") != "" {
				rv.Field(i).Set(saved.Field(i))
			}
		}
	}
	if hasQuery {
		if err := c.Bind().Query(in); err != nil {
			return BadRequest("invalid query parameters")
		}
	}
	return nil
}

var textUnmarshaler = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

func setFromString(v reflect.Value, s string) error {
	if v.Kind() == reflect.Pointer {
		if s == "" {
			return nil
		}
		v.Set(reflect.New(v.Type().Elem()))
		v = v.Elem()
	}
	if v.Type() == reflect.TypeOf(uuid.UUID{}) {
		id, err := uuid.Parse(s)
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(id))
		return nil
	}
	if reflect.PointerTo(v.Type()).Implements(textUnmarshaler) {
		return v.Addr().Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(s))
	}
	switch v.Kind() {
	case reflect.String:
		v.SetString(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		v.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return err
		}
		v.SetUint(n)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		v.SetBool(b)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
		v.SetFloat(f)
	default:
		return fmt.Errorf("unsupported path parameter type %s", v.Type())
	}
	return nil
}

// PathPrefix returns the documented path prefix of the router (e.g. "/api").
func (r *Router) PathPrefix() string { return r.prefix }
