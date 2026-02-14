package routes

import (
	"mithril-rev/database/repositories"
	crudhandlers "mithril-rev/internal/crud/user"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SetupCrudUserRoutes registers REST CRUD for users at /api/users.
func SetupCrudUserRoutes(app *fiber.App, pool *pgxpool.Pool) {
	if pool == nil {
		return
	}
	repo := repositories.NewUserRepository(pool)
	h := crudhandlers.NewHandlers(repo)
	api := app.Group("/api")
	api.Get("/users", h.List)
	api.Get("/users/:id", h.Get)
	api.Post("/users", h.Create)
	api.Put("/users/:id", h.Update)
	api.Delete("/users/:id", h.Delete)
}
