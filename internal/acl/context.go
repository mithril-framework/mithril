package acl

import (
	"errors"

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Context locals set by JWTClaimsMiddleware (after jwtware).
const (
	LocalUserID      = "acl_user_id"
	LocalEmail       = "acl_email"
	LocalRoles       = "acl_roles"
	LocalIsSuperuser = "acl_is_superuser"
	LocalSessionID   = "acl_session_id"
)

// JWTClaimsMiddleware copies MapClaims into Locals for ACL and handlers.
// Requires a token from jwtware.FromContext (Fiber v3 JWT middleware).
func JWTClaimsMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		tok := jwtware.FromContext(c)
		if tok == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized", "message": "missing token"})
		}
		claims, ok := tok.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized", "message": "invalid claims"})
		}
		uid, _ := claims["user_id"].(string)
		if uid == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized", "message": "missing user_id"})
		}
		c.Locals(LocalUserID, uid)
		if email, _ := claims["email"].(string); email != "" {
			c.Locals(LocalEmail, email)
		}
		c.Locals(LocalIsSuperuser, jwtBool(claims["is_superuser"]))
		if sid, _ := claims["session_id"].(string); sid != "" {
			c.Locals(LocalSessionID, sid)
		}
		c.Locals(LocalRoles, jwtStringSlice(claims["roles"]))
		return c.Next()
	}
}

func jwtBool(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case float64:
		return x != 0
	default:
		return false
	}
}

func jwtStringSlice(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// CurrentUserID parses Locals user id as UUID.
func CurrentUserID(c fiber.Ctx) (uuid.UUID, error) {
	s, ok := c.Locals(LocalUserID).(string)
	if !ok || s == "" {
		return uuid.Nil, errors.New("missing user id")
	}
	return uuid.Parse(s)
}

// IsSuperuserLocal reads JWT-derived superuser flag from Locals.
func IsSuperuserLocal(c fiber.Ctx) bool {
	v := c.Locals(LocalIsSuperuser)
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}
