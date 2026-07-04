package blog

import (
	"net/http"

	"github.com/mithril-framework/mithril/database/models"
	"github.com/mithril-framework/mithril/database/repositories"
	"github.com/mithril-framework/mithril/internal/acl"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Handlers holds blog CRUD handlers.
type Handlers struct {
	repo *repositories.BlogRepository
	acl  *acl.Service
}

// NewHandlers returns Blog CRUD handlers. acl may be nil (no permission checks).
func NewHandlers(repo *repositories.BlogRepository, aclSvc *acl.Service) *Handlers {
	return &Handlers{repo: repo, acl: aclSvc}
}

// List returns paginated blogs. Users with blogs.view see all; others see only their posts.
func (h *Handlers) List(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	uid, err := acl.CurrentUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	ctx := c.Context()
	var list []*models.Blog
	if h.acl != nil {
		viewAll, err := h.acl.HasPermission(ctx, c, uid, "blogs.view", acl.IsSuperuserLocal(c))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		if viewAll {
			list, err = h.repo.List(ctx, limit, offset)
		} else {
			list, err = h.repo.ListByAuthor(ctx, uid, limit, offset)
		}
	} else {
		list, err = h.repo.List(ctx, limit, offset)
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(list)
}

// Get returns one blog by id if the caller may read it.
func (h *Handlers) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	m, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	uid, err := acl.CurrentUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	if h.acl != nil {
		ok, err := h.acl.CanAccessOwnedResource(c.Context(), c, uid, m.AuthorID, "blogs.view", acl.IsSuperuserLocal(c))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		}
	}
	return c.JSON(m)
}

// Create creates a blog. Requires blogs.add; author_id is set to the current user.
func (h *Handlers) Create(c *fiber.Ctx) error {
	uid, err := acl.CurrentUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	if h.acl != nil {
		ok, err := h.acl.HasPermission(c.Context(), c, uid, "blogs.add", acl.IsSuperuserLocal(c))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden", "message": "blogs.add required"})
		}
	}
	var m models.Blog
	if err := c.BodyParser(&m); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	m.AuthorID = uid
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
	existing, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	uid, err := acl.CurrentUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	ctx := c.Context()
	if h.acl != nil {
		jwtSu := acl.IsSuperuserLocal(c)
		sup, err := h.acl.Repo().UserIsSuperuser(ctx, uid)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		isElevated := jwtSu || sup
		changeAny, err := h.acl.HasPermission(ctx, c, uid, "blogs.change_any", jwtSu)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		if uid != existing.AuthorID && !changeAny && !isElevated {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		}
		if uid == existing.AuthorID && !isElevated && !changeAny {
			ok, err := h.acl.HasPermission(ctx, c, uid, "blogs.change", jwtSu)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
			}
			if !ok {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden", "message": "blogs.change required"})
			}
		}
	}
	var m models.Blog
	if err := c.BodyParser(&m); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	m.ID = id
	if h.acl != nil {
		changeAny, _ := h.acl.HasPermission(ctx, c, uid, "blogs.change_any", acl.IsSuperuserLocal(c))
		sup, _ := h.acl.Repo().UserIsSuperuser(ctx, uid)
		if !changeAny && !sup && !acl.IsSuperuserLocal(c) {
			m.AuthorID = existing.AuthorID
		}
	}
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
	existing, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	uid, err := acl.CurrentUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	ctx := c.Context()
	if h.acl != nil {
		jwtSu := acl.IsSuperuserLocal(c)
		sup, err := h.acl.Repo().UserIsSuperuser(ctx, uid)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		isElevated := jwtSu || sup
		delAny, err := h.acl.HasPermission(ctx, c, uid, "blogs.delete_any", jwtSu)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		if uid != existing.AuthorID && !delAny && !isElevated {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		}
		if uid == existing.AuthorID && !isElevated && !delAny {
			ok, err := h.acl.HasPermission(ctx, c, uid, "blogs.delete", jwtSu)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
			}
			if !ok {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden", "message": "blogs.delete required"})
			}
		}
	}
	if err := h.repo.Delete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
