package acl

import (
	"github.com/mithril-framework/mithril/database/repositories"

	"github.com/gofiber/fiber/v2"
)

// RequirePermission denies with 403 when the user lacks the codename.
func RequirePermission(svc *Service, codename string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if svc == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden", "message": "acl not configured"})
		}
		uid, err := CurrentUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		ok, err := svc.HasPermission(c.Context(), c, uid, codename, IsSuperuserLocal(c))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "acl_error", "message": err.Error()})
		}
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden", "message": "missing permission: " + codename})
		}
		return c.Next()
	}
}

// RequireAnyPermission allows if any codename matches.
func RequireAnyPermission(svc *Service, codenames ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if svc == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden", "message": "acl not configured"})
		}
		uid, err := CurrentUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		su := IsSuperuserLocal(c)
		for _, codename := range codenames {
			ok, err := svc.HasPermission(c.Context(), c, uid, codename, su)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "acl_error", "message": err.Error()})
			}
			if ok {
				return c.Next()
			}
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden", "message": "missing required permission"})
	}
}

// RequireRole denies with 403 when the user does not have the role name.
func RequireRole(svc *Service, repo *repositories.ACLRepository, roleName string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if svc == nil || repo == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden", "message": "acl not configured"})
		}
		uid, err := CurrentUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		if IsSuperuserLocal(c) {
			return c.Next()
		}
		ok, err := repo.UserIsSuperuser(c.Context(), uid)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "acl_error", "message": err.Error()})
		}
		if ok {
			return c.Next()
		}
		has, err := svc.HasRole(c.Context(), uid, roleName)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "acl_error", "message": err.Error()})
		}
		if !has {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden", "message": "missing role: " + roleName})
		}
		return c.Next()
	}
}

// RequireSuperuser allows only JWT is_superuser or DB superuser.
func RequireSuperuser(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if IsSuperuserLocal(c) {
			return c.Next()
		}
		if svc == nil || svc.repo == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		}
		uid, err := CurrentUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		ok, err := svc.repo.UserIsSuperuser(c.Context(), uid)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "acl_error", "message": err.Error()})
		}
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden", "message": "superuser required"})
		}
		return c.Next()
	}
}

// RequireAdminAccess allows superuser or permission admin.access.
func RequireAdminAccess(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if svc == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden", "message": "acl not configured"})
		}
		uid, err := CurrentUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		su := IsSuperuserLocal(c)
		if su {
			return c.Next()
		}
		ok, err := svc.repo.UserIsSuperuser(c.Context(), uid)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "acl_error", "message": err.Error()})
		}
		if ok {
			return c.Next()
		}
		has, err := svc.HasPermission(c.Context(), c, uid, "admin.access", false)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "acl_error", "message": err.Error()})
		}
		if !has {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden", "message": "admin access denied"})
		}
		return c.Next()
	}
}
