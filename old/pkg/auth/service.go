package auth

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// AuthService handles authentication operations
type AuthService struct {
	db               *gorm.DB
	jwtManager       *JWTManager
	passwordManager  *PasswordManager
	otpManager       *OTPManager
	twoFactorManager *TwoFactorManager
}

// NewAuthService creates a new authentication service
func NewAuthService(db *gorm.DB, jwtManager *JWTManager) *AuthService {
	return &AuthService{
		db:               db,
		jwtManager:       jwtManager,
		passwordManager:  DefaultPasswordManager(),
		otpManager:       DefaultOTPManager(),
		twoFactorManager: DefaultTwoFactorManager("Mithril"),
	}
}

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,password"`
	FirstName string `json:"first_name" validate:"required,min=2,max=50"`
	LastName  string `json:"last_name" validate:"required,min=2,max=50"`
	Phone     string `json:"phone" validate:"omitempty,phone"`
}

// LoginRequest represents a user login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	OTPCode  string `json:"otp_code,omitempty"`
	Remember bool   `json:"remember,omitempty"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	User        interface{} `json:"user"`
	Tokens      *TokenPair  `json:"tokens"`
	Requires2FA bool        `json:"requires_2fa"`
	RequiresOTP bool        `json:"requires_otp"`
}

// Register registers a new user
func (s *AuthService) Register(req *RegisterRequest) error {
	// Check if user already exists
	var existingUser struct {
		ID string
	}
	err := s.db.Table("users").Select("id").Where("email = ?", req.Email).First(&existingUser).Error
	if err == nil {
		return errors.New("user with this email already exists")
	}

	// Validate password strength
	if err := s.passwordManager.ValidatePasswordStrength(req.Password); err != nil {
		return err
	}

	// Hash password
	hashedPassword, err := s.passwordManager.HashPassword(req.Password)
	if err != nil {
		return err
	}

	// Create user
	user := map[string]interface{}{
		"email":      req.Email,
		"password":   hashedPassword,
		"first_name": req.FirstName,
		"last_name":  req.LastName,
		"phone":      req.Phone,
		"is_active":  true,
		"created_at": time.Now(),
		"updated_at": time.Now(),
	}

	if err := s.db.Table("users").Create(&user).Error; err != nil {
		return err
	}

	// TODO: Send email verification
	// TODO: Assign default role

	return nil
}

// Login authenticates a user
func (s *AuthService) Login(req *LoginRequest) (*LoginResponse, error) {
	// Find user by email
	var user struct {
		ID              string
		Email           string
		Password        string
		FirstName       string
		LastName        string
		IsActive        bool
		IsEmailVerified bool
		Is2FAEnabled    bool
		IsPhoneVerified bool
		Phone           string
	}

	err := s.db.Table("users").Where("email = ?", req.Email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid credentials")
		}
		return nil, err
	}

	// Check if user is active
	if !user.IsActive {
		return nil, errors.New("account is deactivated")
	}

	// Verify password
	valid, err := s.passwordManager.VerifyPassword(req.Password, user.Password)
	if err != nil || !valid {
		return nil, errors.New("invalid credentials")
	}

	// Check if 2FA is enabled
	if user.Is2FAEnabled && req.OTPCode == "" {
		return &LoginResponse{
			Requires2FA: true,
		}, nil
	}

	// Check if phone verification is required
	if user.Phone != "" && !user.IsPhoneVerified && req.OTPCode == "" {
		// Generate and send OTP
		otp, err := s.otpManager.GenerateOTP(user.Phone)
		if err != nil {
			return nil, err
		}

		// Store OTP in database
		otpRecord := map[string]interface{}{
			"user_id":    user.ID,
			"phone":      user.Phone,
			"code":       otp.Code,
			"expires_at": otp.ExpiresAt,
			"created_at": time.Now(),
		}

		if err := s.db.Table("otp_verifications").Create(&otpRecord).Error; err != nil {
			return nil, err
		}

		// Send OTP
		if err := s.otpManager.SendOTP(user.Phone, otp.Code); err != nil {
			return nil, err
		}

		return &LoginResponse{
			RequiresOTP: true,
		}, nil
	}

	// Generate tokens
	roles := []string{"user"} // TODO: Get actual roles from database
	sessionID := fmt.Sprintf("session_%d", time.Now().Unix())

	tokens, err := s.jwtManager.GenerateTokenPair(user.ID, user.Email, roles, sessionID)
	if err != nil {
		return nil, err
	}

	// Create session
	session := map[string]interface{}{
		"user_id":       user.ID,
		"token":         tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"is_active":     true,
		"expires_at":    tokens.ExpiresAt,
		"last_used_at":  time.Now(),
		"created_at":    time.Now(),
		"updated_at":    time.Now(),
	}

	if err := s.db.Table("sessions").Create(&session).Error; err != nil {
		return nil, err
	}

	// Update last login
	s.db.Table("users").Where("id = ?", user.ID).Update("last_login_at", time.Now())

	// Return response
	userResponse := map[string]interface{}{
		"id":                user.ID,
		"email":             user.Email,
		"first_name":        user.FirstName,
		"last_name":         user.LastName,
		"is_email_verified": user.IsEmailVerified,
		"is_2fa_enabled":    user.Is2FAEnabled,
	}

	return &LoginResponse{
		User:   userResponse,
		Tokens: tokens,
	}, nil
}

// Logout logs out a user
func (s *AuthService) Logout(token string) error {
	// Blacklist the token
	claims, err := s.jwtManager.ValidateToken(token)
	if err != nil {
		return err
	}

	// Mark session as inactive
	err = s.db.Table("sessions").Where("token = ?", token).Update("is_active", false).Error
	if err != nil {
		return err
	}

	// Blacklist token
	return s.jwtManager.BlacklistToken(token, claims.ExpiresAt.Time)
}

// RefreshToken refreshes an access token
func (s *AuthService) RefreshToken(refreshToken string) (*TokenPair, error) {
	// Validate refresh token
	claims, err := s.jwtManager.ValidateToken(refreshToken)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	// Check if session exists and is active
	var session struct {
		ID        string
		IsActive  bool
		ExpiresAt time.Time
	}

	err = s.db.Table("sessions").Where("refresh_token = ?", refreshToken).First(&session).Error
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	if !session.IsActive {
		return nil, errors.New("session is inactive")
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, errors.New("refresh token expired")
	}

	// Generate new tokens
	roles := []string{"user"} // TODO: Get actual roles from database
	newTokens, err := s.jwtManager.GenerateTokenPair(claims.UserID, claims.Email, roles, claims.SessionID)
	if err != nil {
		return nil, err
	}

	// Update session with new tokens
	updates := map[string]interface{}{
		"token":         newTokens.AccessToken,
		"refresh_token": newTokens.RefreshToken,
		"expires_at":    newTokens.ExpiresAt,
		"last_used_at":  time.Now(),
		"updated_at":    time.Now(),
	}

	err = s.db.Table("sessions").Where("id = ?", session.ID).Updates(updates).Error
	if err != nil {
		return nil, err
	}

	return newTokens, nil
}

// ForgotPassword initiates password reset
func (s *AuthService) ForgotPassword(email string) error {
	// Check if user exists
	var user struct {
		ID    string
		Email string
	}

	err := s.db.Table("users").Select("id, email").Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Don't reveal if user exists or not
			return nil
		}
		return err
	}

	// Generate reset token
	token := fmt.Sprintf("reset_%d", time.Now().Unix())
	expiresAt := time.Now().Add(24 * time.Hour) // 24 hours

	// Store reset token
	resetRecord := map[string]interface{}{
		"user_id":    user.ID,
		"token":      token,
		"email":      user.Email,
		"expires_at": expiresAt,
		"created_at": time.Now(),
	}

	if err := s.db.Table("password_resets").Create(&resetRecord).Error; err != nil {
		return err
	}

	// TODO: Send password reset email

	return nil
}

// ResetPassword resets a user's password
func (s *AuthService) ResetPassword(token, newPassword string) error {
	// Find reset record
	var resetRecord struct {
		ID        string
		UserID    string
		IsUsed    bool
		ExpiresAt time.Time
	}

	err := s.db.Table("password_resets").Where("token = ?", token).First(&resetRecord).Error
	if err != nil {
		return errors.New("invalid reset token")
	}

	if resetRecord.IsUsed {
		return errors.New("reset token already used")
	}

	if time.Now().After(resetRecord.ExpiresAt) {
		return errors.New("reset token expired")
	}

	// Validate new password
	if err := s.passwordManager.ValidatePasswordStrength(newPassword); err != nil {
		return err
	}

	// Hash new password
	hashedPassword, err := s.passwordManager.HashPassword(newPassword)
	if err != nil {
		return err
	}

	// Update password
	err = s.db.Table("users").Where("id = ?", resetRecord.UserID).Update("password", hashedPassword).Error
	if err != nil {
		return err
	}

	// Mark reset token as used
	err = s.db.Table("password_resets").Where("id = ?", resetRecord.ID).Update("is_used", true).Error
	if err != nil {
		return err
	}

	// Invalidate all user sessions
	err = s.db.Table("sessions").Where("user_id = ?", resetRecord.UserID).Update("is_active", false).Error
	if err != nil {
		return err
	}

	return nil
}
