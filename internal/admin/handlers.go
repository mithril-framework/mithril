package admin

import (
	"errors"
	"net/http"
	"strings"

	"github.com/mithril-framework/mithril/database/models"
	"github.com/mithril-framework/mithril/database/repositories"
	"github.com/mithril-framework/mithril/internal/acl"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// Handlers serves /admin/api JSON (JWT + admin access already applied by route group).
type Handlers struct {
	acl        *repositories.ACLRepository
	users      *repositories.UserRepository
	blogs      *repositories.BlogRepository
	bcryptCost int
}

// NewHandlers builds admin API handlers.
func NewHandlers(acl *repositories.ACLRepository, users *repositories.UserRepository, blogs *repositories.BlogRepository) *Handlers {
	return &Handlers{acl: acl, users: users, blogs: blogs, bcryptCost: bcrypt.DefaultCost}
}

func userJSON(u *models.User) fiber.Map {
	return fiber.Map{
		"id": u.ID, "email": u.Email, "first_name": u.FirstName, "last_name": u.LastName,
		"is_active": u.IsActive, "is_superuser": u.IsSuperuser,
		"created_at": u.CreatedAt, "updated_at": u.UpdatedAt,
	}
}

func blogJSON(b *models.Blog) fiber.Map {
	return fiber.Map{
		"id": b.ID, "title": b.Title, "content": b.Content, "author_id": b.AuthorID,
		"is_active": b.IsActive, "created_at": b.CreatedAt, "updated_at": b.UpdatedAt,
	}
}

// Meta returns resource names for the UI.
func (h *Handlers) Meta(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"resources": ResourceNames})
}

// --- permissions ---

func (h *Handlers) ListPermissions(c fiber.Ctx) error {
	list, err := h.acl.ListPermissions(c)
	if err != nil {
		return respondDBErr(c, err)
	}
	out := make([]fiber.Map, 0, len(list))
	for _, p := range list {
		out = append(out, fiber.Map{"id": p.ID, "codename": p.Codename, "description": p.Description, "created_at": p.CreatedAt})
	}
	return c.JSON(out)
}

func (h *Handlers) CreatePermission(c fiber.Ctx) error {
	var body struct {
		Codename    string `json:"codename"`
		Description string `json:"description"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	p, err := h.acl.CreatePermission(c, strings.TrimSpace(body.Codename), body.Description)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(http.StatusCreated).JSON(fiber.Map{"id": p.ID, "codename": p.Codename, "description": p.Description})
}

func (h *Handlers) DeletePermission(c fiber.Ctx) error {
	codename := c.Params("codename")
	if err := h.acl.DeletePermissionByCodename(c, codename); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(http.StatusNoContent)
}

// --- roles ---

func (h *Handlers) ListRoles(c fiber.Ctx) error {
	list, err := h.acl.ListRoles(c)
	if err != nil {
		return respondDBErr(c, err)
	}
	out := make([]fiber.Map, 0, len(list))
	for _, r := range list {
		out = append(out, fiber.Map{"id": r.ID, "name": r.Name, "description": r.Description, "created_at": r.CreatedAt})
	}
	return c.JSON(out)
}

func (h *Handlers) CreateRole(c fiber.Ctx) error {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	r, err := h.acl.CreateRole(c, strings.TrimSpace(body.Name), body.Description)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(http.StatusCreated).JSON(fiber.Map{"id": r.ID, "name": r.Name, "description": r.Description})
}

func (h *Handlers) DeleteRole(c fiber.Ctx) error {
	name := c.Params("name")
	if err := h.acl.DeleteRoleByName(c, name); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(http.StatusNoContent)
}

// --- assignments ---

func (h *Handlers) AssignRole(c fiber.Ctx) error {
	var body struct {
		UserEmail string `json:"user_email"`
		RoleName  string `json:"role_name"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	uid, err := h.acl.GetUserIDByEmail(c, body.UserEmail)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "user not found"})
	}
	rid, err := h.acl.GetRoleIDByName(c, body.RoleName)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "role not found"})
	}
	if err := h.acl.AssignRole(c, uid, rid); err != nil {
		return respondDBErr(c, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *Handlers) RevokeRole(c fiber.Ctx) error {
	var body struct {
		UserEmail string `json:"user_email"`
		RoleName  string `json:"role_name"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	uid, err := h.acl.GetUserIDByEmail(c, body.UserEmail)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "user not found"})
	}
	rid, err := h.acl.GetRoleIDByName(c, body.RoleName)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "role not found"})
	}
	if err := h.acl.RevokeRole(c, uid, rid); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *Handlers) AssignPermissionRole(c fiber.Ctx) error {
	var body struct {
		RoleName string `json:"role_name"`
		Codename string `json:"codename"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	rid, err := h.acl.GetRoleIDByName(c, body.RoleName)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "role not found"})
	}
	pid, err := h.acl.GetPermissionIDByCodename(c, body.Codename)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "permission not found"})
	}
	if err := h.acl.AssignPermissionToRole(c, rid, pid); err != nil {
		return respondDBErr(c, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *Handlers) RevokePermissionRole(c fiber.Ctx) error {
	var body struct {
		RoleName string `json:"role_name"`
		Codename string `json:"codename"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	rid, err := h.acl.GetRoleIDByName(c, body.RoleName)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "role not found"})
	}
	pid, err := h.acl.GetPermissionIDByCodename(c, body.Codename)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "permission not found"})
	}
	if err := h.acl.RevokePermissionFromRole(c, rid, pid); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *Handlers) AssignPermissionUser(c fiber.Ctx) error {
	var body struct {
		UserEmail string `json:"user_email"`
		Codename  string `json:"codename"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	uid, err := h.acl.GetUserIDByEmail(c, body.UserEmail)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "user not found"})
	}
	pid, err := h.acl.GetPermissionIDByCodename(c, body.Codename)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "permission not found"})
	}
	if err := h.acl.AssignPermissionToUser(c, uid, pid); err != nil {
		return respondDBErr(c, err)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *Handlers) RevokePermissionUser(c fiber.Ctx) error {
	var body struct {
		UserEmail string `json:"user_email"`
		Codename  string `json:"codename"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	uid, err := h.acl.GetUserIDByEmail(c, body.UserEmail)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "user not found"})
	}
	pid, err := h.acl.GetPermissionIDByCodename(c, body.Codename)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "permission not found"})
	}
	if err := h.acl.RevokePermissionFromUser(c, uid, pid); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *Handlers) UserRoles(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	names, err := h.acl.ListUserRoleNames(c, id)
	if err != nil {
		return respondDBErr(c, err)
	}
	return c.JSON(fiber.Map{"roles": names})
}

func (h *Handlers) UserDirectPermissions(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	names, err := h.acl.ListUserDirectPermissionCodenames(c, id)
	if err != nil {
		return respondDBErr(c, err)
	}
	return c.JSON(fiber.Map{"permissions": names})
}

func (h *Handlers) RolePermissions(c fiber.Ctx) error {
	name := c.Params("name")
	rid, err := h.acl.GetRoleIDByName(c, name)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "role not found"})
	}
	list, err := h.acl.ListRolePermissionCodenames(c, rid)
	if err != nil {
		return respondDBErr(c, err)
	}
	return c.JSON(fiber.Map{"permissions": list})
}

// --- resource CRUD: users ---

func (h *Handlers) ListUsers(c fiber.Ctx) error {
	limit := fiber.Query[int](c, "limit", 50)
	offset := fiber.Query[int](c, "offset", 0)
	list, err := h.users.List(c, limit, offset)
	if err != nil {
		return respondDBErr(c, err)
	}
	out := make([]fiber.Map, 0, len(list))
	for _, u := range list {
		out = append(out, userJSON(u))
	}
	return c.JSON(out)
}

func (h *Handlers) GetUser(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	u, err := h.users.GetByID(c, id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}
	return c.JSON(userJSON(u))
}

func (h *Handlers) CreateUser(c fiber.Ctx) error {
	var body struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		FirstName   string `json:"first_name"`
		LastName    string `json:"last_name"`
		IsActive    *bool  `json:"is_active"`
		IsSuperuser *bool  `json:"is_superuser"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	if body.Email == "" || body.Password == "" {
		return c.Status(400).JSON(fiber.Map{"error": "email and password required"})
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), h.bcryptCost)
	if err != nil {
		return respondDBErr(c, err)
	}
	u := &models.User{
		Email: body.Email, PasswordHash: string(hash),
		FirstName: body.FirstName, LastName: body.LastName,
		IsActive: true, IsSuperuser: false,
	}
	if body.IsActive != nil {
		u.IsActive = *body.IsActive
	}
	if body.IsSuperuser != nil {
		u.IsSuperuser = *body.IsSuperuser
	}
	if err := h.users.Create(c, u); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(http.StatusCreated).JSON(userJSON(u))
}

func (h *Handlers) UpdateUser(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	existing, err := h.users.GetByID(c, id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not found"})
	}
	var body struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		FirstName   string `json:"first_name"`
		LastName    string `json:"last_name"`
		IsActive    *bool  `json:"is_active"`
		IsSuperuser *bool  `json:"is_superuser"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	u := *existing
	if body.Email != "" {
		u.Email = body.Email
	}
	if body.FirstName != "" {
		u.FirstName = body.FirstName
	}
	if body.LastName != "" {
		u.LastName = body.LastName
	}
	if body.IsActive != nil {
		u.IsActive = *body.IsActive
	}
	if body.IsSuperuser != nil {
		u.IsSuperuser = *body.IsSuperuser
	}
	if body.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), h.bcryptCost)
		if err != nil {
			return respondDBErr(c, err)
		}
		u.PasswordHash = string(hash)
	}
	if err := h.users.Update(c, &u); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	updated, _ := h.users.GetByID(c, id)
	return c.JSON(userJSON(updated))
}

func (h *Handlers) DeleteUser(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.users.Delete(c, id); err != nil {
		return respondDBErr(c, err)
	}
	return c.SendStatus(http.StatusNoContent)
}

// --- resource CRUD: blogs ---

func (h *Handlers) ListBlogs(c fiber.Ctx) error {
	limit := fiber.Query[int](c, "limit", 50)
	offset := fiber.Query[int](c, "offset", 0)
	list, err := h.blogs.List(c, limit, offset)
	if err != nil {
		return respondDBErr(c, err)
	}
	out := make([]fiber.Map, 0, len(list))
	for _, b := range list {
		out = append(out, blogJSON(b))
	}
	return c.JSON(out)
}

func (h *Handlers) GetBlog(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	b, err := h.blogs.GetByID(c, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(404).JSON(fiber.Map{"error": "not found"})
		}
		return respondDBErr(c, err)
	}
	return c.JSON(blogJSON(b))
}

func (h *Handlers) CreateBlog(c fiber.Ctx) error {
	var m models.Blog
	if err := c.Bind().Body(&m); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	uid, err := acl.CurrentUserID(c)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "missing authenticated user"})
	}
	m.AuthorID = uid
	if err := h.blogs.Create(c, &m); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(http.StatusCreated).JSON(blogJSON(&m))
}

func (h *Handlers) UpdateBlog(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	existing, err := h.blogs.GetByID(c, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(404).JSON(fiber.Map{"error": "not found"})
		}
		return respondDBErr(c, err)
	}
	var m models.Blog
	if err := c.Bind().Body(&m); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	m.ID = id
	m.AuthorID = existing.AuthorID
	if err := h.blogs.Update(c, &m); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	b, err := h.blogs.GetByID(c, id)
	if err != nil {
		return respondDBErr(c, err)
	}
	return c.JSON(blogJSON(b))
}

func (h *Handlers) DeleteBlog(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.blogs.Delete(c, id); err != nil {
		return respondDBErr(c, err)
	}
	return c.SendStatus(http.StatusNoContent)
}

// fiber:context-methods migrated
