// Package createsuperuser implements the interactive / non-interactive superuser command.
package createsuperuser

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mithril-framework/mithril/database/models"
	"github.com/mithril-framework/mithril/database/repositories"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

func fail(args ...any) error { return errors.New(fmt.Sprint(args...)) }

// Run creates a superuser. Flags: --email, --password, --first-name, --last-name (or
// CREATESUPERUSER_EMAIL / CREATESUPERUSER_PASSWORD); with neither it prompts interactively.
func Run(ctx context.Context, pool *pgxpool.Pool, args []string) error {
	fs := flag.NewFlagSet("createsuperuser", flag.ContinueOnError)

	emailFlag := fs.String("email", "", "Superuser email (non-interactive)")
	passwordFlag := fs.String("password", "", "Superuser password (non-interactive; min 8 chars)")
	firstNameFlag := fs.String("first-name", "", "First name (optional)")
	lastNameFlag := fs.String("last-name", "", "Last name (optional)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if err := requireACLSchema(ctx, pool); err != nil {
		return fail(err)
	}

	repo := repositories.NewUserRepository(pool)

	email := strings.TrimSpace(strings.ToLower(firstNonEmpty(*emailFlag, os.Getenv("CREATESUPERUSER_EMAIL"))))
	password := firstNonEmpty(*passwordFlag, os.Getenv("CREATESUPERUSER_PASSWORD"))
	if email != "" && password != "" {
		if err := createSuperuser(ctx, repo, email, password, strings.TrimSpace(*firstNameFlag), strings.TrimSpace(*lastNameFlag)); err != nil {
			return fail(err)
		}
		return nil
	}
	if email != "" || password != "" {
		return fail("Non-interactive mode requires both --email and --password (or CREATESUPERUSER_EMAIL and CREATESUPERUSER_PASSWORD)")
	}

	sc := bufio.NewScanner(os.Stdin)

	fmt.Println("Superuser creation (mithril)")
	fmt.Println("Leave email empty to exit.")
	fmt.Println("Non-interactive: mithril createsuperuser --email you@example.com --password 'your-password'")

	for {
		email := promptLine(sc, "Email address: ")
		if email == "" {
			fmt.Println("Cancelled.")
			return nil
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
			return fmt.Errorf("database: %v", err)
		}

		password, ok := promptPasswordWithPolicy(sc)
		if !ok {
			continue
		}

		first := promptLine(sc, "First name: ")
		last := promptLine(sc, "Last name: ")
		first = strings.TrimSpace(first)
		last = strings.TrimSpace(last)

		if err := createSuperuser(ctx, repo, email, password, first, last); err != nil {
			return fail(err)
		}
		return nil
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func createSuperuser(ctx context.Context, repo *repositories.UserRepository, email, password, first, last string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return fmt.Errorf("invalid email address: %s", email)
	}
	if len(password) < MinPasswordLength {
		return fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	}
	_, err := repo.GetByEmail(ctx, email)
	if err == nil {
		return fmt.Errorf("a user with email %s already exists", email)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("database: %w", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("bcrypt: %w", err)
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
		return fmt.Errorf("create user: %w", err)
	}
	fmt.Printf("\nSuperuser created successfully.\n  id: %s\n  email: %s\n", u.ID, u.Email)
	return nil
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
			fmt.Println("Error: read password:", err)
			return "", false
		}
		pw2, err := readPassword("Password (again): ")
		if err != nil {
			fmt.Println("Error: read password:", err)
			return "", false
		}
		if pw1 != pw2 {
			fmt.Println("Error: Passwords do not match. Try again.")
			continue
		}
		if len(pw1) < MinPasswordLength {
			fmt.Printf("Error: Password must be at least %d characters. Try again.\n\n", MinPasswordLength)
			continue
		}
		if issues := PasswordIssues(pw1); len(issues) > 0 {
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
