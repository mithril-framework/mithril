package routes

import (
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/jackc/pgx/v5/pgxpool"
)

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// SetupWebRoutes registers /, /health, /monitor, and /static.
func SetupWebRoutes(app *fiber.App, pool *pgxpool.Pool) {
	app.Static("/static", "./public/static")

	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Mithril Rev API",
			"version": "1.0.0",
		})
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		status := fiber.Map{"status": "ok"}
		if pool != nil {
			if err := pool.Ping(c.Context()); err != nil {
				return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "unhealthy", "database": err.Error()})
			}
			status["database"] = "connected"
		}
		return c.JSON(status)
	})

	appName := getEnv("APP_NAME", "mithril-rev")
	app.Get("/monitor", monitor.New(monitor.Config{
		Title:   appName + " Monitor",
		Refresh: 3 * time.Second,
	}))
}
