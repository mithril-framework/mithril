package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

// OTPManager handles OTP generation and validation
type OTPManager struct {
	codeLength    int
	expiryMinutes int
}

// OTPCode represents an OTP code
type OTPCode struct {
	Code      string    `json:"code"`
	Phone     string    `json:"phone"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// NewOTPManager creates a new OTP manager
func NewOTPManager(codeLength, expiryMinutes int) *OTPManager {
	return &OTPManager{
		codeLength:    codeLength,
		expiryMinutes: expiryMinutes,
	}
}

// GenerateOTP generates a new OTP code
func (o *OTPManager) GenerateOTP(phone string) (*OTPCode, error) {
	code, err := o.generateRandomCode()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(o.expiryMinutes) * time.Minute)

	return &OTPCode{
		Code:      code,
		Phone:     phone,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}, nil
}

// generateRandomCode generates a random numeric code
func (o *OTPManager) generateRandomCode() (string, error) {
	// Calculate the maximum value for the given length
	max := big.NewInt(1)
	for i := 0; i < o.codeLength; i++ {
		max.Mul(max, big.NewInt(10))
	}
	max.Sub(max, big.NewInt(1))

	// Generate random number
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}

	// Format with leading zeros
	format := fmt.Sprintf("%%0%dd", o.codeLength)
	return fmt.Sprintf(format, n.Int64()), nil
}

// ValidateOTP validates an OTP code
func (o *OTPManager) ValidateOTP(code, phone string, storedCode *OTPCode) error {
	// Check if code matches
	if code != storedCode.Code {
		return fmt.Errorf("invalid OTP code")
	}

	// Check if phone matches
	if phone != storedCode.Phone {
		return fmt.Errorf("phone number mismatch")
	}

	// Check if code is expired
	if time.Now().After(storedCode.ExpiresAt) {
		return fmt.Errorf("OTP code has expired")
	}

	return nil
}

// IsExpired checks if an OTP code is expired
func (o *OTPManager) IsExpired(otp *OTPCode) bool {
	return time.Now().After(otp.ExpiresAt)
}

// GetExpiryDuration returns the OTP expiry duration
func (o *OTPManager) GetExpiryDuration() time.Duration {
	return time.Duration(o.expiryMinutes) * time.Minute
}

// FormatPhoneNumber formats a phone number for OTP
func (o *OTPManager) FormatPhoneNumber(phone string) string {
	// Remove all non-digit characters
	cleaned := ""
	for _, char := range phone {
		if char >= '0' && char <= '9' {
			cleaned += string(char)
		}
	}

	// Add country code if not present (assuming US +1)
	if len(cleaned) == 10 {
		cleaned = "1" + cleaned
	}

	return "+" + cleaned
}

// SendOTP sends an OTP code (placeholder - implement with SMS provider)
func (o *OTPManager) SendOTP(phone, code string) error {
	// TODO: Implement SMS sending with providers like Twilio, AWS SNS, etc.
	// For now, just log the code
	fmt.Printf("Sending OTP %s to %s\n", code, phone)
	return nil
}

// DefaultOTPManager returns a default OTP manager
func DefaultOTPManager() *OTPManager {
	return NewOTPManager(6, 5) // 6-digit code, 5 minutes expiry
}
