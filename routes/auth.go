package routes

import (
	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/mithril-framework/mithril/internal/auth"
)

// SetupAuthRoutes registers the /auth group: public routes, then JWT middleware, then protected routes.
func SetupAuthRoutes(app *fiber.App, authHandlers *auth.Handlers, jwtConfig jwtware.Config) {
	authGroup := app.Group("/auth")

	// Public
	authGroup.Post("/register", authHandlers.Register)
	authGroup.Post("/login", authHandlers.Login)
	authGroup.Post("/forgot-password", authHandlers.ForgotPassword)
	authGroup.Post("/reset-password", authHandlers.ResetPassword)
	authGroup.Post("/send-otp", authHandlers.SendOTP)
	authGroup.Post("/verify-otp", authHandlers.VerifyOTP)

	authGroup.Use(jwtware.New(jwtConfig))

	// Protected
	authGroup.Post("/logout", authHandlers.Logout)
	authGroup.Post("/refresh", authHandlers.Refresh)
	authGroup.Get("/me", authHandlers.Me)
	authGroup.Post("/enable-2fa", authHandlers.Enable2FA)
	authGroup.Post("/verify-2fa", authHandlers.Verify2FA)
}
