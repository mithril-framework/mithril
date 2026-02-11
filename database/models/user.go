package models

import "time"

// User represents a user row (pgx scanning).
type User struct {
	ID           int64
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
