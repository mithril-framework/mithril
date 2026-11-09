// Package admin serves the Django-style admin panel: an embedded single-page UI at /admin and a
// JSON API at /admin/api for users, roles, permissions and every registered resource.
package admin

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/mithril-framework/mithril/acl"
	"github.com/mithril-framework/mithril/api"
	"github.com/mithril-framework/mithril/database/repositories"
	"github.com/mithril-framework/mithril/resource"
)

//go:embed ui/*
var uiFS embed.FS

// Config wires the admin panel.
type Config struct {
	JWT jwtware.Config
	// Accessors are called per request: the database may connect after startup.
	ACL       func() *acl.Service
	ACLRepo   func() *repositories.ACLRepository
	Users     func() *repositories.UserRepository
	Resources func() []resource.Resource
	Deps      func() resource.Deps
	// RequireDB is middleware that answers 503 while the database is unavailable.
	RequireDB fiber.Handler
}

// Mount registers /admin (UI) and /admin/api (JSON) on app.
func Mount(app *fiber.App, cfg Config) {
	ui, _ := fs.Sub(uiFS, "ui")

	app.Use("/admin", func(c fiber.Ctx) error {
		if c.Method() == fiber.MethodGet && c.Path() == "/admin" {
			return c.Redirect().Status(fiber.StatusFound).To("/admin/")
		}
		if (c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead) || strings.HasPrefix(c.Path(), "/admin/api") {
			return c.Next()
		}
		// Real asset (admin.js, admin.css) or the SPA shell for client-side routes.
		rest := strings.Trim(strings.TrimPrefix(c.Path(), "/admin"), "/")
		name := path.Clean(rest)
		if rest == "" || strings.HasPrefix(name, "..") {
			name = "index.html"
		}
		data, err := fs.ReadFile(ui, name)
		if err != nil {
			name = "index.html"
			data, _ = fs.ReadFile(ui, name)
		}
		c.Set("X-Frame-Options", "DENY")
		c.Set("Cache-Control", "no-cache")
		c.Type(path.Ext(name))
		return c.Send(data)
	})

	// hh adapts a Handlers method to a fiber.Handler bound to the current repositories.
	hh := func(m func(*Handlers, fiber.Ctx) error) fiber.Handler {
		return func(c fiber.Ctx) error { return m(NewHandlers(cfg.ACLRepo(), cfg.Users()), c) }
	}
	h := struct {
		ListPermissions, ListRoles, RolePermissions, UserRoles, UserDirectPermissions,
		ListUsers, GetUser, CreateUser, UpdateUser, DeleteUser,
		CreatePermission, DeletePermission, CreateRole, DeleteRole, AssignRole, RevokeRole,
		AssignPermissionRole, RevokePermissionRole, AssignPermissionUser, RevokePermissionUser fiber.Handler
	}{
		hh((*Handlers).ListPermissions), hh((*Handlers).ListRoles), hh((*Handlers).RolePermissions),
		hh((*Handlers).UserRoles), hh((*Handlers).UserDirectPermissions),
		hh((*Handlers).ListUsers), hh((*Handlers).GetUser), hh((*Handlers).CreateUser),
		hh((*Handlers).UpdateUser), hh((*Handlers).DeleteUser),
		hh((*Handlers).CreatePermission), hh((*Handlers).DeletePermission), hh((*Handlers).CreateRole),
		hh((*Handlers).DeleteRole), hh((*Handlers).AssignRole), hh((*Handlers).RevokeRole),
		hh((*Handlers).AssignPermissionRole), hh((*Handlers).RevokePermissionRole),
		hh((*Handlers).AssignPermissionUser), hh((*Handlers).RevokePermissionUser),
	}
	r := &resourceHandlers{list: cfg.Resources, deps: cfg.Deps}

	g := app.Group("/admin/api")
	if cfg.RequireDB != nil {
		g.Use(cfg.RequireDB)
	}
	g.Use(jwtware.New(cfg.JWT))
	g.Use(acl.JWTClaimsMiddleware())
	g.Use(func(c fiber.Ctx) error { return acl.RequireAdminAccess(cfg.ACL())(c) })

	g.Get("/meta", r.meta)

	g.Get("/permissions", h.ListPermissions)
	g.Get("/roles", h.ListRoles)
	g.Get("/roles/:name/permissions", h.RolePermissions)
	g.Get("/users/:id/roles", h.UserRoles)
	g.Get("/users/:id/permissions", h.UserDirectPermissions)

	// Users: handlers additionally require a (DB) superuser for anything touching superuser
	// accounts or the is_superuser flag.
	g.Get("/resources/users", h.ListUsers)
	g.Get("/resources/users/:id", h.GetUser)
	g.Post("/resources/users", h.CreateUser)
	g.Put("/resources/users/:id", h.UpdateUser)
	g.Delete("/resources/users/:id", h.DeleteUser)

	// Registered resources; each operation also checks the resource's own permissions.
	g.Get("/r/:resource", r.handle(func(c fiber.Ctx, ops resource.AdminOps) (any, int, error) {
		v, err := ops.List(c, fiber.Query[int](c, "limit", 50), fiber.Query[int](c, "offset", 0))
		return v, http.StatusOK, err
	}))
	g.Get("/r/:resource/:id", r.handle(func(c fiber.Ctx, ops resource.AdminOps) (any, int, error) {
		v, err := ops.Get(c, c.Params("id"))
		return v, http.StatusOK, err
	}))
	g.Post("/r/:resource", r.handle(func(c fiber.Ctx, ops resource.AdminOps) (any, int, error) {
		v, err := ops.Create(c, c.Body())
		return v, http.StatusCreated, err
	}))
	g.Patch("/r/:resource/:id", r.handle(func(c fiber.Ctx, ops resource.AdminOps) (any, int, error) {
		v, err := ops.Update(c, c.Params("id"), c.Body())
		return v, http.StatusOK, err
	}))
	g.Delete("/r/:resource/:id", r.handle(func(c fiber.Ctx, ops resource.AdminOps) (any, int, error) {
		return nil, http.StatusNoContent, ops.Delete(c, c.Params("id"))
	}))

	// ACL mutations grant privileges, so admin.access alone is not enough: an admin.access holder
	// could otherwise grant themselves any permission. Superuser (checked in the DB) is required.
	su := func(c fiber.Ctx) error { return acl.RequireSuperuserDB(cfg.ACL())(c) }
	g.Post("/permissions", su, h.CreatePermission)
	g.Delete("/permissions/:codename", su, h.DeletePermission)
	g.Post("/roles", su, h.CreateRole)
	g.Delete("/roles/:name", su, h.DeleteRole)
	g.Post("/assign/role", su, h.AssignRole)
	g.Post("/revoke/role", su, h.RevokeRole)
	g.Post("/assign/permission/role", su, h.AssignPermissionRole)
	g.Post("/revoke/permission/role", su, h.RevokePermissionRole)
	g.Post("/assign/permission/user", su, h.AssignPermissionUser)
	g.Post("/revoke/permission/user", su, h.RevokePermissionUser)
}

type resourceHandlers struct {
	list func() []resource.Resource
	deps func() resource.Deps
}

func (r *resourceHandlers) find(name string) resource.Resource {
	for _, res := range r.list() {
		if res.Meta().Name == name {
			return res
		}
	}
	return nil
}

func (r *resourceHandlers) meta(c fiber.Ctx) error {
	metas := []resource.Meta{}
	for _, res := range r.list() {
		metas = append(metas, res.Meta())
	}
	return c.JSON(fiber.Map{"resources": metas})
}

func (r *resourceHandlers) handle(fn func(c fiber.Ctx, ops resource.AdminOps) (any, int, error)) fiber.Handler {
	return func(c fiber.Ctx) error {
		res := r.find(c.Params("resource"))
		if res == nil {
			return api.WriteError(c, api.ErrNotFound)
		}
		out, status, err := fn(c, res.Admin(r.deps))
		if err != nil {
			return api.WriteError(c, err)
		}
		if out == nil {
			return c.SendStatus(status)
		}
		return c.Status(status).JSON(out)
	}
}
