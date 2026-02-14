package routes

import (
	"mithril-rev/database/repositories"
	"mithril-rev/internal/auth"

	"github.com/gofiber/fiber/v2"
	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RegisterAll registers all route setup functions (core, web, auth, vendor, and any generated CRUD).
func RegisterAll(app *fiber.App, pool *pgxpool.Pool, userRepo *repositories.UserRepository, jwtSecret string) {
	jwtConfig := jwtware.Config{
		SigningKey: jwtware.SigningKey{Key: []byte(jwtSecret)},
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized", "message": err.Error()})
		},
	}
	authHandlers := auth.NewHandlers(userRepo, jwtSecret)

	SetupCoreRoutes(app, pool)
	SetupWebRoutes(app, pool)
	SetupAuthRoutes(app, authHandlers, jwtConfig)
	SetupVendorRoutes(app, pool)
	SetupCrudBlogRoutes(app, pool)
	SetupCrudUserRoutes(app, pool)
}
