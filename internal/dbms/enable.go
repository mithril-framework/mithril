package dbms

import (
	"os"
	"strings"
)

// Enabled returns true when ENABLE_DBMS is truthy or .dbms-enabled exists.
func Enabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("ENABLE_DBMS")))
	if v == "true" || v == "1" || v == "yes" {
		return true
	}
	_, err := os.Stat(".dbms-enabled")
	return err == nil
}
