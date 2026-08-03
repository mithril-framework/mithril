package routes

import (
	"os"
	"time"

	"github.com/gofiber/fiber/v3/middleware/static"

	"github.com/gofiber/contrib/v3/monitor"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mithril-framework/mithril/pkg/version"
)

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// SetupCoreRoutes registers /, /health, /monitor, and /static.
func SetupCoreRoutes(app *fiber.App, pool *pgxpool.Pool) {
	app.Use("/static", static.New("./public/static"))

	app.Get("/", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Mithril API",
			"version": version.Version,
		})
	})

	app.Get("/health", func(c fiber.Ctx) error {
		status := fiber.Map{"status": "ok"}
		if pool != nil {
			if err := pool.Ping(c); err != nil {
				return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "unhealthy", "database": err.Error()})
			}
			status["database"] = "connected"
		}
		return c.JSON(status)
	})

	appName := getEnv("APP_NAME", "mithril")
	app.Get("/monitor", monitor.New(monitor.Config{
		Title:   appName + " Monitor",
		Refresh: 3 * time.Second,
	}))
}