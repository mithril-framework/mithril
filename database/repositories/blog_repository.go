package repositories

import (
	"context"

	"mithril-rev/database/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BlogRepository provides blog persistence.
type BlogRepository struct {
	db *pgxpool.Pool
}

// NewBlogRepository returns a new BlogRepository.
func NewBlogRepository(db *pgxpool.Pool) *BlogRepository {
	return &BlogRepository{db: db}
}

// Create inserts a blog.
func (r *BlogRepository) Create(ctx context.Context, m *models.Blog) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	query := `
		INSERT INTO blogs (id, title, content, author_id, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at
	`
	return r.db.QueryRow(ctx, query, m.ID, m.Title, m.Content, m.AuthorID, m.IsActive).Scan(&m.CreatedAt, &m.UpdatedAt)
}

// GetByID returns a blog by id.
func (r *BlogRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Blog, error) {
	query := `SELECT id, title, content, author_id, is_active, created_at, updated_at FROM blogs WHERE id = $1`
	var m models.Blog
	err := r.db.QueryRow(ctx, query, id).Scan(&m.ID, &m.Title, &m.Content, &m.AuthorID, &m.IsActive, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// Update updates a blog by id.
func (r *BlogRepository) Update(ctx context.Context, m *models.Blog) error {
	query := `UPDATE blogs SET title = $1, content = $2, author_id = $3, is_active = $4, updated_at = NOW() WHERE id = $5`
	_, err := r.db.Exec(ctx, query, m.Title, m.Content, m.AuthorID, m.IsActive, m.ID)
	return err
}

// Delete deletes a blog by id.
func (r *BlogRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM blogs WHERE id = $1`, id)
	return err
}

// List returns blogs with limit and offset.
func (r *BlogRepository) List(ctx context.Context, limit, offset int) ([]*models.Blog, error) {
	query := `SELECT id, title, content, author_id, is_active, created_at, updated_at FROM blogs ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBlogRows(rows)
}

// ListByAuthor returns blogs for one author.
func (r *BlogRepository) ListByAuthor(ctx context.Context, authorID uuid.UUID, limit, offset int) ([]*models.Blog, error) {
	query := `SELECT id, title, content, author_id, is_active, created_at, updated_at FROM blogs WHERE author_id = $3 ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, query, limit, offset, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBlogRows(rows)
}

func scanBlogRows(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}) ([]*models.Blog, error) {
	var list []*models.Blog
	for rows.Next() {
		var m models.Blog
		if err := rows.Scan(&m.ID, &m.Title, &m.Content, &m.AuthorID, &m.IsActive, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, &m)
	}
	return list, rows.Err()
}
