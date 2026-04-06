package repositories

import (
	"context"

	"mithril-rev/database/models"

	"github.com/google/uuid"
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

// Create inserts a user and sets ID (if not set), CreatedAt, UpdatedAt.
// ID is generated in the app so new rows never have NULL id regardless of DB default.
func (r *UserRepository) Create(ctx context.Context, u *models.User) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	query := `
		INSERT INTO users (id, email, password_hash, first_name, last_name, is_active, is_superuser)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at
	`
	return r.db.QueryRow(ctx, query, u.ID, u.Email, u.PasswordHash, u.FirstName, u.LastName, u.IsActive, u.IsSuperuser).
		Scan(&u.CreatedAt, &u.UpdatedAt)
}

// GetByID returns a user by id.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, is_active, is_superuser, created_at, updated_at
		FROM users WHERE id = $1
	`
	var u models.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName,
		&u.IsActive, &u.IsSuperuser, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetByEmail returns a user by email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, is_active, is_superuser, created_at, updated_at
		FROM users WHERE email = $1
	`
	var u models.User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName,
		&u.IsActive, &u.IsSuperuser, &u.CreatedAt, &u.UpdatedAt,
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
		SET email = $1, password_hash = $2, first_name = $3, last_name = $4, is_active = $5, is_superuser = $6, updated_at = NOW()
		WHERE id = $7
	`
	_, err := r.db.Exec(ctx, query, u.Email, u.PasswordHash, u.FirstName, u.LastName, u.IsActive, u.IsSuperuser, u.ID)
	return err
}

// Delete deletes a user by id.
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

// List returns users with limit and offset.
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*models.User, error) {
	query := `SELECT id, email, password_hash, first_name, last_name, is_active, is_superuser, created_at, updated_at FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &u.IsActive, &u.IsSuperuser, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, &u)
	}
	return list, rows.Err()
}
