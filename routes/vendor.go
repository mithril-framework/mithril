package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"mithril-rev/internal/vendor"
)

// SetupVendorRoutes registers the /vendor group and GET /vendor/dashboard.
func SetupVendorRoutes(app *fiber.App, pool *pgxpool.Pool) {
	vendorGroup := app.Group("/vendor")
	vendorGroup.Get("/dashboard", vendor.Dashboard(pool))
}
