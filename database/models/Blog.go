package models

import (
	"time"

	"github.com/google/uuid"
)

// Blog represents a blog row (pgx scanning).
type Blog struct {
	ID        uuid.UUID
	Title     string
	Content   string
	AuthorID  uuid.UUID
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
