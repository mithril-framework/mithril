package db

import (
	"fmt"
	"os"
	"strconv"
)

// DSNFromEnv returns a PostgreSQL connection string from environment.
// Prefers DATABASE_URL; if unset, builds from DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, DB_SSLMODE.
func DSNFromEnv() string {
	if s := os.Getenv("DATABASE_URL"); s != "" {
		return s
	}
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	if _, err := strconv.Atoi(port); err != nil {
		port = "5432"
	}
	user := getEnv("DB_USER", "postgres")
	password := os.Getenv("DB_PASSWORD")
	dbname := getEnv("DB_NAME", "mithril_rev")
	sslmode := getEnv("DB_SSLMODE", "disable")
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, password, host, port, dbname, sslmode)
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
