package createsuperuser

import (
	"unicode"
)

// MinPasswordLength is the minimum length; shorter passwords are rejected (no override).
const MinPasswordLength = 8

// PasswordIssues returns human-readable recommendations. If any are returned, the password
// is considered "weak" for policy purposes and the operator should confirm before use.
// PasswordIssues lists quality recommendations (length is already >= MinPasswordLength).
func PasswordIssues(password string) []string {
	var lines []string
	if len(password) < MinPasswordLength {
		return nil
	}
	var hasLower, hasUpper, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}
	if !hasLower {
		lines = append(lines, "Use at least one lowercase letter.")
	}
	if !hasUpper {
		lines = append(lines, "Use at least one uppercase letter.")
	}
	if !hasDigit {
		lines = append(lines, "Use at least one digit.")
	}
	if !hasSpecial {
		lines = append(lines, "Use at least one special character (e.g. !@#$%).")
	}
	if len(password) < 12 {
		lines = append(lines, "Use at least 12 characters for a stronger password.")
	}
	return lines
}
