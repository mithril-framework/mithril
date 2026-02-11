package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/mithril-framework/mithril/pkg/middleware"
)

// RBAC wraps the framework's guard middleware for RBAC functionality
var guard = middleware.NewGuard()

// RequireRole requires user to have specific role
func RequireRole(role string) fiber.Handler {
	return guard.RequireRole(role)
}

// RequireAnyRole requires user to have any of the specified roles
func RequireAnyRole(roles ...string) fiber.Handler {
	return guard.RequireAnyRole(roles...)
}

// RequirePermission requires user to have specific permission
func RequirePermission(permission string) fiber.Handler {
	return guard.RequirePermission(permission)
}

// RequireAnyPermission requires user to have any of the specified permissions
func RequireAnyPermission(permissions ...string) fiber.Handler {
	return guard.RequireAnyPermission(permissions...)
}
