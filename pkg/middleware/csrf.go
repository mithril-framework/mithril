package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
)

// CSRFConfig holds configuration for CSRF middleware
type CSRFConfig struct {
	TokenLength    int                           // Length of CSRF token
	TokenName      string                        // Name of the token in form/header
	CookieName     string                        // Name of the cookie to store token
	CookiePath     string                        // Path for the cookie
	CookieDomain   string                        // Domain for the cookie
	CookieSecure   bool                          // Secure flag for cookie
	CookieHTTPOnly bool                          // HTTPOnly flag for cookie
	CookieSameSite string                        // SameSite attribute for cookie
	Expiration     time.Duration                 // Token expiration time
	HeaderName     string                        // Header name for token
	ErrorHandler   func(*fiber.Ctx, error) error // Error handler
}

// DefaultCSRFConfig returns default CSRF configuration
func DefaultCSRFConfig() CSRFConfig {
	return CSRFConfig{
		TokenLength:    32,
		TokenName:      "_token",
		CookieName:     "csrf_token",
		CookiePath:     "/",
		CookieDomain:   "",
		CookieSecure:   false, // Set to true in production with HTTPS
		CookieHTTPOnly: true,
		CookieSameSite: "Lax",
		Expiration:     24 * time.Hour,
		HeaderName:     "X-CSRF-Token",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(403).JSON(fiber.Map{
				"error":   "CSRF token mismatch",
				"message": "Invalid or missing CSRF token",
			})
		},
	}
}

// CSRF implements CSRF protection middleware
type CSRF struct {
	config CSRFConfig
}

// NewCSRF creates a new CSRF middleware
func NewCSRF(config CSRFConfig) *CSRF {
	return &CSRF{
		config: config,
	}
}

// generateToken generates a random CSRF token
func (c *CSRF) generateToken() (string, error) {
	bytes := make([]byte, c.config.TokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// setTokenCookie sets the CSRF token cookie
func (c *CSRF) setTokenCookie(ctx *fiber.Ctx, token string) {
	cookie := &fiber.Cookie{
		Name:     c.config.CookieName,
		Value:    token,
		Path:     c.config.CookiePath,
		Domain:   c.config.CookieDomain,
		Secure:   c.config.CookieSecure,
		HTTPOnly: c.config.CookieHTTPOnly,
		SameSite: c.config.CookieSameSite,
		Expires:  time.Now().Add(c.config.Expiration),
	}
	ctx.Cookie(cookie)
}

// getTokenFromRequest retrieves the CSRF token from request
func (c *CSRF) getTokenFromRequest(ctx *fiber.Ctx) string {
	// Try to get token from header first
	if token := ctx.Get(c.config.HeaderName); token != "" {
		return token
	}

	// Try to get token from form data
	if token := ctx.FormValue(c.config.TokenName); token != "" {
		return token
	}

	// Try to get token from query parameter
	if token := ctx.Query(c.config.TokenName); token != "" {
		return token
	}

	return ""
}

// getTokenFromCookie retrieves the CSRF token from cookie
func (c *CSRF) getTokenFromCookie(ctx *fiber.Ctx) string {
	return ctx.Cookies(c.config.CookieName)
}

// validateToken validates the CSRF token
func (c *CSRF) validateToken(ctx *fiber.Ctx) bool {
	requestToken := c.getTokenFromRequest(ctx)
	cookieToken := c.getTokenFromCookie(ctx)

	if requestToken == "" || cookieToken == "" {
		return false
	}

	// Use constant time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare([]byte(requestToken), []byte(cookieToken)) == 1
}

// isExempt checks if the request is exempt from CSRF protection
func (c *CSRF) isExempt(ctx *fiber.Ctx) bool {
	// Exempt GET, HEAD, OPTIONS requests
	if ctx.Method() == "GET" || ctx.Method() == "HEAD" || ctx.Method() == "OPTIONS" {
		return true
	}

	// Add more exemption logic here if needed
	// For example, exempt certain paths or API endpoints

	return false
}

// Handler returns the CSRF middleware handler
func (c *CSRF) Handler() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		// Check if request is exempt
		if c.isExempt(ctx) {
			return ctx.Next()
		}

		// Generate new token if not exists
		cookieToken := c.getTokenFromCookie(ctx)
		if cookieToken == "" {
			token, err := c.generateToken()
			if err != nil {
				return c.config.ErrorHandler(ctx, err)
			}
			c.setTokenCookie(ctx, token)
			cookieToken = token
		}

		// Validate token for state-changing requests
		if !c.validateToken(ctx) {
			return c.config.ErrorHandler(ctx, fmt.Errorf("CSRF token validation failed"))
		}

		// Store token in locals for use in templates
		ctx.Locals("csrf_token", cookieToken)

		return ctx.Next()
	}
}

// CSRFMiddleware creates a CSRF middleware with default configuration
func CSRFMiddleware(config ...CSRFConfig) fiber.Handler {
	cfg := DefaultCSRFConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	csrf := NewCSRF(cfg)
	return csrf.Handler()
}

// GetCSRFToken retrieves the CSRF token from context
func GetCSRFToken(ctx *fiber.Ctx) string {
	if token := ctx.Locals("csrf_token"); token != nil {
		return token.(string)
	}
	return ""
}

// CSRFTokenField generates a hidden input field for CSRF token
func CSRFTokenField(ctx *fiber.Ctx) string {
	token := GetCSRFToken(ctx)
	if token == "" {
		return ""
	}
	return fmt.Sprintf(`<input type="hidden" name="%s" value="%s">`, "csrf_token", token)
}

// CSRFMetaTag generates a meta tag for CSRF token
func CSRFMetaTag(ctx *fiber.Ctx) string {
	token := GetCSRFToken(ctx)
	if token == "" {
		return ""
	}
	return fmt.Sprintf(`<meta name="csrf-token" content="%s">`, token)
}
