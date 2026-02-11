package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTManager handles JWT token operations
type JWTManager struct {
	secretKey     string
	tokenDuration time.Duration
	issuer        string
}

// Claims represents JWT claims
type Claims struct {
	UserID    string   `json:"user_id"`
	Email     string   `json:"email"`
	Roles     []string `json:"roles"`
	SessionID string   `json:"session_id"`
	jwt.RegisteredClaims
}

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(secretKey string, tokenDuration time.Duration, issuer string) *JWTManager {
	return &JWTManager{
		secretKey:     secretKey,
		tokenDuration: tokenDuration,
		issuer:        issuer,
	}
}

// GenerateTokenPair generates both access and refresh tokens
func (j *JWTManager) GenerateTokenPair(userID, email string, roles []string, sessionID string) (*TokenPair, error) {
	now := time.Now()
	expiresAt := now.Add(j.tokenDuration)

	// Generate access token
	accessToken, err := j.generateToken(userID, email, roles, sessionID, expiresAt, "access")
	if err != nil {
		return nil, err
	}

	// Generate refresh token (longer duration)
	refreshExpiresAt := now.Add(j.tokenDuration * 24 * 7) // 7 days
	refreshToken, err := j.generateToken(userID, email, roles, sessionID, refreshExpiresAt, "refresh")
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    "Bearer",
	}, nil
}

// generateToken generates a JWT token
func (j *JWTManager) generateToken(userID, email string, roles []string, sessionID string, expiresAt time.Time, tokenType string) (string, error) {
	claims := &Claims{
		UserID:    userID,
		Email:     email,
		Roles:     roles,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    j.issuer,
			Subject:   userID,
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.secretKey))
}

// ValidateToken validates and parses a JWT token
func (j *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// RefreshToken generates a new access token from a refresh token
func (j *JWTManager) RefreshToken(refreshTokenString string) (*TokenPair, error) {
	claims, err := j.ValidateToken(refreshTokenString)
	if err != nil {
		return nil, err
	}

	// Check if it's a refresh token
	if claims.ID == "" {
		return nil, errors.New("invalid refresh token")
	}

	// Generate new token pair
	return j.GenerateTokenPair(claims.UserID, claims.Email, claims.Roles, claims.SessionID)
}

// ExtractTokenFromHeader extracts token from Authorization header
func (j *JWTManager) ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", errors.New("authorization header is required")
	}

	// Check if it starts with "Bearer "
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return "", errors.New("authorization header must start with 'Bearer '")
	}

	return authHeader[7:], nil
}

// GetTokenExpiration returns the token expiration time
func (j *JWTManager) GetTokenExpiration() time.Duration {
	return j.tokenDuration
}

// BlacklistToken adds a token to the blacklist (in a real implementation, you'd use Redis)
func (j *JWTManager) BlacklistToken(tokenString string, expiresAt time.Time) error {
	// TODO: Implement token blacklisting with Redis
	// For now, we'll rely on token expiration
	return nil
}

// IsTokenBlacklisted checks if a token is blacklisted
func (j *JWTManager) IsTokenBlacklisted(tokenString string) bool {
	// TODO: Implement token blacklist checking with Redis
	// For now, we'll rely on token expiration
	return false
}
