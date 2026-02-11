package main

import (
	"fmt"
	"log"
	"os"
	"time"

	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/contrib/swagger"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/golang-jwt/jwt/v5"
)

func main() {
	app := fiber.New(fiber.Config{
		AppName: getEnv("APP_NAME", "mithril-rev"),
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error":   true,
				"message": err.Error(),
				"code":    code,
			})
		},
	})

	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} (${ip}) ${latency}\n",
	}))
	app.Use(recover.New())
	app.Use(healthcheck.New())
	app.Use(swagger.New(swagger.Config{
		BasePath: "/",
		FilePath: "./docs/swagger.json",
		Path:     "docs",
		Title:    "Mithril Rev API",
		CacheAge: 0, // no cache so doc updates show after restart
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Mithril Rev API",
			"version": "1.0.0",
		})

	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	app.Get("/monitor", monitor.New(monitor.Config{
		Title:   getEnv("APP_NAME", "mithril-rev") + " Monitor",
		Refresh: 3 * time.Second,
	}))

	jwtSecret := getEnv("JWT_SECRET", "secret")
	jwtConfig := jwtware.Config{
		SigningKey: jwtware.SigningKey{Key: []byte(jwtSecret)},
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized", "message": err.Error()})
		},
	}

	auth := app.Group("/auth")
	auth.Post("/register", authRegister)
	auth.Post("/login", authLogin(jwtSecret))
	auth.Post("/forgot-password", authForgotPassword)
	auth.Post("/reset-password", authResetPassword)
	auth.Post("/send-otp", authSendOTP)
	auth.Post("/verify-otp", authVerifyOTP)

	auth.Use(jwtware.New(jwtConfig))
	auth.Post("/logout", authLogout)
	auth.Post("/refresh", authRefresh(jwtSecret))
	auth.Get("/me", authMe)
	auth.Post("/enable-2fa", authEnable2FA)
	auth.Post("/verify-2fa", authVerify2FA)

	port := getEnv("PORT", "4000")
	log.Printf("Starting server on port %s", port)
	log.Fatal(app.Listen(":" + port))
}

// In-memory demo user for login (plan: minimal implementation)
const demoEmail = "user@example.com"
const demoPassword = "password"

func authRegister(c *fiber.Ctx) error {
	var body struct {
		Email     string `json:"email"`
		Password  string `json:"password"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "validation_error", "message": "Invalid request format"})
	}
	return c.Status(201).JSON(fiber.Map{
		"success": true,
		"message": "User registered successfully. Please check your email for verification.",
	})
}

func authLogin(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			Remember bool   `json:"remember"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "validation_error", "message": "Invalid request format"})
		}
		if req.Email != demoEmail || req.Password != demoPassword {
			return c.Status(401).JSON(fiber.Map{"error": "authentication_failed", "message": "invalid credentials"})
		}
		userID := "demo-user-id"
		roles := []string{"user"}
		sessionID := "session_" + fmt.Sprintf("%d", time.Now().Unix())
		accessToken, refreshToken, expiresAt, err := issueTokenPair(jwtSecret, userID, req.Email, roles, sessionID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "token_error", "message": err.Error()})
		}
		tokens := fiber.Map{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"expires_at":    expiresAt,
			"token_type":    "Bearer",
		}
		userData := fiber.Map{
			"id":         userID,
			"email":      req.Email,
			"first_name": "Demo",
			"last_name":  "User",
		}
		if req.Remember {
			c.Cookie(&fiber.Cookie{Name: "access_token", Value: accessToken, HTTPOnly: true, SameSite: "Lax", Expires: expiresAt})
			c.Cookie(&fiber.Cookie{Name: "refresh_token", Value: refreshToken, HTTPOnly: true, SameSite: "Lax", Expires: time.Now().Add(7 * 24 * time.Hour)})
		}
		return c.JSON(fiber.Map{
			"success": true,
			"message": "Login successful",
			"data":    fiber.Map{"user": userData, "tokens": tokens},
		})
	}
}

func authForgotPassword(c *fiber.Ctx) error {
	var body struct {
		Email string `json:"email"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "validation_error", "message": "Invalid request format"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Password reset email sent successfully"})
}

func authResetPassword(c *fiber.Ctx) error {
	var body struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "validation_error", "message": "Invalid request format"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Password reset successfully"})
}

func authSendOTP(c *fiber.Ctx) error {
	var body struct {
		Phone string `json:"phone"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "validation_error", "message": "Invalid request format"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "OTP sent successfully"})
}

func authVerifyOTP(c *fiber.Ctx) error {
	var body struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "validation_error", "message": "Invalid request format"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "OTP verified successfully"})
}

func authLogout(c *fiber.Ctx) error {
	c.ClearCookie("access_token")
	c.ClearCookie("refresh_token")
	return c.JSON(fiber.Map{"success": true, "message": "Logout successful"})
}

func authRefresh(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			RefreshToken string `json:"refresh_token"`
		}
		_ = c.BodyParser(&body)
		if body.RefreshToken == "" {
			body.RefreshToken = c.Cookies("refresh_token")
		}
		if body.RefreshToken == "" {
			return c.Status(400).JSON(fiber.Map{"error": "validation_error", "message": "Refresh token is required"})
		}
		claims := jwt.MapClaims{}
		tok, err := jwt.ParseWithClaims(body.RefreshToken, &claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})
		if err != nil || !tok.Valid {
			return c.Status(401).JSON(fiber.Map{"error": "token_refresh_failed", "message": "invalid refresh token"})
		}
		userID, _ := claims["user_id"].(string)
		email, _ := claims["email"].(string)
		sessionID, _ := claims["session_id"].(string)
		var roles []string
		if r, ok := claims["roles"].([]interface{}); ok {
			for _, v := range r {
				if s, ok := v.(string); ok {
					roles = append(roles, s)
				}
			}
		}
		if len(roles) == 0 {
			roles = []string{"user"}
		}
		accessToken, refreshToken, expiresAt, err := issueTokenPair(jwtSecret, userID, email, roles, sessionID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "token_error", "message": err.Error()})
		}
		return c.JSON(fiber.Map{
			"success": true,
			"data": fiber.Map{
				"access_token":  accessToken,
				"refresh_token": refreshToken,
				"expires_at":    expiresAt,
				"token_type":    "Bearer",
			},
		})
	}
}

func authMe(c *fiber.Ctx) error {
	token := c.Locals("user").(*jwt.Token)
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized", "message": "invalid claims"})
	}
	userID, _ := claims["user_id"].(string)
	email, _ := claims["email"].(string)
	var roles []string
	if r, ok := claims["roles"].([]interface{}); ok {
		for _, v := range r {
			if s, ok := v.(string); ok {
				roles = append(roles, s)
			}
		}
	}
	return c.JSON(fiber.Map{
		"success": true,
		"data":    fiber.Map{"id": userID, "email": email, "roles": roles},
	})
}

func authEnable2FA(c *fiber.Ctx) error {
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

func authVerify2FA(c *fiber.Ctx) error {
	var body struct {
		Code string `json:"code"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "validation_error", "message": "Invalid request format"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "2FA verified successfully"})
}

func issueTokenPair(jwtSecret, userID, email string, roles []string, sessionID string) (accessToken, refreshToken string, expiresAt time.Time, err error) {
	now := time.Now()
	accessExp := now.Add(24 * time.Hour)
	refreshExp := now.Add(7 * 24 * time.Hour)
	accessClaims := jwt.MapClaims{
		"user_id": userID, "email": email, "roles": roles, "session_id": sessionID,
		"exp": accessExp.Unix(), "type": "access",
	}
	refreshClaims := jwt.MapClaims{
		"user_id": userID, "email": email, "roles": roles, "session_id": sessionID,
		"exp": refreshExp.Unix(), "type": "refresh",
	}
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err = at.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", "", time.Time{}, err
	}
	rt := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err = rt.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", "", time.Time{}, err
	}
	return accessToken, refreshToken, accessExp, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
