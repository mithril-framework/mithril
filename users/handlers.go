package users

import (
	"errors"
	"log"
	"net/http"

	"github.com/mithril-framework/mithril/acl"
	"github.com/mithril-framework/mithril/database/models"
	"github.com/mithril-framework/mithril/database/repositories"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
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
func (h *Handlers) List(c fiber.Ctx) error {
	limit := fiber.Query[int](c, "limit", 20)
	offset := fiber.Query[int](c, "offset", 0)
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	list, err := h.repo.List(c, limit, offset)
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
func (h *Handlers) Get(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	m, err := h.repo.GetByID(c, id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	if h.acl != nil {
		self, err := acl.CurrentUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		if self != id {
			ok, err := h.acl.HasPermission(c, c, self, "users.view", acl.IsSuperuserLocal(c))
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

// createUserRequest is the only shape accepted by POST /api/users. Binding into models.User
// directly would let callers set is_superuser, id or a raw password_hash (mass assignment).
type createUserRequest struct {
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=8,max=72"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	IsActive    *bool  `json:"is_active"`
	IsSuperuser *bool  `json:"is_superuser"`
}

// Create creates a user (requires users.add via middleware). Only a superuser may create a superuser.
func (h *Handlers) Create(c fiber.Ctx) error {
	var req createUserRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	actor, err := h.actorFor(c, uuid.Nil)
	if err != nil {
		return respondInternal(c, "resolve actor", err)
	}
	u, perr := buildNewUser(&req, actor)
	if perr != nil {
		return c.Status(perr.status).JSON(fiber.Map{"error": perr.msg})
	}
	if err := h.repo.Create(c, u); err != nil {
		if isUniqueViolation(err) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "email already exists"})
		}
		return respondInternal(c, "create user", err)
	}
	return c.Status(http.StatusCreated).JSON(userPublic(u))
}

// updateUserRequest has pointer fields so omitted fields are left unchanged (PATCH semantics);
// a PUT with only first_name must not wipe last_name or deactivate the account.
type updateUserRequest struct {
	Email           *string `json:"email"`
	FirstName       *string `json:"first_name"`
	LastName        *string `json:"last_name"`
	Password        *string `json:"password"`
	CurrentPassword string  `json:"current_password"`
	IsActive        *bool   `json:"is_active"`
	IsSuperuser     *bool   `json:"is_superuser"`
}

// Update updates a user by id. Self-service (profile fields; email/password need current_password)
// or users.change (non-superuser targets). Superuser status can only be changed by a superuser.
func (h *Handlers) Update(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	existing, err := h.repo.GetByID(c, id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	actor, err := h.actorFor(c, id)
	if err != nil {
		return respondInternal(c, "resolve actor", err)
	}
	var req updateUserRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	updated, perr := applyUserUpdate(&req, *existing, actor)
	if perr != nil {
		return c.Status(perr.status).JSON(fiber.Map{"error": perr.msg})
	}
	if err := h.repo.Update(c, &updated); err != nil {
		if isUniqueViolation(err) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "email already exists"})
		}
		return respondInternal(c, "update user", err)
	}
	fresh, err := h.repo.GetByID(c, id)
	if err != nil {
		return respondInternal(c, "reload user", err)
	}
	return c.JSON(userPublic(fresh))
}

// Delete deletes a user by id (requires users.delete via middleware). Superuser accounts can only be
// deleted by a superuser, and nobody can delete their own account through this endpoint.
func (h *Handlers) Delete(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	target, err := h.repo.GetByID(c, id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	actor, err := h.actorFor(c, id)
	if err != nil {
		return respondInternal(c, "resolve actor", err)
	}
	if perr := checkUserDelete(target, actor); perr != nil {
		return c.Status(perr.status).JSON(fiber.Map{"error": perr.msg})
	}
	if err := h.repo.Delete(c, id); err != nil {
		return respondInternal(c, "delete user", err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// actorFor resolves the caller's privileges from the database (not the JWT, which may be stale).
func (h *Handlers) actorFor(c fiber.Ctx, targetID uuid.UUID) (actor, error) {
	self, err := acl.CurrentUserID(c)
	if err != nil {
		return actor{}, err
	}
	a := actor{isSelf: targetID != uuid.Nil && self == targetID}
	if h.acl == nil {
		return a, nil
	}
	if a.isSuperuser, err = h.acl.IsSuperuserDB(c, c, self); err != nil {
		return actor{}, err
	}
	if a.canChange, err = h.acl.HasPermission(c, c, self, "users.change", a.isSuperuser); err != nil {
		return actor{}, err
	}
	return a, nil
}

func respondInternal(c fiber.Ctx, op string, err error) error {
	log.Printf("users: %s: %v", op, err)
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// fiber:context-methods migrated
