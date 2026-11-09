package resource

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/mithril-framework/mithril/acl"
	"github.com/mithril-framework/mithril/api"
	"github.com/mithril-framework/mithril/openapi"
)

type caller struct {
	id   uuid.UUID
	deps Deps
}

func callerFrom(c fiber.Ctx, deps Deps) (*caller, error) {
	if deps.DB == nil {
		return nil, &api.Error{Status: http.StatusServiceUnavailable, Code: "service_unavailable", Message: "database unavailable"}
	}
	id, err := acl.CurrentUserID(c)
	if err != nil {
		return nil, api.ErrUnauthorized
	}
	return &caller{id: id, deps: deps}, nil
}

func (p *caller) can(c fiber.Ctx, codename string) (bool, error) {
	if p.deps.ACL == nil {
		return false, nil
	}
	return p.deps.ACL.HasPermission(c, c, p.id, codename, acl.IsSuperuserLocal(c))
}

func (r *R[T]) ownerOf(v *T) uuid.UUID {
	return reflect.ValueOf(v).Elem().FieldByIndex(r.owner.Index).Interface().(uuid.UUID)
}

// mayModify applies the change/delete rules for action "change" or "delete".
func (r *R[T]) mayModify(c fiber.Ctx, p *caller, action string, row *T) (bool, error) {
	if r.owner == nil {
		return p.can(c, r.perm(action))
	}
	if ok, err := p.can(c, r.perm(action+"_any")); err != nil || ok {
		return ok, err
	}
	if r.ownerOf(row) != p.id {
		return false, nil
	}
	return p.can(c, r.perm(action))
}

func (r *R[T]) mayView(c fiber.Ctx, p *caller, row *T) (bool, error) {
	if ok, err := p.can(c, r.perm("view")); err != nil || ok {
		return ok, err
	}
	return r.owner != nil && r.ownerOf(row) == p.id, nil
}

func (r *R[T]) clampPage(c fiber.Ctx) (int, int) {
	limit := fiber.Query[int](c, "limit", 20)
	offset := fiber.Query[int](c, "offset", 0)
	if limit <= 0 {
		limit = 20
	}
	if limit > r.opts.MaxLimit {
		limit = r.opts.MaxLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// Mount registers the REST routes and their OpenAPI documentation.
func (r *R[T]) Mount(rt *api.Router, deps func() Deps) {
	base := "/" + r.opts.Name
	f := rt.Fiber()
	wrap := func(h func(c fiber.Ctx, p *caller) (any, int, error)) fiber.Handler {
		return func(c fiber.Ctx) error {
			p, err := callerFrom(c, deps())
			if err != nil {
				return api.WriteError(c, err)
			}
			out, status, err := h(c, p)
			if err != nil {
				return api.WriteError(c, dbError(err))
			}
			if out == nil {
				return c.SendStatus(status)
			}
			return c.Status(status).JSON(out)
		}
	}

	f.Get(base, wrap(func(c fiber.Ctx, p *caller) (any, int, error) {
		limit, offset := r.clampPage(c)
		all, err := p.can(c, r.perm("view"))
		if err != nil {
			return nil, 0, err
		}
		var owner *uuid.UUID
		if !all {
			if r.owner == nil {
				return nil, 0, api.ErrForbidden
			}
			owner = &p.id
		}
		page, err := r.list(c, p.deps.DB, owner, limit, offset)
		return page, http.StatusOK, err
	}))

	f.Post(base, wrap(func(c fiber.Ctx, p *caller) (any, int, error) {
		if ok, err := p.can(c, r.perm("add")); err != nil || !ok {
			return nil, 0, firstErr(err, api.ErrForbidden)
		}
		v, present, err := r.decode(c.Body(), false)
		if err != nil {
			return nil, 0, err
		}
		out, err := r.insert(c, p.deps.DB, v, present, &p.id)
		return out, http.StatusCreated, err
	}))

	f.Get(base+"/:id", wrap(func(c fiber.Ctx, p *caller) (any, int, error) {
		id, err := r.parseID(c.Params("id"))
		if err != nil {
			return nil, 0, err
		}
		row, err := r.get(c, p.deps.DB, id)
		if err != nil {
			return nil, 0, err
		}
		if ok, err := r.mayView(c, p, row); err != nil || !ok {
			return nil, 0, firstErr(err, api.ErrNotFound) // do not reveal that the row exists
		}
		return row, http.StatusOK, nil
	}))

	update := wrap(func(c fiber.Ctx, p *caller) (any, int, error) {
		id, err := r.parseID(c.Params("id"))
		if err != nil {
			return nil, 0, err
		}
		row, err := r.get(c, p.deps.DB, id)
		if err != nil {
			return nil, 0, err
		}
		if ok, err := r.mayModify(c, p, "change", row); err != nil || !ok {
			if visible, _ := r.mayView(c, p, row); !visible {
				return nil, 0, api.ErrNotFound
			}
			return nil, 0, firstErr(err, api.ErrForbidden)
		}
		v, present, err := r.decode(c.Body(), true)
		if err != nil {
			return nil, 0, err
		}
		out, err := r.update(c, p.deps.DB, id, v, present)
		return out, http.StatusOK, err
	})
	f.Patch(base+"/:id", update)
	f.Put(base+"/:id", update)

	f.Delete(base+"/:id", wrap(func(c fiber.Ctx, p *caller) (any, int, error) {
		id, err := r.parseID(c.Params("id"))
		if err != nil {
			return nil, 0, err
		}
		row, err := r.get(c, p.deps.DB, id)
		if err != nil {
			return nil, 0, err
		}
		if ok, err := r.mayModify(c, p, "delete", row); err != nil || !ok {
			if visible, _ := r.mayView(c, p, row); !visible {
				return nil, 0, api.ErrNotFound
			}
			return nil, 0, firstErr(err, api.ErrForbidden)
		}
		return nil, http.StatusNoContent, r.delete(c, p.deps.DB, id)
	}))

	if rt.Doc != nil {
		r.document(rt, base)
	}
}

func firstErr(err, fallback error) error {
	if err != nil {
		return err
	}
	return fallback
}

func (r *R[T]) document(rt *api.Router, base string) {
	d := rt.Doc
	name := r.typ.Name()
	if name == "" {
		name = strings.ToUpper(r.opts.Name[:1]) + r.opts.Name[1:]
	}
	props := map[string]*openapi.Schema{}
	var required []string
	for _, f := range r.fields {
		if f.JSON == "" {
			continue
		}
		sf := r.typ.FieldByIndex(f.Index)
		s := d.SchemaForType(f.Type)
		openapi.ApplyTags(s, sf)
		if !f.writable() {
			s.ReadOnly = true
		}
		props[f.JSON] = s
		if hasKey(f.Rules, "required") {
			required = append(required, f.JSON)
		}
	}
	item := d.SetComponent(name, &openapi.Schema{Type: "object", Properties: props, Required: required})
	page := &openapi.Schema{Type: "object", Properties: map[string]*openapi.Schema{
		"items":  {Type: "array", Items: item},
		"total":  {Type: "integer", Format: "int64"},
		"limit":  {Type: "integer"},
		"offset": {Type: "integer"},
	}}
	errSchema := d.SchemaForType(reflect.TypeOf(api.Error{}))
	json := func(s *openapi.Schema) map[string]openapi.MediaType {
		return map[string]openapi.MediaType{"application/json": {Schema: s}}
	}
	errs := func(codes ...string) map[string]openapi.Response {
		m := map[string]openapi.Response{}
		for _, c := range codes {
			m[c] = openapi.Response{Description: http.StatusText(atoi(c)), Content: json(errSchema)}
		}
		return m
	}
	tag := []string{r.opts.Name}
	full := rt.PathPrefix() + base
	idParam := []openapi.Parameter{{Name: "id", In: "path", Required: true, Schema: d.SchemaForType(r.pk.Type)}}
	body := &openapi.RequestBody{Required: true, Content: json(item)}
	op := func(id, summary string, params []openapi.Parameter, rb *openapi.RequestBody, ok string, okBody *openapi.Schema, e ...string) openapi.Operation {
		resp := errs(append([]string{"401", "403"}, e...)...)
		r := openapi.Response{Description: http.StatusText(atoi(ok))}
		if okBody != nil {
			r.Content = json(okBody)
		}
		resp[ok] = r
		return openapi.Operation{OperationID: id, Summary: summary, Tags: tag, Parameters: params, RequestBody: rb,
			Responses: resp, Security: openapi.BearerAuth}
	}
	pageParams := []openapi.Parameter{
		{Name: "limit", In: "query", Schema: &openapi.Schema{Type: "integer"}, Description: "Page size (max " + itoa(r.opts.MaxLimit) + ")"},
		{Name: "offset", In: "query", Schema: &openapi.Schema{Type: "integer"}},
	}
	listDesc := "Requires " + r.perm("view")
	if r.owner != nil {
		listDesc += "; without it only your own rows are returned"
	}
	lo := op("list"+name, "List "+r.opts.Label, pageParams, nil, "200", page)
	lo.Description = listDesc
	d.AddOperation("GET", full, lo)
	co := op("create"+name, "Create "+r.opts.Label, nil, body, "201", item, "409", "422")
	co.Description = "Requires " + r.perm("add") + ". Read-only fields are ignored."
	d.AddOperation("POST", full, co)
	d.AddOperation("GET", full+"/:id", op("get"+name, "Get "+r.opts.Label, idParam, nil, "200", item, "404"))
	patch := op("update"+name, "Update "+r.opts.Label+" (partial)", idParam, body, "200", item, "404", "409", "422")
	patch.Description = "Only the fields sent are changed. Requires " + r.perm("change")
	if r.owner != nil {
		patch.Description += " (own rows) or " + r.perm("change_any")
	}
	d.AddOperation("PATCH", full+"/:id", patch)
	put := patch
	put.OperationID = "replace" + name
	put.Summary = "Update " + r.opts.Label + " (same as PATCH)"
	d.AddOperation("PUT", full+"/:id", put)
	del := op("delete"+name, "Delete "+r.opts.Label, idParam, nil, "204", nil, "404")
	del.Description = "Requires " + r.perm("delete")
	if r.owner != nil {
		del.Description += " (own rows) or " + r.perm("delete_any")
	}
	d.AddOperation("DELETE", full+"/:id", del)
}

// ---- admin --------------------------------------------------------------------------------------

type adminOps[T any] struct {
	r    *R[T]
	deps func() Deps
}

// Admin returns the admin-panel operations.
func (r *R[T]) Admin(deps func() Deps) AdminOps { return &adminOps[T]{r: r, deps: deps} }

func (a *adminOps[T]) require(c fiber.Ctx, codenames ...string) (*caller, error) {
	p, err := callerFrom(c, a.deps())
	if err != nil {
		return nil, err
	}
	for _, code := range codenames {
		ok, err := p.can(c, code)
		if err != nil {
			return nil, err
		}
		if ok {
			return p, nil
		}
	}
	return nil, &api.Error{Status: http.StatusForbidden, Code: "forbidden", Message: "missing permission: " + codenames[0]}
}

func (a *adminOps[T]) anyCode(action string) string {
	if a.r.owner != nil {
		return a.r.perm(action + "_any")
	}
	return a.r.perm(action)
}

func (a *adminOps[T]) List(c fiber.Ctx, limit, offset int) (any, error) {
	p, err := a.require(c, a.r.perm("view"))
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > a.r.opts.MaxLimit {
		limit = a.r.opts.MaxLimit
	}
	if offset < 0 {
		offset = 0
	}
	return a.r.list(c, p.deps.DB, nil, limit, offset)
}

func (a *adminOps[T]) Get(c fiber.Ctx, id string) (any, error) {
	p, err := a.require(c, a.r.perm("view"))
	if err != nil {
		return nil, err
	}
	pk, err := a.r.parseID(id)
	if err != nil {
		return nil, err
	}
	return a.r.get(c, p.deps.DB, pk)
}

func (a *adminOps[T]) Create(c fiber.Ctx, body []byte) (any, error) {
	p, err := a.require(c, a.r.perm("add"))
	if err != nil {
		return nil, err
	}
	v, present, err := a.r.decode(body, false)
	if err != nil {
		return nil, err
	}
	out, err := a.r.insert(c, p.deps.DB, v, present, &p.id)
	return out, dbErrOrNil(err)
}

func (a *adminOps[T]) Update(c fiber.Ctx, id string, body []byte) (any, error) {
	p, err := a.require(c, a.anyCode("change"))
	if err != nil {
		return nil, err
	}
	pk, err := a.r.parseID(id)
	if err != nil {
		return nil, err
	}
	v, present, err := a.r.decode(body, true)
	if err != nil {
		return nil, err
	}
	out, err := a.r.update(c, p.deps.DB, pk, v, present)
	return out, dbErrOrNil(err)
}

func (a *adminOps[T]) Delete(c fiber.Ctx, id string) error {
	p, err := a.require(c, a.anyCode("delete"))
	if err != nil {
		return err
	}
	pk, err := a.r.parseID(id)
	if err != nil {
		return err
	}
	return dbErrOrNil(a.r.delete(c, p.deps.DB, pk))
}

func dbErrOrNil(err error) error {
	if err == nil {
		return nil
	}
	return dbError(err)
}
