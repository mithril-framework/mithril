package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/mithril-framework/mithril/pkg/auth"
)

// AuthMiddleware handles authentication
type AuthMiddleware struct {
	jwtManager *auth.JWTManager
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(jwtManager *auth.JWTManager) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager: jwtManager,
	}
}

// RequireAuth middleware that requires authentication
func (a *AuthMiddleware) RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, err := a.extractToken(c)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": err.Error(),
			})
		}

		claims, err := a.jwtManager.ValidateToken(token)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "Invalid token",
			})
		}

		// Check if token is blacklisted
		if a.jwtManager.IsTokenBlacklisted(token) {
			return c.Status(401).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "Token has been revoked",
			})
		}

		// Store user info in context
		c.Locals("user_id", claims.UserID)
		c.Locals("user_email", claims.Email)
		c.Locals("user_roles", claims.Roles)
		c.Locals("session_id", claims.SessionID)

		return c.Next()
	}
}

// RequireRole middleware that requires a specific role
func (a *AuthMiddleware) RequireRole(role string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// First check if user is authenticated
		if err := a.RequireAuth()(c); err != nil {
			return err
		}

		// Get user roles from context
		roles, ok := c.Locals("user_roles").([]string)
		if !ok {
			return c.Status(403).JSON(fiber.Map{
				"error":   "forbidden",
				"message": "Unable to determine user roles",
			})
		}

		// Check if user has required role
		hasRole := false
		for _, r := range roles {
			if r == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			return c.Status(403).JSON(fiber.Map{
				"error":   "forbidden",
				"message": "Insufficient permissions",
			})
		}

		return c.Next()
	}
}

// RequireAnyRole middleware that requires any of the specified roles
func (a *AuthMiddleware) RequireAnyRole(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// First check if user is authenticated
		if err := a.RequireAuth()(c); err != nil {
			return err
		}

		// Get user roles from context
		userRoles, ok := c.Locals("user_roles").([]string)
		if !ok {
			return c.Status(403).JSON(fiber.Map{
				"error":   "forbidden",
				"message": "Unable to determine user roles",
			})
		}

		// Check if user has any of the required roles
		hasRole := false
		for _, requiredRole := range roles {
			for _, userRole := range userRoles {
				if userRole == requiredRole {
					hasRole = true
					break
				}
			}
			if hasRole {
				break
			}
		}

		if !hasRole {
			return c.Status(403).JSON(fiber.Map{
				"error":   "forbidden",
				"message": "Insufficient permissions",
			})
		}

		return c.Next()
	}
}

// RequirePermission middleware that requires a specific permission
func (a *AuthMiddleware) RequirePermission(permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// First check if user is authenticated
		if err := a.RequireAuth()(c); err != nil {
			return err
		}

		// TODO: Implement permission checking
		// This would require database access to check user permissions
		// For now, we'll just check if user is authenticated

		return c.Next()
	}
}

// OptionalAuth middleware that adds user info to context if authenticated
func (a *AuthMiddleware) OptionalAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, err := a.extractToken(c)
		if err != nil {
			// No token provided, continue without authentication
			return c.Next()
		}

		claims, err := a.jwtManager.ValidateToken(token)
		if err != nil {
			// Invalid token, continue without authentication
			return c.Next()
		}

		// Check if token is blacklisted
		if a.jwtManager.IsTokenBlacklisted(token) {
			// Token is blacklisted, continue without authentication
			return c.Next()
		}

		// Store user info in context
		c.Locals("user_id", claims.UserID)
		c.Locals("user_email", claims.Email)
		c.Locals("user_roles", claims.Roles)
		c.Locals("session_id", claims.SessionID)

		return c.Next()
	}
}

// extractToken extracts the JWT token from the request
func (a *AuthMiddleware) extractToken(c *fiber.Ctx) (string, error) {
	// Try to get token from Authorization header
	authHeader := c.Get("Authorization")
	if authHeader != "" {
		return a.jwtManager.ExtractTokenFromHeader(authHeader)
	}

	// Try to get token from query parameter
	token := c.Query("token")
	if token != "" {
		return token, nil
	}

	// Try to get token from cookie
	token = c.Cookies("access_token")
	if token != "" {
		return token, nil
	}

	return "", fiber.NewError(401, "No authentication token provided")
}

// GetUserID gets the user ID from context
func GetUserID(c *fiber.Ctx) (string, bool) {
	userID, ok := c.Locals("user_id").(string)
	return userID, ok
}

// GetUserEmail gets the user email from context
func GetUserEmail(c *fiber.Ctx) (string, bool) {
	email, ok := c.Locals("user_email").(string)
	return email, ok
}

// GetUserRoles gets the user roles from context
func GetUserRoles(c *fiber.Ctx) ([]string, bool) {
	roles, ok := c.Locals("user_roles").([]string)
	return roles, ok
}

// GetSessionID gets the session ID from context
func GetSessionID(c *fiber.Ctx) (string, bool) {
	sessionID, ok := c.Locals("session_id").(string)
	return sessionID, ok
}

// IsAuthenticated checks if the user is authenticated
func IsAuthenticated(c *fiber.Ctx) bool {
	_, ok := c.Locals("user_id").(string)
	return ok
}

// HasRole checks if the user has a specific role
func HasRole(c *fiber.Ctx, role string) bool {
	roles, ok := c.Locals("user_roles").([]string)
	if !ok {
		return false
	}

	for _, r := range roles {
		if r == role {
			return true
		}
	}

	return false
}

// HasAnyRole checks if the user has any of the specified roles
func HasAnyRole(c *fiber.Ctx, roles ...string) bool {
	userRoles, ok := c.Locals("user_roles").([]string)
	if !ok {
		return false
	}

	for _, requiredRole := range roles {
		for _, userRole := range userRoles {
			if userRole == requiredRole {
				return true
			}
		}
	}

	return false
}
