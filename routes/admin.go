package routes

import (
	"mithril-rev/database/repositories"
	"mithril-rev/internal/acl"
	"mithril-rev/internal/admin"

	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SetupAdminRoutes registers /admin static files and /admin/api (when panel enabled and pool set).
func SetupAdminRoutes(app *fiber.App, pool *pgxpool.Pool, userRepo *repositories.UserRepository, jwtCfg jwtware.Config, aclRepo *repositories.ACLRepository, aclSvc *acl.Service) {
	if pool == nil || !admin.PanelEnabled() || aclRepo == nil || aclSvc == nil {
		return
	}
	blogRepo := repositories.NewBlogRepository(pool)
	h := admin.NewHandlers(aclRepo, userRepo, blogRepo)

	app.Get("/admin", func(c *fiber.Ctx) error {
		return c.Redirect("/admin/")
	})

	app.Static("/admin", "./public/admin", fiber.Static{
		Index: "index.html",
	})

	api := app.Group("/admin/api")
	api.Use(jwtware.New(jwtCfg))
	api.Use(acl.JWTClaimsMiddleware())
	api.Use(acl.RequireAdminAccess(aclSvc))

	api.Get("/meta", h.Meta)

	api.Get("/permissions", h.ListPermissions)
	api.Post("/permissions", h.CreatePermission)
	api.Delete("/permissions/:codename", h.DeletePermission)

	api.Get("/roles", h.ListRoles)
	api.Post("/roles", h.CreateRole)
	api.Get("/roles/:name/permissions", h.RolePermissions)
	api.Delete("/roles/:name", h.DeleteRole)

	api.Post("/assign/role", h.AssignRole)
	api.Post("/revoke/role", h.RevokeRole)
	api.Post("/assign/permission/role", h.AssignPermissionRole)
	api.Post("/revoke/permission/role", h.RevokePermissionRole)
	api.Post("/assign/permission/user", h.AssignPermissionUser)
	api.Post("/revoke/permission/user", h.RevokePermissionUser)

	api.Get("/users/:id/roles", h.UserRoles)
	api.Get("/users/:id/permissions", h.UserDirectPermissions)

	api.Get("/resources/users", h.ListUsers)
	api.Get("/resources/users/:id", h.GetUser)
	api.Post("/resources/users", h.CreateUser)
	api.Put("/resources/users/:id", h.UpdateUser)
	api.Delete("/resources/users/:id", h.DeleteUser)

	api.Get("/resources/blogs", h.ListBlogs)
	api.Get("/resources/blogs/:id", h.GetBlog)
	api.Post("/resources/blogs", h.CreateBlog)
	api.Put("/resources/blogs/:id", h.UpdateBlog)
	api.Delete("/resources/blogs/:id", h.DeleteBlog)
}
