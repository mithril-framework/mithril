// Command createsuperuser interactively creates a superuser (Django-style).
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/mithril-framework/mithril/database/models"
	"github.com/mithril-framework/mithril/database/repositories"
	csu "github.com/mithril-framework/mithril/internal/createsuperuser"
	"github.com/mithril-framework/mithril/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

func main() {
	loadEnvFile(".env")
	ctx := context.Background()
	if os.Getenv("DATABASE_URL") == "" && os.Getenv("DB_HOST") == "" {
		log.Fatal("Set DATABASE_URL or DB_HOST (and DB_USER, DB_PASSWORD, DB_NAME) in the environment or .env")
	}
	dsn := db.DSNFromEnv()
	pool, err := db.New(ctx, dsn)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	if err := requireACLSchema(ctx, pool); err != nil {
		log.Fatal(err)
	}

	repo := repositories.NewUserRepository(pool)
	sc := bufio.NewScanner(os.Stdin)

	fmt.Println("Superuser creation (mithril)")
	fmt.Println("Leave email empty to exit.")

	for {
		email := promptLine(sc, "Email address: ")
		if email == "" {
			fmt.Println("Cancelled.")
			return
		}
		email = strings.TrimSpace(strings.ToLower(email))
		if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
			fmt.Println("Error: Enter a valid email address.")
			continue
		}
		_, err := repo.GetByEmail(ctx, email)
		if err == nil {
			fmt.Println("Error: A user with that email already exists.")
			continue
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			log.Fatalf("database: %v", err)
		}

		password, ok := promptPasswordWithPolicy(sc)
		if !ok {
			continue
		}

		first := promptLine(sc, "First name: ")
		last := promptLine(sc, "Last name: ")
		first = strings.TrimSpace(first)
		last = strings.TrimSpace(last)

		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("bcrypt: %v", err)
		}
		u := &models.User{
			Email:        email,
			PasswordHash: string(hash),
			FirstName:    first,
			LastName:     last,
			IsActive:     true,
			IsSuperuser:  true,
		}
		if err := repo.Create(ctx, u); err != nil {
			log.Fatalf("create user: %v", err)
		}
		fmt.Printf("\nSuperuser created successfully.\n  id: %s\n  email: %s\n", u.ID, u.Email)
		return
	}
}

func promptLine(sc *bufio.Scanner, label string) string {
	fmt.Print(label)
	if !sc.Scan() {
		return ""
	}
	return sc.Text()
}

func readPassword(label string) (string, error) {
	fmt.Print(label)
	fd := int(os.Stdin.Fd())
	pw, err := term.ReadPassword(fd)
	fmt.Println()
	if err != nil {
		return "", err
	}
	return string(pw), nil
}

// promptPasswordWithPolicy asks twice for match, enforces min length, weak-password confirmation.
func promptPasswordWithPolicy(sc *bufio.Scanner) (string, bool) {
	for {
		pw1, err := readPassword("Password: ")
		if err != nil {
			log.Fatalf("read password: %v", err)
		}
		pw2, err := readPassword("Password (again): ")
		if err != nil {
			log.Fatalf("read password: %v", err)
		}
		if pw1 != pw2 {
			fmt.Println("Error: Passwords do not match. Try again.")
			continue
		}
		if len(pw1) < csu.MinPasswordLength {
			fmt.Printf("Error: Password must be at least %d characters. Try again.\n\n", csu.MinPasswordLength)
			continue
		}
		if issues := csu.PasswordIssues(pw1); len(issues) > 0 {
			fmt.Println("\nThis password does not meet the recommended complexity:")
			for _, line := range issues {
				fmt.Println("  -", line)
			}
			fmt.Print("\nUse this password anyway? Type 'yes' to confirm, or press Enter to choose another: ")
			if !sc.Scan() {
				return "", false
			}
			if strings.TrimSpace(strings.ToLower(sc.Text())) != "yes" {
				fmt.Println("Choose a new password.")
				continue
			}
		}
		return pw1, true
	}
}

func requireACLSchema(ctx context.Context, pool *pgxpool.Pool) error {
	var ok bool
	// Use pg_catalog + pg_table_is_visible so we detect the same `users` row type this session would use in SQL.
	// information_schema + broad schema filters can miss or mismatch search_path vs goose (public).
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM pg_catalog.pg_attribute a
			INNER JOIN pg_catalog.pg_class c ON c.oid = a.attrelid
			WHERE pg_catalog.pg_table_is_visible(c.oid)
			  AND c.relkind IN ('r', 'p')
			  AND c.relname = 'users'
			  AND a.attname = 'is_superuser'
			  AND a.attnum > 0
			  AND NOT a.attisdropped
		)
	`).Scan(&ok)
	if err != nil {
		return fmt.Errorf("database schema check: %w", err)
	}
	if !ok {
		var dbName, searchPath string
		_ = pool.QueryRow(ctx, `SELECT current_database(), current_setting('search_path')`).Scan(&dbName, &searchPath)
		return fmt.Errorf("connected database %q (search_path=%q) has no visible table users with column is_superuser.\n"+
			"If goose shows migrations applied, confirm DATABASE_URL / DB_* matches the DB you migrated (make migrate-status).\n"+
			"Otherwise run: make migrate-up", dbName, searchPath)
	}
	return nil
}

func loadEnvFile(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		value := strings.TrimSpace(line[idx+1:])
		value = strings.Trim(value, `"`)
		os.Setenv(key, value)
	}
}
