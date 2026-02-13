package utils

import (
	"regexp"
)

// IsValidEmail returns true if email matches a common email pattern.
func IsValidEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

// IsValidPhone returns true if phone has 10–15 digits (after stripping non-digits).
func IsValidPhone(phone string) bool {
	digits := regexp.MustCompile(`\D`).ReplaceAllString(phone, "")
	return len(digits) >= 10 && len(digits) <= 15
}

// IsValidURL returns true if url looks like http(s) URL.
func IsValidURL(url string) bool {
	pattern := `^https?://[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}(/.*)?$`
	matched, _ := regexp.MatchString(pattern, url)
	return matched
}

// IsStrongPassword returns true if password has at least 8 chars and upper, lower, digit, special.
func IsStrongPassword(password string) bool {
	if len(password) < 8 {
		return false
	}
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{}|;:,.<>?]`).MatchString(password)
	return hasUpper && hasLower && hasDigit && hasSpecial
}

// IsValidUsername returns true if username is 3–20 chars and alphanumeric/underscore.
func IsValidUsername(username string) bool {
	if len(username) < 3 || len(username) > 20 {
		return false
	}
	pattern := `^[a-zA-Z0-9_]+$`
	matched, _ := regexp.MatchString(pattern, username)
	return matched
}

// IsValidSlug returns true if slug is empty or matches [a-z0-9-]+.
func IsValidSlug(slug string) bool {
	if slug == "" {
		return true
	}
	pattern := `^[a-z0-9-]+$`
	matched, _ := regexp.MatchString(pattern, slug)
	return matched
}
