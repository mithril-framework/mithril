package db

import "github.com/mithril-framework/mithril/config"

// DSNFromEnv returns the PostgreSQL connection string from the environment.
// Deprecated: use config.DSNFromEnv (kept for the bundled CLIs).
func DSNFromEnv() string { return config.DSNFromEnv() }
