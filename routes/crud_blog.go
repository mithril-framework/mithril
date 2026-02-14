package routes

import (
	"mithril-rev/database/repositories"
	crudhandlers "mithril-rev/internal/crud/blog"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SetupCrudBlogRoutes registers REST CRUD for blogs at /api/blogs.
func SetupCrudBlogRoutes(app *fiber.App, pool *pgxpool.Pool) {
	if pool == nil {
		return
	}
	repo := repositories.NewBlogRepository(pool)
	h := crudhandlers.NewHandlers(repo)
	api := app.Group("/api")
	api.Get("/blogs", h.List)
	api.Get("/blogs/:id", h.Get)
	api.Post("/blogs", h.Create)
	api.Put("/blogs/:id", h.Update)
	api.Delete("/blogs/:id", h.Delete)
}
