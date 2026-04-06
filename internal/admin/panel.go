package admin

import (
	"os"
	"strings"
)

// PanelEnabled returns true when ENABLE_ADMIN_PANEL is truthy or .admin-panel-enabled exists.
func PanelEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("ENABLE_ADMIN_PANEL")))
	if v == "true" || v == "1" || v == "yes" {
		return true
	}
	_, err := os.Stat(".admin-panel-enabled")
	return err == nil
}
