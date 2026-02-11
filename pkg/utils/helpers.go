package utils

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// String utilities

// IsEmpty checks if a string is empty or contains only whitespace
func IsEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}

// IsNotEmpty checks if a string is not empty and contains non-whitespace characters
func IsNotEmpty(s string) bool {
	return !IsEmpty(s)
}

// Truncate truncates a string to the specified length
func Truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}

// TruncateWords truncates a string to the specified number of words
func TruncateWords(s string, wordCount int) string {
	words := strings.Fields(s)
	if len(words) <= wordCount {
		return s
	}
	return strings.Join(words[:wordCount], " ") + "..."
}

// Contains checks if a string contains a substring (case-insensitive)
func Contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// ContainsAny checks if a string contains any of the substrings
func ContainsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if Contains(s, substr) {
			return true
		}
	}
	return false
}

// RemoveDuplicates removes duplicate strings from a slice
func RemoveDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}

// Number utilities

// ParseInt safely parses a string to int with default value
func ParseInt(s string, defaultValue int) int {
	if value, err := strconv.Atoi(s); err == nil {
		return value
	}
	return defaultValue
}

// ParseFloat safely parses a string to float64 with default value
func ParseFloat(s string, defaultValue float64) float64 {
	if value, err := strconv.ParseFloat(s, 64); err == nil {
		return value
	}
	return defaultValue
}

// ParseBool safely parses a string to bool with default value
func ParseBool(s string, defaultValue bool) bool {
	if value, err := strconv.ParseBool(s); err == nil {
		return value
	}
	return defaultValue
}

// Min returns the minimum of two integers
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Max returns the maximum of two integers
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Clamp clamps a value between min and max
func Clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// Round rounds a float64 to the specified number of decimal places
func Round(value float64, places int) float64 {
	multiplier := math.Pow(10, float64(places))
	return math.Round(value*multiplier) / multiplier
}

// FormatBytes formats bytes into human-readable format
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Time utilities

// FormatDuration formats a duration into human-readable format
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
	return fmt.Sprintf("%.1fd", d.Hours()/24)
}

// TimeAgo returns a human-readable time ago string
func TimeAgo(t time.Time) string {
	now := time.Now()
	duration := now.Sub(t)

	if duration < time.Minute {
		return "just now"
	}
	if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	}
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}
	if duration < 30*24*time.Hour {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
	if duration < 365*24*time.Hour {
		months := int(duration.Hours() / (24 * 30))
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
	years := int(duration.Hours() / (24 * 365))
	if years == 1 {
		return "1 year ago"
	}
	return fmt.Sprintf("%d years ago", years)
}

// IsValidEmail checks if a string is a valid email address
func IsValidEmail(email string) bool {
	// Simple email validation
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

// IsValidURL checks if a string is a valid URL
func IsValidURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

// Slice utilities

// Contains checks if a slice contains a value
func SliceContains(slice interface{}, value interface{}) bool {
	sliceValue := reflect.ValueOf(slice)
	if sliceValue.Kind() != reflect.Slice {
		return false
	}

	for i := 0; i < sliceValue.Len(); i++ {
		if reflect.DeepEqual(sliceValue.Index(i).Interface(), value) {
			return true
		}
	}
	return false
}

// Filter filters a slice based on a predicate function
func Filter(slice interface{}, predicate func(interface{}) bool) interface{} {
	sliceValue := reflect.ValueOf(slice)
	if sliceValue.Kind() != reflect.Slice {
		return slice
	}

	sliceType := sliceValue.Type()
	result := reflect.MakeSlice(sliceType, 0, 0)

	for i := 0; i < sliceValue.Len(); i++ {
		item := sliceValue.Index(i).Interface()
		if predicate(item) {
			result = reflect.Append(result, sliceValue.Index(i))
		}
	}

	return result.Interface()
}

// Map applies a function to each element of a slice
func Map(slice interface{}, mapper func(interface{}) interface{}) interface{} {
	sliceValue := reflect.ValueOf(slice)
	if sliceValue.Kind() != reflect.Slice {
		return slice
	}

	sliceType := sliceValue.Type()
	result := reflect.MakeSlice(sliceType, sliceValue.Len(), sliceValue.Len())

	for i := 0; i < sliceValue.Len(); i++ {
		item := sliceValue.Index(i).Interface()
		mapped := mapper(item)
		result.Index(i).Set(reflect.ValueOf(mapped))
	}

	return result.Interface()
}

// Reduce reduces a slice to a single value
func Reduce(slice interface{}, initial interface{}, reducer func(interface{}, interface{}) interface{}) interface{} {
	sliceValue := reflect.ValueOf(slice)
	if sliceValue.Kind() != reflect.Slice {
		return initial
	}

	result := initial
	for i := 0; i < sliceValue.Len(); i++ {
		item := sliceValue.Index(i).Interface()
		result = reducer(result, item)
	}

	return result
}

// Chunk splits a slice into chunks of specified size
func Chunk(slice interface{}, chunkSize int) interface{} {
	sliceValue := reflect.ValueOf(slice)
	if sliceValue.Kind() != reflect.Slice {
		return slice
	}

	sliceType := sliceValue.Type()
	chunkType := reflect.SliceOf(sliceType)
	result := reflect.MakeSlice(reflect.SliceOf(chunkType), 0, 0)

	for i := 0; i < sliceValue.Len(); i += chunkSize {
		end := i + chunkSize
		if end > sliceValue.Len() {
			end = sliceValue.Len()
		}

		chunk := sliceValue.Slice(i, end)
		result = reflect.Append(result, chunk)
	}

	return result.Interface()
}

// Validation utilities

// IsValidUUID checks if a string is a valid UUID
func IsValidUUID(uuid string) bool {
	// Simple UUID validation (v4 format)
	parts := strings.Split(uuid, "-")
	if len(parts) != 5 {
		return false
	}

	expectedLengths := []int{8, 4, 4, 4, 12}
	for i, part := range parts {
		if len(part) != expectedLengths[i] {
			return false
		}
		// Check if all characters are hexadecimal
		for _, char := range part {
			if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
				return false
			}
		}
	}

	return true
}

// IsValidPhoneNumber checks if a string is a valid phone number
func IsValidPhoneNumber(phone string) bool {
	// Remove all non-digit characters
	digits := strings.ReplaceAll(phone, " ", "")
	digits = strings.ReplaceAll(digits, "-", "")
	digits = strings.ReplaceAll(digits, "(", "")
	digits = strings.ReplaceAll(digits, ")", "")
	digits = strings.ReplaceAll(digits, "+", "")

	// Check if it contains only digits and has reasonable length
	if len(digits) < 10 || len(digits) > 15 {
		return false
	}

	for _, char := range digits {
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}

// IsStrongPassword checks if a password meets strength requirements
func IsStrongPassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", char):
			hasSpecial = true
		}
	}

	return hasUpper && hasLower && hasDigit && hasSpecial
}
