package models

import (
	"time"

	"github.com/google/uuid"
)

// Permission is a named capability (codename), similar to Django auth.Permission.
type Permission struct {
	ID          uuid.UUID
	Codename    string
	Description string
	CreatedAt   time.Time
}

// Role is a named group of permissions, similar to Django auth.Group.
type Role struct {
	ID          uuid.UUID
	Name        string
	Description string
	CreatedAt   time.Time
}
