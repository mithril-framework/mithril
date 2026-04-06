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

	"mithril-rev/database/models"
	"mithril-rev/database/repositories"
	csu "mithril-rev/internal/createsuperuser"
	"mithril-rev/internal/db"

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

	fmt.Println("Superuser creation (mithril-rev)")
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
			fmt.Println("Error: Passwords do not match. Try again.\n")
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
				fmt.Println("Choose a new password.\n")
				continue
			}
		}
		return pw1, true
	}
}

func requireACLSchema(ctx context.Context, pool *pgxpool.Pool) error {
	var ok bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = 'users'
			  AND column_name = 'is_superuser'
		)
	`).Scan(&ok)
	if err != nil {
		return fmt.Errorf("database schema check: %w", err)
	}
	if !ok {
		return fmt.Errorf("database is missing ACL schema (e.g. users.is_superuser); run: make migrate-up")
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
