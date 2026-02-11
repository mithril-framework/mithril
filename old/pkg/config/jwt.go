package config

import (
	"time"
)

// JWTConfig holds JWT configuration
type JWTConfig struct {
	// Secret key for signing tokens
	Secret string `env:"JWT_SECRET" required:"true" min:"32"`

	// Token expiration times
	AccessTokenExpiry  time.Duration `env:"JWT_ACCESS_TOKEN_EXPIRY" default:"15m"`
	RefreshTokenExpiry time.Duration `env:"JWT_REFRESH_TOKEN_EXPIRY" default:"7d"`

	// Algorithm for signing tokens
	Algorithm string `env:"JWT_ALGORITHM" default:"HS256"`

	// Issuer and audience
	Issuer   string `env:"JWT_ISSUER" default:"mithril"`
	Audience string `env:"JWT_AUDIENCE" default:"mithril-users"`

	// Token refresh settings
	RefreshThreshold time.Duration `env:"JWT_REFRESH_THRESHOLD" default:"5m"`

	// Cookie settings for refresh tokens
	CookieName     string        `env:"JWT_COOKIE_NAME" default:"refresh_token"`
	CookiePath     string        `env:"JWT_COOKIE_PATH" default:"/"`
	CookieDomain   string        `env:"JWT_COOKIE_DOMAIN" default:""`
	CookieSecure   bool          `env:"JWT_COOKIE_SECURE" default:"false"`
	CookieHTTPOnly bool          `env:"JWT_COOKIE_HTTP_ONLY" default:"true"`
	CookieSameSite string        `env:"JWT_COOKIE_SAME_SITE" default:"Lax"`
	CookieMaxAge   time.Duration `env:"JWT_COOKIE_MAX_AGE" default:"604800"` // 7 days

	// Blacklist settings
	BlacklistEnabled bool          `env:"JWT_BLACKLIST_ENABLED" default:"true"`
	BlacklistTTL     time.Duration `env:"JWT_BLACKLIST_TTL" default:"24h"`

	// Rate limiting for token generation
	TokenRateLimitEnabled bool          `env:"JWT_RATE_LIMIT_ENABLED" default:"true"`
	TokenRateLimitRPS     int           `env:"JWT_RATE_LIMIT_RPS" default:"10"`
	TokenRateLimitBurst   int           `env:"JWT_RATE_LIMIT_BURST" default:"20"`
	TokenRateLimitWindow  time.Duration `env:"JWT_RATE_LIMIT_WINDOW" default:"1m"`

	// 2FA settings
	TwoFactorEnabled bool          `env:"JWT_2FA_ENABLED" default:"true"`
	TwoFactorExpiry  time.Duration `env:"JWT_2FA_EXPIRY" default:"5m"`

	// Password reset settings
	PasswordResetExpiry time.Duration `env:"JWT_PASSWORD_RESET_EXPIRY" default:"1h"`

	// Email verification settings
	EmailVerificationExpiry time.Duration `env:"JWT_EMAIL_VERIFICATION_EXPIRY" default:"24h"`
}

// GetAccessTokenExpiry returns access token expiry duration
func (c *JWTConfig) GetAccessTokenExpiry() time.Duration {
	if c.AccessTokenExpiry <= 0 {
		return 15 * time.Minute
	}
	return c.AccessTokenExpiry
}

// GetRefreshTokenExpiry returns refresh token expiry duration
func (c *JWTConfig) GetRefreshTokenExpiry() time.Duration {
	if c.RefreshTokenExpiry <= 0 {
		return 7 * 24 * time.Hour
	}
	return c.RefreshTokenExpiry
}

// GetAlgorithm returns the JWT algorithm
func (c *JWTConfig) GetAlgorithm() string {
	if c.Algorithm == "" {
		return "HS256"
	}
	return c.Algorithm
}

// GetIssuer returns the JWT issuer
func (c *JWTConfig) GetIssuer() string {
	if c.Issuer == "" {
		return "mithril"
	}
	return c.Issuer
}

// GetAudience returns the JWT audience
func (c *JWTConfig) GetAudience() string {
	if c.Audience == "" {
		return "mithril-users"
	}
	return c.Audience
}

// GetRefreshThreshold returns the refresh threshold
func (c *JWTConfig) GetRefreshThreshold() time.Duration {
	if c.RefreshThreshold <= 0 {
		return 5 * time.Minute
	}
	return c.RefreshThreshold
}

// GetCookieName returns the refresh token cookie name
func (c *JWTConfig) GetCookieName() string {
	if c.CookieName == "" {
		return "refresh_token"
	}
	return c.CookieName
}

// GetCookiePath returns the refresh token cookie path
func (c *JWTConfig) GetCookiePath() string {
	if c.CookiePath == "" {
		return "/"
	}
	return c.CookiePath
}

// GetCookieDomain returns the refresh token cookie domain
func (c *JWTConfig) GetCookieDomain() string {
	return c.CookieDomain
}

// IsCookieSecure returns whether the cookie should be secure
func (c *JWTConfig) IsCookieSecure() bool {
	return c.CookieSecure
}

// IsCookieHTTPOnly returns whether the cookie should be HTTP only
func (c *JWTConfig) IsCookieHTTPOnly() bool {
	return c.CookieHTTPOnly
}

// GetCookieSameSite returns the cookie SameSite setting
func (c *JWTConfig) GetCookieSameSite() string {
	if c.CookieSameSite == "" {
		return "Lax"
	}
	return c.CookieSameSite
}

// GetCookieMaxAge returns the cookie max age in seconds
func (c *JWTConfig) GetCookieMaxAge() int {
	if c.CookieMaxAge <= 0 {
		return int(7 * 24 * time.Hour.Seconds())
	}
	return int(c.CookieMaxAge.Seconds())
}

// IsBlacklistEnabled returns whether token blacklisting is enabled
func (c *JWTConfig) IsBlacklistEnabled() bool {
	return c.BlacklistEnabled
}

// GetBlacklistTTL returns the blacklist TTL
func (c *JWTConfig) GetBlacklistTTL() time.Duration {
	if c.BlacklistTTL <= 0 {
		return 24 * time.Hour
	}
	return c.BlacklistTTL
}

// IsTokenRateLimitEnabled returns whether token rate limiting is enabled
func (c *JWTConfig) IsTokenRateLimitEnabled() bool {
	return c.TokenRateLimitEnabled
}

// GetTokenRateLimitRPS returns the token rate limit RPS
func (c *JWTConfig) GetTokenRateLimitRPS() int {
	if c.TokenRateLimitRPS <= 0 {
		return 10
	}
	return c.TokenRateLimitRPS
}

// GetTokenRateLimitBurst returns the token rate limit burst
func (c *JWTConfig) GetTokenRateLimitBurst() int {
	if c.TokenRateLimitBurst <= 0 {
		return 20
	}
	return c.TokenRateLimitBurst
}

// GetTokenRateLimitWindow returns the token rate limit window
func (c *JWTConfig) GetTokenRateLimitWindow() time.Duration {
	if c.TokenRateLimitWindow <= 0 {
		return time.Minute
	}
	return c.TokenRateLimitWindow
}

// IsTwoFactorEnabled returns whether 2FA is enabled
func (c *JWTConfig) IsTwoFactorEnabled() bool {
	return c.TwoFactorEnabled
}

// GetTwoFactorExpiry returns the 2FA token expiry
func (c *JWTConfig) GetTwoFactorExpiry() time.Duration {
	if c.TwoFactorExpiry <= 0 {
		return 5 * time.Minute
	}
	return c.TwoFactorExpiry
}

// GetPasswordResetExpiry returns the password reset token expiry
func (c *JWTConfig) GetPasswordResetExpiry() time.Duration {
	if c.PasswordResetExpiry <= 0 {
		return time.Hour
	}
	return c.PasswordResetExpiry
}

// GetEmailVerificationExpiry returns the email verification token expiry
func (c *JWTConfig) GetEmailVerificationExpiry() time.Duration {
	if c.EmailVerificationExpiry <= 0 {
		return 24 * time.Hour
	}
	return c.EmailVerificationExpiry
}
