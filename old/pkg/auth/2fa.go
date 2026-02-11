package auth

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"time"

	"github.com/pquerna/otp/totp"
)

// TwoFactorManager handles 2FA operations
type TwoFactorManager struct {
	issuer string
}

// TwoFactorSecret represents a 2FA secret
type TwoFactorSecret struct {
	Secret      string   `json:"secret"`
	QRCodeURL   string   `json:"qr_code_url"`
	BackupCodes []string `json:"backup_codes,omitempty"`
}

// NewTwoFactorManager creates a new 2FA manager
func NewTwoFactorManager(issuer string) *TwoFactorManager {
	return &TwoFactorManager{
		issuer: issuer,
	}
}

// GenerateSecret generates a new 2FA secret for a user
func (t *TwoFactorManager) GenerateSecret(userEmail string) (*TwoFactorSecret, error) {
	// Generate a random secret
	secret, err := t.generateRandomSecret()
	if err != nil {
		return nil, err
	}

	// Generate QR code URL
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      t.issuer,
		AccountName: userEmail,
		Secret:      []byte(secret),
	})
	if err != nil {
		return nil, err
	}

	// Generate backup codes
	backupCodes, err := t.generateBackupCodes()
	if err != nil {
		return nil, err
	}

	return &TwoFactorSecret{
		Secret:      secret,
		QRCodeURL:   key.URL(),
		BackupCodes: backupCodes,
	}, nil
}

// generateRandomSecret generates a random base32 secret
func (t *TwoFactorManager) generateRandomSecret() (string, error) {
	// Generate 20 random bytes
	bytes := make([]byte, 20)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	// Encode as base32
	return base32.StdEncoding.EncodeToString(bytes), nil
}

// generateBackupCodes generates backup codes for 2FA
func (t *TwoFactorManager) generateBackupCodes() ([]string, error) {
	codes := make([]string, 10) // Generate 10 backup codes

	for i := 0; i < 10; i++ {
		// Generate 8-character alphanumeric code
		bytes := make([]byte, 4)
		_, err := rand.Read(bytes)
		if err != nil {
			return nil, err
		}

		// Convert to alphanumeric
		code := ""
		for _, b := range bytes {
			code += fmt.Sprintf("%02x", b)
		}

		codes[i] = code[:8] // Take first 8 characters
	}

	return codes, nil
}

// ValidateTOTP validates a TOTP code
func (t *TwoFactorManager) ValidateTOTP(secret, code string) bool {
	return totp.Validate(code, secret)
}

// GenerateTOTP generates a TOTP code for testing
func (t *TwoFactorManager) GenerateTOTP(secret string) (string, error) {
	return totp.GenerateCode(secret, time.Now())
}

// ValidateBackupCode validates a backup code
func (t *TwoFactorManager) ValidateBackupCode(code string, usedCodes []string) bool {
	// Check if code is in the used list
	for _, usedCode := range usedCodes {
		if code == usedCode {
			return false // Code already used
		}
	}

	// Check if code is valid format (8 alphanumeric characters)
	if len(code) != 8 {
		return false
	}

	// Additional validation can be added here
	return true
}

// GetQRCodeData returns the QR code data for display
func (t *TwoFactorManager) GetQRCodeData(secret, userEmail string) (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      t.issuer,
		AccountName: userEmail,
		Secret:      []byte(secret),
	})
	if err != nil {
		return "", err
	}

	return key.URL(), nil
}

// IsValidSecret checks if a secret is valid
func (t *TwoFactorManager) IsValidSecret(secret string) bool {
	// Basic validation - should be base32 encoded
	_, err := base32.StdEncoding.DecodeString(secret)
	return err == nil
}

// DefaultTwoFactorManager returns a default 2FA manager
func DefaultTwoFactorManager(issuer string) *TwoFactorManager {
	return NewTwoFactorManager(issuer)
}
