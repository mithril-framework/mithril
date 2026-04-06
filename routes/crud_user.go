package routes

import (
	"mithril-rev/database/repositories"
	"mithril-rev/internal/acl"
	crudhandlers "mithril-rev/internal/crud/user"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MountCrudUserRoutes registers user CRUD on an existing /api group (middleware already applied).
func MountCrudUserRoutes(api fiber.Router, pool *pgxpool.Pool, aclSvc *acl.Service) {
	if pool == nil {
		return
	}
	repo := repositories.NewUserRepository(pool)
	h := crudhandlers.NewHandlers(repo, aclSvc)
	api.Get("/users", acl.RequirePermission(aclSvc, "users.view"), h.List)
	api.Get("/users/:id", h.Get)
	api.Post("/users", acl.RequirePermission(aclSvc, "users.add"), h.Create)
	api.Put("/users/:id", h.Update)
	api.Delete("/users/:id", acl.RequirePermission(aclSvc, "users.delete"), h.Delete)
}
