// Package blog is the reference app's example resource: a blog post owned by its author.
package blog

import (
	"embed"
	"time"

	"github.com/google/uuid"
	"github.com/mithril-framework/mithril/resource"
)

// Migrations holds the blog table migration (version 00005 matches the pre-library history).
//
//go:embed migrations/*.sql
var Migrations embed.FS

// Blog is a blog post. Authors manage their own posts with blogs.change/blogs.delete;
// blogs.change_any/blogs.delete_any and blogs.view grant access to everyone's posts.
type Blog struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title" validate:"required,max=200" example:"Hello, Mithril"`
	Content   string    `json:"content" mithril:"text"`
	AuthorID  uuid.UUID `json:"author_id" mithril:"owner"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Resource is the declarative blogs resource: CRUD under /api/blogs, OpenAPI, permissions, admin.
func Resource() resource.Resource {
	return resource.Define[Blog](resource.Options{Name: "blogs", Label: "Blogs"})
}
