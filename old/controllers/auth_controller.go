package controllers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mithril-framework/mithril/pkg/auth"
	"github.com/mithril-framework/mithril/pkg/middleware"
)

// AuthController handles authentication endpoints
type AuthController struct {
	authService *auth.AuthService
}

// NewAuthController creates a new auth controller
func NewAuthController(authService *auth.AuthService) *AuthController {
	return &AuthController{
		authService: authService,
	}
}

// Register handles user registration
func (ac *AuthController) Register(c *fiber.Ctx) error {
	var req auth.RegisterRequest

	// Parse and validate request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error":   "validation_error",
			"message": "Invalid request format",
		})
	}

	// Register user
	if err := ac.authService.Register(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error":   "registration_failed",
			"message": err.Error(),
		})
	}

	return c.Status(201).JSON(fiber.Map{
		"success": true,
		"message": "User registered successfully. Please check your email for verification.",
	})
}

// Login handles user login
func (ac *AuthController) Login(c *fiber.Ctx) error {
	var req auth.LoginRequest

	// Parse and validate request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error":   "validation_error",
			"message": "Invalid request format",
		})
	}

	// Login user
	response, err := ac.authService.Login(&req)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error":   "authentication_failed",
			"message": err.Error(),
		})
	}

	// If 2FA is required
	if response.Requires2FA {
		return c.Status(200).JSON(fiber.Map{
			"success":      false,
			"requires_2fa": true,
			"message":      "Please provide your 2FA code",
		})
	}

	// If OTP is required
	if response.RequiresOTP {
		return c.Status(200).JSON(fiber.Map{
			"success":      false,
			"requires_otp": true,
			"message":      "Please provide the OTP sent to your phone",
		})
	}

	// Set cookies if remember is true
	if req.Remember {
		c.Cookie(&fiber.Cookie{
			Name:     "access_token",
			Value:    response.Tokens.AccessToken,
			Expires:  response.Tokens.ExpiresAt,
			HTTPOnly: true,
			Secure:   true,
			SameSite: "Lax",
		})

		c.Cookie(&fiber.Cookie{
			Name:     "refresh_token",
			Value:    response.Tokens.RefreshToken,
			Expires:  time.Now().Add(7 * 24 * time.Hour), // 7 days
			HTTPOnly: true,
			Secure:   true,
			SameSite: "Lax",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Login successful",
		"data":    response,
	})
}

// Logout handles user logout
func (ac *AuthController) Logout(c *fiber.Ctx) error {
	// Get token from context or header
	token := c.Get("Authorization")
	if token == "" {
		token = c.Cookies("access_token")
	}

	if token != "" {
		// Remove "Bearer " prefix if present
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		// Logout user
		_ = ac.authService.Logout(token) // Ignore error - user might have already logged out
	}

	// Clear cookies
	c.ClearCookie("access_token")
	c.ClearCookie("refresh_token")

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Logout successful",
	})
}

// RefreshToken handles token refresh
func (ac *AuthController) RefreshToken(c *fiber.Ctx) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	// Parse request
	if err := c.BodyParser(&req); err != nil {
		// Try to get from cookie
		req.RefreshToken = c.Cookies("refresh_token")
	}

	if req.RefreshToken == "" {
		return c.Status(400).JSON(fiber.Map{
			"error":   "validation_error",
			"message": "Refresh token is required",
		})
	}

	// Refresh token
	tokens, err := ac.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error":   "token_refresh_failed",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    tokens,
	})
}

// ForgotPassword handles password reset request
func (ac *AuthController) ForgotPassword(c *fiber.Ctx) error {
	var req struct {
		Email string `json:"email" validate:"required,email"`
	}

	// Parse and validate request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error":   "validation_error",
			"message": "Invalid request format",
		})
	}

	// Send password reset email
	if err := ac.authService.ForgotPassword(req.Email); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":   "password_reset_failed",
			"message": "Failed to send password reset email",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Password reset email sent successfully",
	})
}

// ResetPassword handles password reset
func (ac *AuthController) ResetPassword(c *fiber.Ctx) error {
	var req struct {
		Token       string `json:"token" validate:"required"`
		NewPassword string `json:"new_password" validate:"required,password"`
	}

	// Parse and validate request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error":   "validation_error",
			"message": "Invalid request format",
		})
	}

	// Reset password
	if err := ac.authService.ResetPassword(req.Token, req.NewPassword); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error":   "password_reset_failed",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Password reset successfully",
	})
}

// Me returns current user information
func (ac *AuthController) Me(c *fiber.Ctx) error {
	// Get user ID from context
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.Status(401).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "User not authenticated",
		})
	}

	// TODO: Get user from database
	email, _ := middleware.GetUserEmail(c)
	roles, _ := middleware.GetUserRoles(c)
	user := fiber.Map{
		"id":    userID,
		"email": email,
		"roles": roles,
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    user,
	})
}

// SendOTP sends OTP to user's phone
func (ac *AuthController) SendOTP(c *fiber.Ctx) error {
	var req struct {
		Phone string `json:"phone" validate:"required,phone"`
	}

	// Parse and validate request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error":   "validation_error",
			"message": "Invalid request format",
		})
	}

	// TODO: Implement OTP sending
	// This would generate and send OTP to the phone number

	return c.JSON(fiber.Map{
		"success": true,
		"message": "OTP sent successfully",
	})
}

// VerifyOTP verifies OTP code
func (ac *AuthController) VerifyOTP(c *fiber.Ctx) error {
	var req struct {
		Phone string `json:"phone" validate:"required,phone"`
		Code  string `json:"code" validate:"required,len=6"`
	}

	// Parse and validate request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error":   "validation_error",
			"message": "Invalid request format",
		})
	}

	// TODO: Implement OTP verification
	// This would verify the OTP code against the stored code

	return c.JSON(fiber.Map{
		"success": true,
		"message": "OTP verified successfully",
	})
}

// Enable2FA enables 2FA for the user
func (ac *AuthController) Enable2FA(c *fiber.Ctx) error {
	// Get user ID from context
	_, ok := middleware.GetUserID(c)
	if !ok {
		return c.Status(401).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "User not authenticated",
		})
	}

	// TODO: Implement 2FA setup
	// This would generate a secret and QR code for the user

	return c.JSON(fiber.Map{
		"success": true,
		"message": "2FA setup initiated",
		"data": fiber.Map{
			"secret":       "placeholder_secret",
			"qr_code":      "placeholder_qr_code_url",
			"backup_codes": []string{"code1", "code2", "code3"},
		},
	})
}

// Verify2FA verifies 2FA code
func (ac *AuthController) Verify2FA(c *fiber.Ctx) error {
	var req struct {
		Code string `json:"code" validate:"required,len=6"`
	}

	// Parse and validate request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error":   "validation_error",
			"message": "Invalid request format",
		})
	}

	// TODO: Implement 2FA verification
	// This would verify the 2FA code against the user's secret

	return c.JSON(fiber.Map{
		"success": true,
		"message": "2FA verified successfully",
	})
}
