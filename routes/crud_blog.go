package routes

import (
	"mithril-rev/database/repositories"
	"mithril-rev/internal/acl"
	crudhandlers "mithril-rev/internal/crud/blog"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MountCrudBlogRoutes registers blog CRUD on an existing /api group (middleware already applied).
func MountCrudBlogRoutes(api fiber.Router, pool *pgxpool.Pool, aclSvc *acl.Service) {
	if pool == nil {
		return
	}
	repo := repositories.NewBlogRepository(pool)
	h := crudhandlers.NewHandlers(repo, aclSvc)
	api.Get("/blogs", h.List)
	api.Get("/blogs/:id", h.Get)
	api.Post("/blogs", h.Create)
	api.Put("/blogs/:id", h.Update)
	api.Delete("/blogs/:id", h.Delete)
}
