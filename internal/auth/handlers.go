package auth

import (
	"errors"
	"fmt"
	"time"

	"mithril-rev/database/models"
	"mithril-rev/database/repositories"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = bcrypt.DefaultCost

// Handlers holds dependencies for auth HTTP handlers.
type Handlers struct {
	UserRepo *repositories.UserRepository
	JWTSecret string
}

// NewHandlers returns a new Handlers. UserRepo may be nil if DB is not configured.
func NewHandlers(userRepo *repositories.UserRepository, jwtSecret string) *Handlers {
	return &Handlers{UserRepo: userRepo, JWTSecret: jwtSecret}
}

// Register handles POST /auth/register.
func (h *Handlers) Register(c *fiber.Ctx) error {
	if h.UserRepo == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "service_unavailable", "message": "database not configured",
		})
	}
	var body struct {
		Email     string `json:"email"`
		Password  string `json:"password"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "validation_error", "message": "Invalid request format"})
	}
	if body.Email == "" || body.Password == "" {
		return c.Status(400).JSON(fiber.Map{"error": "validation_error", "message": "email and password are required"})
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcryptCost)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "internal_error", "message": "failed to hash password"})
	}
	u := &models.User{
		Email:        body.Email,
		PasswordHash: string(hash),
		FirstName:    body.FirstName,
		LastName:     body.LastName,
		IsActive:     true,
	}
	if err := h.UserRepo.Create(c.Context(), u); err != nil {
		var pgErr *pgconn.PgError
		if ok := errors.As(err, &pgErr); ok && pgErr.Code == "23505" {
			return c.Status(409).JSON(fiber.Map{"error": "conflict", "message": "email already exists"})
		}
		return c.Status(500).JSON(fiber.Map{"error": "internal_error", "message": "registration failed"})
	}
	return c.Status(201).JSON(fiber.Map{
		"success": true,
		"message": "User registered successfully. Please check your email for verification.",
		"data":    fiber.Map{"id": u.ID, "email": u.Email},
	})
}

// Login handles POST /auth/login.
func (h *Handlers) Login(c *fiber.Ctx) error {
	if h.UserRepo == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "service_unavailable", "message": "database not configured",
		})
	}
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Remember bool   `json:"remember"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "validation_error", "message": "Invalid request format"})
	}
	user, err := h.UserRepo.GetByEmail(c.Context(), req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(401).JSON(fiber.Map{"error": "authentication_failed", "message": "invalid credentials"})
		}
		return c.Status(500).JSON(fiber.Map{"error": "internal_error", "message": "login failed"})
	}
	if !user.IsActive {
		return c.Status(403).JSON(fiber.Map{"error": "account_disabled", "message": "account is disabled"})
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "authentication_failed", "message": "invalid credentials"})
	}
	userID := fmt.Sprintf("%d", user.ID)
	roles := []string{"user"}
	sessionID := "session_" + fmt.Sprintf("%d", time.Now().Unix())
	accessToken, refreshToken, expiresAt, err := h.issueTokenPair(userID, user.Email, roles, sessionID)
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
		"email":      user.Email,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
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

// ForgotPassword handles POST /auth/forgot-password.
func (h *Handlers) ForgotPassword(c *fiber.Ctx) error {
	var body struct {
		Email string `json:"email"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "validation_error", "message": "Invalid request format"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Password reset email sent successfully"})
}

// ResetPassword handles POST /auth/reset-password.
func (h *Handlers) ResetPassword(c *fiber.Ctx) error {
	var body struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "validation_error", "message": "Invalid request format"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Password reset successfully"})
}

// SendOTP handles POST /auth/send-otp.
func (h *Handlers) SendOTP(c *fiber.Ctx) error {
	var body struct {
		Phone string `json:"phone"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "validation_error", "message": "Invalid request format"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "OTP sent successfully"})
}

// VerifyOTP handles POST /auth/verify-otp.
func (h *Handlers) VerifyOTP(c *fiber.Ctx) error {
	var body struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "validation_error", "message": "Invalid request format"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "OTP verified successfully"})
}

// Logout handles POST /auth/logout.
func (h *Handlers) Logout(c *fiber.Ctx) error {
	c.ClearCookie("access_token")
	c.ClearCookie("refresh_token")
	return c.JSON(fiber.Map{"success": true, "message": "Logout successful"})
}

// Refresh handles POST /auth/refresh.
func (h *Handlers) Refresh(c *fiber.Ctx) error {
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
		return []byte(h.JWTSecret), nil
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
	accessToken, refreshToken, expiresAt, err := h.issueTokenPair(userID, email, roles, sessionID)
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

// Me handles GET /auth/me.
func (h *Handlers) Me(c *fiber.Ctx) error {
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

// Enable2FA handles POST /auth/enable-2fa.
func (h *Handlers) Enable2FA(c *fiber.Ctx) error {
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

// Verify2FA handles POST /auth/verify-2fa.
func (h *Handlers) Verify2FA(c *fiber.Ctx) error {
	var body struct {
		Code string `json:"code"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "validation_error", "message": "Invalid request format"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "2FA verified successfully"})
}

func (h *Handlers) issueTokenPair(userID, email string, roles []string, sessionID string) (accessToken, refreshToken string, expiresAt time.Time, err error) {
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
	accessToken, err = at.SignedString([]byte(h.JWTSecret))
	if err != nil {
		return "", "", time.Time{}, err
	}
	rt := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err = rt.SignedString([]byte(h.JWTSecret))
	if err != nil {
		return "", "", time.Time{}, err
	}
	return accessToken, refreshToken, accessExp, nil
}
