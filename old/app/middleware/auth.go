package middleware

import (
	"github.com/mithril-framework/mithril/pkg/auth"
	"github.com/mithril-framework/mithril/pkg/middleware"
)

// AuthMiddleware wraps the framework's auth middleware
type AuthMiddleware struct {
	*middleware.AuthMiddleware
}

// NewAuthMiddleware creates a new auth middleware instance
func NewAuthMiddleware(jwtManager *auth.JWTManager) *AuthMiddleware {
	return &AuthMiddleware{
		AuthMiddleware: middleware.NewAuthMiddleware(jwtManager),
	}
}
