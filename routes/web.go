package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SetupWebRoutes registers routes for web pages (HTML/templates).
// Add page routes here as they are implemented.
func SetupWebRoutes(app *fiber.App, pool *pgxpool.Pool) {
	// Web page routes go here
	_ = pool
}
