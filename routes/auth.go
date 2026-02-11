package routes

import (
	// "time"

	"github.com/gofiber/fiber/v2"
	"github.com/mithril-framework/mithril/pkg/auth"
	"github.com/mithril-framework/mithril/pkg/middleware"
)

// SetupAuthRoutes sets up authentication routes
func SetupAuthRoutes(app *fiber.App, jwtManager *auth.JWTManager, authMiddleware *middleware.AuthMiddleware) {
	auth := app.Group("/auth")

	// Public auth routes
	auth.Post("/register", func(c *fiber.Ctx) error {
		var req struct {
			Email    string `json:"email" validate:"required,email"`
			Password string `json:"password" validate:"required,min=8"`
			Name     string `json:"name" validate:"required"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error":   "validation_error",
				"message": "Invalid request format",
			})
		}

		// TODO: Implement actual user registration
		// For now, just return success
		return c.Status(201).JSON(fiber.Map{
			"success": true,
			"message": "User registered successfully. Please check your email for verification.",
		})
	})

	auth.Post("/login", func(c *fiber.Ctx) error {
		var req struct {
			Email    string `json:"email" validate:"required,email"`
			Password string `json:"password" validate:"required"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error":   "validation_error",
				"message": "Invalid request format",
			})
		}

		// TODO: Implement actual user login
		// For now, just return success with mock tokens
		userID := "550e8400-e29b-41d4-a716-446655440000"
		sessionID := "session-123"
		roles := []string{"user"}

		tokenPair, err := jwtManager.GenerateTokenPair(userID, req.Email, roles, sessionID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error":   "token_generation_failed",
				"message": "Failed to generate authentication tokens",
			})
		}

		return c.JSON(fiber.Map{
			"success": true,
			"message": "Login successful",
			"data": fiber.Map{
				"user": fiber.Map{
					"id":    userID,
					"email": req.Email,
					"roles": roles,
				},
				"tokens": tokenPair,
			},
		})
	})

	// Protected auth routes
	auth.Use(authMiddleware.RequireAuth())

	auth.Post("/logout", func(c *fiber.Ctx) error {
		// TODO: Implement actual logout (blacklist token)
		return c.JSON(fiber.Map{
			"success": true,
			"message": "Logout successful",
		})
	})

	auth.Get("/me", func(c *fiber.Ctx) error {
		// Get user info from context
		userID, _ := middleware.GetUserID(c)
		email, _ := middleware.GetUserEmail(c)
		roles, _ := middleware.GetUserRoles(c)

		return c.JSON(fiber.Map{
			"success": true,
			"data": fiber.Map{
				"id":    userID,
				"email": email,
				"roles": roles,
			},
		})
	})

	auth.Post("/refresh", func(c *fiber.Ctx) error {
		var req struct {
			RefreshToken string `json:"refresh_token" validate:"required"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error":   "validation_error",
				"message": "Invalid request format",
			})
		}

		// Use JWT manager's RefreshToken method
		tokenPair, err := jwtManager.RefreshToken(req.RefreshToken)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{
				"error":   "invalid_refresh_token",
				"message": "Invalid or expired refresh token",
			})
		}

		return c.JSON(fiber.Map{
			"success": true,
			"data":    tokenPair,
		})
	})
}
