package repositories

import (
	"context"

	"mithril-rev/database/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository provides user persistence.
type UserRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository returns a new UserRepository.
func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a user and sets ID, CreatedAt, UpdatedAt.
func (r *UserRepository) Create(ctx context.Context, u *models.User) error {
	query := `
		INSERT INTO users (email, password_hash, first_name, last_name, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(ctx, query, u.Email, u.PasswordHash, u.FirstName, u.LastName, u.IsActive).
		Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

// GetByID returns a user by id.
func (r *UserRepository) GetByID(ctx context.Context, id int64) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, is_active, created_at, updated_at
		FROM users WHERE id = $1
	`
	var u models.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName,
		&u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetByEmail returns a user by email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, is_active, created_at, updated_at
		FROM users WHERE email = $1
	`
	var u models.User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName,
		&u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// Update updates a user by id.
func (r *UserRepository) Update(ctx context.Context, u *models.User) error {
	query := `
		UPDATE users
		SET email = $1, password_hash = $2, first_name = $3, last_name = $4, is_active = $5, updated_at = NOW()
		WHERE id = $6
	`
	_, err := r.db.Exec(ctx, query, u.Email, u.PasswordHash, u.FirstName, u.LastName, u.IsActive, u.ID)
	return err
}

// Delete deletes a user by id.
func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}
