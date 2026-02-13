package utils

import (
	"crypto/rand"
	"math/big"
	"regexp"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

// GenerateRandomString returns a random string of the given length from alphanumeric charset.
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[num.Int64()]
	}
	return string(b)
}

// GenerateRandomBytes returns n random bytes from crypto/rand.
func GenerateRandomBytes(length int) []byte {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return bytes
}

// Slugify converts a string to a URL-friendly slug (lowercase, hyphens).
func Slugify(s string) string {
	s = strings.ToLower(s)
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	s = reg.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

// Truncate truncates s to length and appends "..." if truncated.
func Truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}

// TruncateWords truncates to wordCount words and appends "..." if truncated.
func TruncateWords(s string, wordCount int) string {
	words := strings.Fields(s)
	if len(words) <= wordCount {
		return s
	}
	return strings.Join(words[:wordCount], " ") + "..."
}

// Capitalize uppercases the first rune of s.
func Capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// TitleCase capitalizes the first letter of each word.
func TitleCase(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		words[i] = Capitalize(word)
	}
	return strings.Join(words, " ")
}

// SnakeCase converts to snake_case.
func SnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			result = append(result, '_')
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}

// CamelCase converts to camelCase.
func CamelCase(s string) string {
	words := strings.Fields(strings.ReplaceAll(s, "_", " "))
	if len(words) == 0 {
		return ""
	}
	result := strings.ToLower(words[0])
	for _, word := range words[1:] {
		result += Capitalize(word)
	}
	return result
}

// PascalCase converts to PascalCase.
func PascalCase(s string) string {
	words := strings.Fields(strings.ReplaceAll(s, "_", " "))
	result := ""
	for _, word := range words {
		result += Capitalize(word)
	}
	return result
}

// RemoveDuplicates returns a new slice with duplicate strings removed (order preserved).
func RemoveDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	result := []string{}
	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	return result
}

// IsEmpty returns true if s is empty or only whitespace.
func IsEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}

// IsNotEmpty returns true if s has non-whitespace content.
func IsNotEmpty(s string) bool {
	return !IsEmpty(s)
}

// GenerateUUID returns a new UUID v4 string.
func GenerateUUID() string {
	return uuid.New().String()
}

// GenerateShortUUID returns the first 12 characters of a UUID without hyphens.
func GenerateShortUUID() string {
	return strings.ReplaceAll(GenerateUUID(), "-", "")[:12]
}
