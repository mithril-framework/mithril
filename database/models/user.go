package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user row (pgx scanning).
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
