package main

import (
	"context"
	"errors"
	"log"
	"os"

	"mithril-rev/database/models"
	"mithril-rev/database/repositories"
	"mithril-rev/internal/db"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	if os.Getenv("ALLOW_SEED") != "1" {
		log.Fatal("Seed is manual-only. Run: ALLOW_SEED=1 make seed")
	}
	ctx := context.Background()
	dsn := db.DSNFromEnv()
	if os.Getenv("DATABASE_URL") == "" && os.Getenv("DB_HOST") == "" {
		log.Fatal("Set DATABASE_URL or DB_* env vars")
	}
	pool, err := db.New(ctx, dsn)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	repo := repositories.NewUserRepository(pool)
	_, err = repo.GetByEmail(ctx, "user@example.com")
	if err == nil {
		log.Println("Demo user user@example.com already exists, skip seed")
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		log.Fatalf("get by email: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("bcrypt: %v", err)
	}
	u := &models.User{
		Email:        "user@example.com",
		PasswordHash: string(hash),
		FirstName:    "Demo",
		LastName:     "User",
		IsActive:     true,
	}
	if err := repo.Create(ctx, u); err != nil {
		log.Fatalf("seed create user: %v", err)
	}
	log.Printf("Seeded demo user id=%s email=user@example.com", u.ID)
}
