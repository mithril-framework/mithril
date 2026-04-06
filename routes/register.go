package routes

import (
	"mithril-rev/database/repositories"
	"mithril-rev/internal/acl"
	"mithril-rev/internal/auth"

	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RegisterAll registers all route setup functions (core, web, auth, vendor, CRUD, optional admin).
func RegisterAll(app *fiber.App, pool *pgxpool.Pool, userRepo *repositories.UserRepository, jwtSecret string) {
	jwtConfig := jwtware.Config{
		SigningKey: jwtware.SigningKey{Key: []byte(jwtSecret)},
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized", "message": err.Error()})
		},
	}

	var aclRepo *repositories.ACLRepository
	var aclSvc *acl.Service
	if pool != nil {
		aclRepo = repositories.NewACLRepository(pool)
		aclSvc = acl.NewService(aclRepo)
	}

	authHandlers := auth.NewHandlers(userRepo, aclRepo, jwtSecret)

	SetupCoreRoutes(app, pool)
	SetupWebRoutes(app, pool)
	SetupAuthRoutes(app, authHandlers, jwtConfig)
	SetupVendorRoutes(app, pool)
	if pool != nil {
		api := app.Group("/api")
		api.Use(jwtware.New(jwtConfig))
		api.Use(acl.JWTClaimsMiddleware())
		MountCrudBlogRoutes(api, pool, aclSvc)
		MountCrudUserRoutes(api, pool, aclSvc)
	}
	SetupAdminRoutes(app, pool, userRepo, jwtConfig, aclRepo, aclSvc)
}
