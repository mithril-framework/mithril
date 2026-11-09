package users

import (
	"reflect"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/mithril-framework/mithril/acl"
	"github.com/mithril-framework/mithril/api"
	"github.com/mithril-framework/mithril/database/repositories"
	"github.com/mithril-framework/mithril/openapi"
)

// User is the public representation of a user (documentation type for the API).
type User struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email" validate:"email"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	IsActive    bool      `json:"is_active"`
	IsSuperuser bool      `json:"is_superuser"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Mount registers /users routes on r (an /api router with JWT + claims middleware applied).
// repo returns the current repository (nil while the database is unavailable).
func Mount(r *api.Router, repo func() *repositories.UserRepository, aclSvc func() *acl.Service) {
	h := func(fn func(*Handlers) fiber.Handler) fiber.Handler {
		return func(c fiber.Ctx) error {
			ur := repo()
			if ur == nil {
				return api.WriteError(c, api.Errorf(fiber.StatusServiceUnavailable, "service_unavailable", "database unavailable"))
			}
			return fn(NewHandlers(ur, aclSvc()))(c)
		}
	}
	perm := func(code string) fiber.Handler {
		return func(c fiber.Ctx) error { return acl.RequirePermission(aclSvc(), code)(c) }
	}
	f := r.Fiber()
	f.Get("/users", perm("users.view"), h(func(x *Handlers) fiber.Handler { return x.List }))
	f.Get("/users/:id", h(func(x *Handlers) fiber.Handler { return x.Get }))
	f.Post("/users", perm("users.add"), h(func(x *Handlers) fiber.Handler { return x.Create }))
	f.Put("/users/:id", h(func(x *Handlers) fiber.Handler { return x.Update }))
	f.Patch("/users/:id", h(func(x *Handlers) fiber.Handler { return x.Update }))
	f.Delete("/users/:id", perm("users.delete"), h(func(x *Handlers) fiber.Handler { return x.Delete }))

	if r.Doc != nil {
		document(r)
	}
}

func document(r *api.Router) {
	d, base := r.Doc, r.PathPrefix()+"/users"
	user := d.SchemaForType(reflect.TypeOf(User{}))
	errS := d.SchemaForType(reflect.TypeOf(api.Error{}))
	js := func(s *openapi.Schema) map[string]openapi.MediaType {
		return map[string]openapi.MediaType{"application/json": {Schema: s}}
	}
	resp := func(code, desc string, s *openapi.Schema) map[string]openapi.Response {
		m := map[string]openapi.Response{
			"401": {Description: "Unauthorized", Content: js(errS)},
			"403": {Description: "Forbidden", Content: js(errS)},
		}
		rr := openapi.Response{Description: desc}
		if s != nil {
			rr.Content = js(s)
		}
		m[code] = rr
		return m
	}
	tags := []string{"users"}
	sec := openapi.BearerAuth
	d.AddOperation("GET", base, openapi.Operation{OperationID: "listUsers", Summary: "List users", Tags: tags, Security: sec,
		Description: "Requires users.view.",
		Parameters: []openapi.Parameter{
			{Name: "limit", In: "query", Schema: &openapi.Schema{Type: "integer"}},
			{Name: "offset", In: "query", Schema: &openapi.Schema{Type: "integer"}},
		},
		Responses: resp("200", "OK", &openapi.Schema{Type: "array", Items: user})})
	d.AddOperation("POST", base, openapi.Operation{OperationID: "createUser", Summary: "Create a user", Tags: tags, Security: sec,
		Description: "Requires users.add. Only a superuser may set is_superuser.",
		RequestBody: &openapi.RequestBody{Required: true, Content: js(d.SchemaForType(reflect.TypeOf(createUserRequest{})))},
		Responses:   resp("201", "Created", user)})
	d.AddOperation("GET", base+"/:id", openapi.Operation{OperationID: "getUser", Summary: "Get a user", Tags: tags, Security: sec,
		Description: "Yourself, or any user with users.view.", Responses: resp("200", "OK", user)})
	upd := openapi.Operation{OperationID: "updateUser", Summary: "Update a user (partial)", Tags: tags, Security: sec,
		Description: "Yourself (profile fields; email/password need current_password) or users.change on non-superusers. Only superusers change is_superuser.",
		RequestBody: &openapi.RequestBody{Required: true, Content: js(d.SchemaForType(reflect.TypeOf(updateUserRequest{})))},
		Responses:   resp("200", "OK", user)}
	d.AddOperation("PATCH", base+"/:id", upd)
	upd.OperationID = "replaceUser"
	d.AddOperation("PUT", base+"/:id", upd)
	d.AddOperation("DELETE", base+"/:id", openapi.Operation{OperationID: "deleteUser", Summary: "Delete a user", Tags: tags, Security: sec,
		Description: "Requires users.delete; superuser accounts only by superusers.", Responses: resp("204", "No Content", nil)})
}
