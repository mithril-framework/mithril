package middleware

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// GuardConfig holds configuration for guard middleware
type GuardConfig struct {
	ErrorHandler func(*fiber.Ctx, error) error // Error handler for unauthorized access
	UserKey      string                        // Key to retrieve user from context
	RoleKey      string                        // Key to retrieve user role from context
}

// DefaultGuardConfig returns default guard configuration
func DefaultGuardConfig() GuardConfig {
	return GuardConfig{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(403).JSON(fiber.Map{
				"error":   "Access denied",
				"message": "You don't have permission to access this resource",
			})
		},
		UserKey: "user",
		RoleKey: "user_role",
	}
}

// Guard implements role-based access control middleware
type Guard struct {
	config GuardConfig
}

// NewGuardMiddleware creates a new guard middleware
func NewGuardMiddleware(config GuardConfig) *Guard {
	return &Guard{
		config: config,
	}
}

// getUserFromContext retrieves user from context
func (g *Guard) getUserFromContext(ctx *fiber.Ctx) interface{} {
	return ctx.Locals(g.config.UserKey)
}

// getUserRole retrieves user role from context
func (g *Guard) getUserRole(ctx *fiber.Ctx) string {
	if role := ctx.Locals(g.config.RoleKey); role != nil {
		return role.(string)
	}
	return ""
}

// hasRole checks if user has the specified role
func (g *Guard) hasRole(ctx *fiber.Ctx, requiredRole string) bool {
	user := g.getUserFromContext(ctx)
	if user == nil {
		return false
	}

	// Try to use User model's HasRole method if available
	type RoleChecker interface {
		HasRole(string) bool
	}

	if userModel, ok := user.(RoleChecker); ok {
		return userModel.HasRole(requiredRole)
	}

	// Fallback to simple role string comparison
	userRole := g.getUserRole(ctx)
	return userRole == requiredRole
}

// hasAnyRole checks if user has any of the specified roles
func (g *Guard) hasAnyRole(ctx *fiber.Ctx, requiredRoles []string) bool {
	user := g.getUserFromContext(ctx)
	if user == nil {
		return false
	}

	// Try to use User model's HasRole method if available
	type RoleChecker interface {
		HasRole(string) bool
	}

	if userModel, ok := user.(RoleChecker); ok {
		for _, role := range requiredRoles {
			if userModel.HasRole(role) {
				return true
			}
		}
		return false
	}

	// Fallback to simple role string comparison
	userRole := g.getUserRole(ctx)
	for _, role := range requiredRoles {
		if userRole == role {
			return true
		}
	}
	return false
}

// hasPermission checks if user has the specified permission
func (g *Guard) hasPermission(ctx *fiber.Ctx, permission string) bool {
	user := g.getUserFromContext(ctx)
	if user == nil {
		return false
	}

	// Try to use User model's HasPermission method if available
	type PermissionChecker interface {
		HasPermission(string) bool
	}

	if userModel, ok := user.(PermissionChecker); ok {
		return userModel.HasPermission(permission)
	}

	// Fallback to simple role-based approach
	userRole := g.getUserRole(ctx)

	// Define role-permission mapping
	rolePermissions := map[string][]string{
		"admin":     {"*"}, // Admin has all permissions
		"moderator": {"read", "write", "moderate"},
		"user":      {"read"},
		"guest":     {"read"},
	}

	permissions, exists := rolePermissions[userRole]
	if !exists {
		return false
	}

	// Check if user has the permission
	for _, perm := range permissions {
		if perm == "*" || perm == permission {
			return true
		}
	}

	return false
}

// hasAnyPermission checks if user has any of the specified permissions
func (g *Guard) hasAnyPermission(ctx *fiber.Ctx, permissions []string) bool {
	for _, permission := range permissions {
		if g.hasPermission(ctx, permission) {
			return true
		}
	}
	return false
}

// RequireAuth requires user to be authenticated
func (g *Guard) RequireAuth() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := g.getUserFromContext(ctx)
		if user == nil {
			return g.config.ErrorHandler(ctx, fmt.Errorf("authentication required"))
		}
		return ctx.Next()
	}
}

// RequireRole requires user to have specific role
func (g *Guard) RequireRole(role string) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		if !g.hasRole(ctx, role) {
			return g.config.ErrorHandler(ctx, fmt.Errorf("role '%s' required", role))
		}
		return ctx.Next()
	}
}

// RequireAnyRole requires user to have any of the specified roles
func (g *Guard) RequireAnyRole(roles ...string) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		if !g.hasAnyRole(ctx, roles) {
			return g.config.ErrorHandler(ctx, fmt.Errorf("one of roles %v required", roles))
		}
		return ctx.Next()
	}
}

// RequirePermission requires user to have specific permission
func (g *Guard) RequirePermission(permission string) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		if !g.hasPermission(ctx, permission) {
			return g.config.ErrorHandler(ctx, fmt.Errorf("permission '%s' required", permission))
		}
		return ctx.Next()
	}
}

// RequireAnyPermission requires user to have any of the specified permissions
func (g *Guard) RequireAnyPermission(permissions ...string) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		if !g.hasAnyPermission(ctx, permissions) {
			return g.config.ErrorHandler(ctx, fmt.Errorf("one of permissions %v required", permissions))
		}
		return ctx.Next()
	}
}

// RequireGuest requires user to be a guest (not authenticated)
func (g *Guard) RequireGuest() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := g.getUserFromContext(ctx)
		if user != nil {
			return g.config.ErrorHandler(ctx, fmt.Errorf("guest access required"))
		}
		return ctx.Next()
	}
}

// RequireVerified requires user to be verified
func (g *Guard) RequireVerified() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := g.getUserFromContext(ctx)
		if user == nil {
			return g.config.ErrorHandler(ctx, fmt.Errorf("authentication required"))
		}

		// Check if user is verified (this would depend on your user model)
		// For now, we'll assume there's a verified field
		if verified := ctx.Locals("user_verified"); verified == nil || !verified.(bool) {
			return g.config.ErrorHandler(ctx, fmt.Errorf("email verification required"))
		}

		return ctx.Next()
	}
}

// RequireAPIKey requires valid API key
func (g *Guard) RequireAPIKey() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		apiKey := ctx.Get("X-API-Key")
		if apiKey == "" {
			return g.config.ErrorHandler(ctx, fmt.Errorf("API key required"))
		}

		// Validate API key (this would typically check against database)
		// For now, we'll do a simple validation
		if len(apiKey) < 32 {
			return g.config.ErrorHandler(ctx, fmt.Errorf("invalid API key"))
		}

		return ctx.Next()
	}
}

// RequireIP allows access only from specific IP addresses
func (g *Guard) RequireIP(allowedIPs ...string) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		clientIP := ctx.IP()

		for _, allowedIP := range allowedIPs {
			if clientIP == allowedIP {
				return ctx.Next()
			}
		}

		return g.config.ErrorHandler(ctx, fmt.Errorf("access denied from IP %s", clientIP))
	}
}

// RequireIPRange allows access only from specific IP ranges
func (g *Guard) RequireIPRange(allowedRanges ...string) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		clientIP := ctx.IP()

		for _, allowedRange := range allowedRanges {
			if g.isIPInRange(clientIP, allowedRange) {
				return ctx.Next()
			}
		}

		return g.config.ErrorHandler(ctx, fmt.Errorf("access denied from IP %s", clientIP))
	}
}

// isIPInRange checks if IP is in the specified range
func (g *Guard) isIPInRange(ip, rangeStr string) bool {
	// Simple implementation - in production, use proper IP range checking
	return strings.HasPrefix(ip, rangeStr)
}

// NewGuard creates a guard middleware with default configuration
func NewGuard(config ...GuardConfig) *Guard {
	cfg := DefaultGuardConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return NewGuardMiddleware(cfg)
}

// Convenience functions for common guard patterns
func RequireAuth() fiber.Handler {
	guard := NewGuard()
	return guard.RequireAuth()
}

func RequireRole(role string) fiber.Handler {
	guard := NewGuard()
	return guard.RequireRole(role)
}

func RequireAnyRole(roles ...string) fiber.Handler {
	guard := NewGuard()
	return guard.RequireAnyRole(roles...)
}

func RequirePermission(permission string) fiber.Handler {
	guard := NewGuard()
	return guard.RequirePermission(permission)
}

func RequireAnyPermission(permissions ...string) fiber.Handler {
	guard := NewGuard()
	return guard.RequireAnyPermission(permissions...)
}

func RequireGuest() fiber.Handler {
	guard := NewGuard()
	return guard.RequireGuest()
}

func RequireVerified() fiber.Handler {
	guard := NewGuard()
	return guard.RequireVerified()
}

func RequireAPIKey() fiber.Handler {
	guard := NewGuard()
	return guard.RequireAPIKey()
}

func RequireIP(allowedIPs ...string) fiber.Handler {
	guard := NewGuard()
	return guard.RequireIP(allowedIPs...)
}

func RequireIPRange(allowedRanges ...string) fiber.Handler {
	guard := NewGuard()
	return guard.RequireIPRange(allowedRanges...)
}
