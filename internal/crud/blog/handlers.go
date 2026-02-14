package blog

import (
	"net/http"

	"mithril-rev/database/models"
	"mithril-rev/database/repositories"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Handlers holds blog CRUD handlers.
type Handlers struct {
	repo *repositories.BlogRepository
}

// NewHandlers returns Blog CRUD handlers.
func NewHandlers(repo *repositories.BlogRepository) *Handlers {
	return &Handlers{repo: repo}
}

// List returns paginated blogs. Query: limit (default 20), offset (default 0).
func (h *Handlers) List(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	list, err := h.repo.List(c.Context(), limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(list)
}

// Get returns one blog by id.
func (h *Handlers) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	m, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	return c.JSON(m)
}

// Create creates a blog. Request body: JSON matching models.Blog.
func (h *Handlers) Create(c *fiber.Ctx) error {
	var m models.Blog
	if err := c.BodyParser(&m); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := h.repo.Create(c.Context(), &m); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(http.StatusCreated).JSON(m)
}

// Update updates a blog by id.
func (h *Handlers) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var m models.Blog
	if err := c.BodyParser(&m); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	m.ID = id
	if err := h.repo.Update(c.Context(), &m); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(m)
}

// Delete deletes a blog by id.
func (h *Handlers) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.repo.Delete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
