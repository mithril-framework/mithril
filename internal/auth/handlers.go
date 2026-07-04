package auth

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/mithril-framework/mithril/database/models"
	"github.com/mithril-framework/mithril/database/repositories"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = bcrypt.DefaultCost

// Handlers holds dependencies for auth HTTP handlers.
type Handlers struct {
	UserRepo  *repositories.UserRepository
	ACLRepo   *repositories.ACLRepository
	JWTSecret string
}

// NewHandlers returns a new Handlers. UserRepo may be nil if DB is not configured. ACLRepo may be nil.
func NewHandlers(userRepo *repositories.UserRepository, aclRepo *repositories.ACLRepository, jwtSecret string) *Handlers {
	return &Handlers{UserRepo: userRepo, ACLRepo: aclRepo, JWTSecret: jwtSecret}
}

// isRegisterEnabled returns true when ENABLE_REGISTER is a truthy value (true, 1, yes); false otherwise.
func isRegisterEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("ENABLE_REGISTER")))
	return v == "true" || v == "1" || v == "yes"
}

// Register handles POST /auth/register.
func (h *Handlers) Register(c fiber.Ctx) error {
	if !isRegisterEnabled() {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error":   "registration_disabled",
			"message": "Registration is disabled. Set ENABLE_REGISTER=true to enable.",
		})
	}
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
	if err := c.Bind().Body(&body); err != nil {
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
	if err := h.UserRepo.Create(c, u); err != nil {
		var pgErr *pgconn.PgError
		if ok := errors.As(err, &pgErr); ok && pgErr.Code == "23505" {
			return c.Status(409).JSON(fiber.Map{"error": "conflict", "message": "email already exists"})
		}
		return c.Status(500).JSON(fiber.Map{"error": "internal_error", "message": "registration failed"})
	}
	log.Printf("user registered: email=%s id=%s", u.Email, u.ID)
	return c.Status(201).JSON(fiber.Map{
		"success": true,
		"message": "User registered successfully. Please check your email for verification.",
		"data":    fiber.Map{"id": u.ID, "email": u.Email},
	})
}

// Login handles POST /auth/login.
func (h *Handlers) Login(c fiber.Ctx) error {
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
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "validation_error", "message": "Invalid request format"})
	}
	user, err := h.UserRepo.GetByEmail(c, req.Email)
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
	userID := user.ID.String()
	roles := []string{"user"}
	if h.ACLRepo != nil {
		if names, err := h.ACLRepo.UserRoleNames(c, user.ID); err == nil && len(names) > 0 {
			roles = names
		}
	}
	sessionID := "session_" + fmt.Sprintf("%d", time.Now().Unix())
	accessToken, refreshToken, expiresAt, err := h.issueTokenPair(userID, user.Email, roles, sessionID, user.IsSuperuser)
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
		"id":           userID,
		"email":        user.Email,
		"first_name":   user.FirstName,
		"last_name":    user.LastName,
		"is_superuser": user.IsSuperuser,
		"roles":        roles,
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

// notImplemented responds with 501 for unimplemented auth endpoints.
func notImplemented(c fiber.Ctx, feature string) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
		"error":   "not_implemented",
		"message": feature + " is not implemented yet; see ROADMAP.md",
	})
}

// ForgotPassword handles POST /auth/forgot-password.
func (h *Handlers) ForgotPassword(c fiber.Ctx) error {
	return notImplemented(c, "password reset")
}

// ResetPassword handles POST /auth/reset-password.
func (h *Handlers) ResetPassword(c fiber.Ctx) error {
	return notImplemented(c, "password reset")
}

// SendOTP handles POST /auth/send-otp.
func (h *Handlers) SendOTP(c fiber.Ctx) error {
	return notImplemented(c, "OTP")
}

// VerifyOTP handles POST /auth/verify-otp.
func (h *Handlers) VerifyOTP(c fiber.Ctx) error {
	return notImplemented(c, "OTP verification")
}

// Logout handles POST /auth/logout.
func (h *Handlers) Logout(c fiber.Ctx) error {
	c.ClearCookie("access_token")
	c.ClearCookie("refresh_token")
	return c.JSON(fiber.Map{"success": true, "message": "Logout successful"})
}

// Refresh handles POST /auth/refresh.
func (h *Handlers) Refresh(c fiber.Ctx) error {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = c.Bind().Body(&body)
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
	if userID == "" {
		return c.Status(401).JSON(fiber.Map{"error": "token_refresh_failed", "message": "invalid refresh token"})
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "token_refresh_failed", "message": "invalid refresh token"})
	}
	var roles []string
	if r, ok := claims["roles"].([]interface{}); ok {
		for _, v := range r {
			if s, ok := v.(string); ok {
				roles = append(roles, s)
			}
		}
	}
	isSuper := jwtClaimBool(claims, "is_superuser")
	if h.UserRepo != nil {
		u, err := h.UserRepo.GetByID(c, userUUID)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "token_refresh_failed", "message": "user not found"})
		}
		isSuper = u.IsSuperuser
		if h.ACLRepo != nil {
			if names, err := h.ACLRepo.UserRoleNames(c, userUUID); err == nil && len(names) > 0 {
				roles = names
			}
		}
	}
	if len(roles) == 0 {
		roles = []string{"user"}
	}
	accessToken, refreshToken, expiresAt, err := h.issueTokenPair(userID, email, roles, sessionID, isSuper)
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
func (h *Handlers) Me(c fiber.Ctx) error {
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
		"data": fiber.Map{
			"id": userID, "email": email, "roles": roles,
			"is_superuser": jwtClaimBool(claims, "is_superuser"),
		},
	})
}

func jwtClaimBool(claims jwt.MapClaims, key string) bool {
	v, ok := claims[key]
	if !ok {
		return false
	}
	switch x := v.(type) {
	case bool:
		return x
	case float64:
		return x != 0
	default:
		return false
	}
}

// Enable2FA handles POST /auth/enable-2fa.
func (h *Handlers) Enable2FA(c fiber.Ctx) error {
	return notImplemented(c, "2FA")
}

// Verify2FA handles POST /auth/verify-2fa.
func (h *Handlers) Verify2FA(c fiber.Ctx) error {
	return notImplemented(c, "2FA verification")
}

// issueTokenPair signs access and refresh JWTs. Claims: user_id, email, roles ([]string),
// session_id, is_superuser (bool), exp, type ("access"|"refresh").
func (h *Handlers) issueTokenPair(userID, email string, roles []string, sessionID string, isSuperuser bool) (accessToken, refreshToken string, expiresAt time.Time, err error) {
	now := time.Now()
	accessExp := now.Add(24 * time.Hour)
	refreshExp := now.Add(7 * 24 * time.Hour)
	accessClaims := jwt.MapClaims{
		"user_id": userID, "email": email, "roles": roles, "session_id": sessionID,
		"is_superuser": isSuperuser,
		"exp":          accessExp.Unix(), "type": "access",
	}
	refreshClaims := jwt.MapClaims{
		"user_id": userID, "email": email, "roles": roles, "session_id": sessionID,
		"is_superuser": isSuperuser,
		"exp":          refreshExp.Unix(), "type": "refresh",
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

// fiber:context-methods migrated
