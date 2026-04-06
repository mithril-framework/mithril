// Package timezone sets the process default timezone from environment variables.
// Call InitFromEnv immediately after loading .env so time.Local and time.Now match APP_TIMEZONE or TZ.
package timezone

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// InitFromEnv sets TZ for the process from APP_TIMEZONE (if set), else TZ, else UTC.
// The value must be a valid IANA zone name (e.g. America/New_York, UTC).
func InitFromEnv() error {
	zone := strings.TrimSpace(os.Getenv("APP_TIMEZONE"))
	if zone == "" {
		zone = strings.TrimSpace(os.Getenv("TZ"))
	}
	if zone == "" {
		zone = "UTC"
	}
	if _, err := time.LoadLocation(zone); err != nil {
		return fmt.Errorf("timezone: invalid zone %q: %w", zone, err)
	}
	if err := os.Setenv("TZ", zone); err != nil {
		return fmt.Errorf("timezone: set TZ: %w", err)
	}
	return nil
}
