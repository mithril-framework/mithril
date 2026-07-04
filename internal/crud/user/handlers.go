package user

import (
	"net/http"

	"github.com/mithril-framework/mithril/database/models"
	"github.com/mithril-framework/mithril/database/repositories"
	"github.com/mithril-framework/mithril/internal/acl"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Handlers holds user CRUD handlers.
type Handlers struct {
	repo *repositories.UserRepository
	acl  *acl.Service
}

// NewHandlers returns User CRUD handlers. acl may be nil.
func NewHandlers(repo *repositories.UserRepository, aclSvc *acl.Service) *Handlers {
	return &Handlers{repo: repo, acl: aclSvc}
}

func userPublic(u *models.User) fiber.Map {
	return fiber.Map{
		"id":           u.ID,
		"email":        u.Email,
		"first_name":   u.FirstName,
		"last_name":    u.LastName,
		"is_active":    u.IsActive,
		"is_superuser": u.IsSuperuser,
		"created_at":   u.CreatedAt,
		"updated_at":   u.UpdatedAt,
	}
}

// List returns paginated users (requires users.view via middleware).
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
	out := make([]fiber.Map, 0, len(list))
	for _, u := range list {
		out = append(out, userPublic(u))
	}
	return c.JSON(out)
}

// Get returns one user. Callers may read themselves without users.view.
func (h *Handlers) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	m, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	if h.acl != nil {
		self, err := acl.CurrentUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		if self != id {
			ok, err := h.acl.HasPermission(c.Context(), c, self, "users.view", acl.IsSuperuserLocal(c))
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
			}
			if !ok {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
			}
		}
	}
	return c.JSON(userPublic(m))
}

// Create creates a user (requires users.add via middleware).
func (h *Handlers) Create(c *fiber.Ctx) error {
	var m models.User
	if err := c.BodyParser(&m); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := h.repo.Create(c.Context(), &m); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(http.StatusCreated).JSON(userPublic(&m))
}

// Update updates a user by id. Self-service or users.change.
func (h *Handlers) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	existing, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	if h.acl != nil {
		self, err := acl.CurrentUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		if self != id {
			ok, err := h.acl.HasPermission(c.Context(), c, self, "users.change", acl.IsSuperuserLocal(c))
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
			}
			if !ok {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
			}
		}
	}
	var m models.User
	if err := c.BodyParser(&m); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	m.ID = id
	if m.PasswordHash == "" {
		m.PasswordHash = existing.PasswordHash
	}
	if m.Email == "" {
		m.Email = existing.Email
	}
	if h.acl != nil {
		self, _ := acl.CurrentUserID(c)
		canElevate, _ := h.acl.HasPermission(c.Context(), c, self, "users.change", acl.IsSuperuserLocal(c))
		sup, _ := h.acl.Repo().UserIsSuperuser(c.Context(), self)
		if !canElevate && !sup && !acl.IsSuperuserLocal(c) {
			m.IsSuperuser = existing.IsSuperuser
		}
	}
	if err := h.repo.Update(c.Context(), &m); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	updated, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(userPublic(updated))
}

// Delete deletes a user by id (requires users.delete via middleware).
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
