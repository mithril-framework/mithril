package acl

import (
	"errors"

	"github.com/mithril-framework/mithril/app/models"
	"gorm.io/gorm"
)

// RBAC manages role-based access control
type RBAC struct {
	db *gorm.DB
}

// NewRBAC creates a new RBAC instance
func NewRBAC(db *gorm.DB) *RBAC {
	return &RBAC{db: db}
}

// CreateRole creates a new role
func (r *RBAC) CreateRole(role *models.Role) error {
	return r.db.Create(role).Error
}

// GetRoleBySlug retrieves a role by its slug
func (r *RBAC) GetRoleBySlug(slug string) (*models.Role, error) {
	var role models.Role
	err := r.db.Preload("Permissions").Where("slug = ?", slug).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// GetRoleByID retrieves a role by its ID
func (r *RBAC) GetRoleByID(id string) (*models.Role, error) {
	var role models.Role
	err := r.db.Preload("Permissions").First(&role, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// UpdateRole updates a role
func (r *RBAC) UpdateRole(role *models.Role) error {
	if role.IsSystem {
		return errors.New("cannot update system role")
	}
	return r.db.Save(role).Error
}

// DeleteRole deletes a role
func (r *RBAC) DeleteRole(id string) error {
	var role models.Role
	if err := r.db.First(&role, "id = ?", id).Error; err != nil {
		return err
	}
	if role.IsSystem {
		return errors.New("cannot delete system role")
	}
	return r.db.Delete(&role).Error
}

// ListRoles lists all roles
func (r *RBAC) ListRoles() ([]models.Role, error) {
	var roles []models.Role
	err := r.db.Preload("Permissions").Find(&roles).Error
	return roles, err
}

// CreatePermission creates a new permission
func (r *RBAC) CreatePermission(permission *models.Permission) error {
	return r.db.Create(permission).Error
}

// GetPermissionBySlug retrieves a permission by its slug
func (r *RBAC) GetPermissionBySlug(slug string) (*models.Permission, error) {
	var permission models.Permission
	err := r.db.Where("slug = ?", slug).First(&permission).Error
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

// GetPermissionByID retrieves a permission by its ID
func (r *RBAC) GetPermissionByID(id string) (*models.Permission, error) {
	var permission models.Permission
	err := r.db.First(&permission, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

// UpdatePermission updates a permission
func (r *RBAC) UpdatePermission(permission *models.Permission) error {
	if permission.IsSystem {
		return errors.New("cannot update system permission")
	}
	return r.db.Save(permission).Error
}

// DeletePermission deletes a permission
func (r *RBAC) DeletePermission(id string) error {
	var permission models.Permission
	if err := r.db.First(&permission, "id = ?", id).Error; err != nil {
		return err
	}
	if permission.IsSystem {
		return errors.New("cannot delete system permission")
	}
	return r.db.Delete(&permission).Error
}

// ListPermissions lists all permissions
func (r *RBAC) ListPermissions() ([]models.Permission, error) {
	var permissions []models.Permission
	err := r.db.Find(&permissions).Error
	return permissions, err
}

// AssignRoleToUser assigns a role to a user
func (r *RBAC) AssignRoleToUser(userID, roleID string) error {
	var user models.User
	if err := r.db.First(&user, "id = ?", userID).Error; err != nil {
		return err
	}

	var role models.Role
	if err := r.db.First(&role, "id = ?", roleID).Error; err != nil {
		return err
	}

	return user.AssignRole(r.db, &role)
}

// RemoveRoleFromUser removes a role from a user
func (r *RBAC) RemoveRoleFromUser(userID, roleID string) error {
	var user models.User
	if err := r.db.First(&user, "id = ?", userID).Error; err != nil {
		return err
	}

	var role models.Role
	if err := r.db.First(&role, "id = ?", roleID).Error; err != nil {
		return err
	}

	return user.RemoveRole(r.db, &role)
}

// GivePermissionToUser gives a direct permission to a user
func (r *RBAC) GivePermissionToUser(userID, permissionID string) error {
	var user models.User
	if err := r.db.First(&user, "id = ?", userID).Error; err != nil {
		return err
	}

	var permission models.Permission
	if err := r.db.First(&permission, "id = ?", permissionID).Error; err != nil {
		return err
	}

	return user.GivePermission(r.db, &permission)
}

// RevokePermissionFromUser revokes a direct permission from a user
func (r *RBAC) RevokePermissionFromUser(userID, permissionID string) error {
	var user models.User
	if err := r.db.First(&user, "id = ?", userID).Error; err != nil {
		return err
	}

	var permission models.Permission
	if err := r.db.First(&permission, "id = ?", permissionID).Error; err != nil {
		return err
	}

	return user.RevokePermission(r.db, &permission)
}

// AssignPermissionToRole assigns a permission to a role
func (r *RBAC) AssignPermissionToRole(roleID, permissionID string) error {
	var role models.Role
	if err := r.db.First(&role, "id = ?", roleID).Error; err != nil {
		return err
	}

	var permission models.Permission
	if err := r.db.First(&permission, "id = ?", permissionID).Error; err != nil {
		return err
	}

	return role.AddPermission(r.db, &permission)
}

// RemovePermissionFromRole removes a permission from a role
func (r *RBAC) RemovePermissionFromRole(roleID, permissionID string) error {
	var role models.Role
	if err := r.db.First(&role, "id = ?", roleID).Error; err != nil {
		return err
	}

	var permission models.Permission
	if err := r.db.First(&permission, "id = ?", permissionID).Error; err != nil {
		return err
	}

	return role.RemovePermission(r.db, &permission)
}

// UserHasRole checks if a user has a specific role
func (r *RBAC) UserHasRole(userID, roleSlug string) (bool, error) {
	var user models.User
	if err := r.db.Preload("Roles").First(&user, "id = ?", userID).Error; err != nil {
		return false, err
	}
	return user.HasRole(roleSlug), nil
}

// UserHasPermission checks if a user has a specific permission
func (r *RBAC) UserHasPermission(userID, permissionSlug string) (bool, error) {
	var user models.User
	if err := r.db.Preload("Permissions").Preload("Roles.Permissions").First(&user, "id = ?", userID).Error; err != nil {
		return false, err
	}
	return user.HasPermission(permissionSlug), nil
}

// GetUserRoles returns all roles assigned to a user
func (r *RBAC) GetUserRoles(userID string) ([]models.Role, error) {
	var user models.User
	if err := r.db.Preload("Roles.Permissions").First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}

	roles := make([]models.Role, len(user.Roles))
	for i, role := range user.Roles {
		roles[i] = *role
	}
	return roles, nil
}

// GetUserPermissions returns all permissions (direct + from roles) for a user
func (r *RBAC) GetUserPermissions(userID string) ([]string, error) {
	var user models.User
	if err := r.db.Preload("Permissions").Preload("Roles.Permissions").First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	return user.GetAllPermissions(), nil
}

// SeedDefaultData creates default roles and permissions
func (r *RBAC) SeedDefaultData() error {
	// Create default permissions
	defaultPerms := DefaultPermissions()
	for _, perm := range defaultPerms {
		var exists models.Permission
		if err := r.db.Where("slug = ?", perm.Slug).First(&exists).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := r.db.Create(&perm).Error; err != nil {
					return err
				}
			}
		}
	}

	// Create default roles
	defaultRoles := DefaultRoles()
	for _, role := range defaultRoles {
		var exists models.Role
		if err := r.db.Where("slug = ?", role.Slug).First(&exists).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := r.db.Create(&role).Error; err != nil {
					return err
				}
			}
		}
	}

	// Assign all permissions to admin role
	var adminRole models.Role
	if err := r.db.Where("slug = ?", "admin").First(&adminRole).Error; err != nil {
		return err
	}

	var allPerms []models.Permission
	if err := r.db.Find(&allPerms).Error; err != nil {
		return err
	}

	permPtrs := make([]*models.Permission, len(allPerms))
	for i := range allPerms {
		permPtrs[i] = &allPerms[i]
	}

	return adminRole.SyncPermissions(r.db, permPtrs)
}

// DefaultRoles returns default system roles
func DefaultRoles() []models.Role {
	return []models.Role{
		{Name: "Admin", Slug: "admin", Description: "Administrator with full access", IsSystem: true},
		{Name: "Moderator", Slug: "moderator", Description: "Moderator with limited access", IsSystem: true},
		{Name: "User", Slug: "user", Description: "Regular user", IsSystem: true},
		{Name: "Guest", Slug: "guest", Description: "Guest user with minimal access", IsSystem: true},
	}
}

// DefaultPermissions returns default system permissions
func DefaultPermissions() []models.Permission {
	return []models.Permission{
		// User permissions
		{Name: "Create User", Slug: "user.create", Resource: "user", Action: "create", IsSystem: true},
		{Name: "Read User", Slug: "user.read", Resource: "user", Action: "read", IsSystem: true},
		{Name: "Update User", Slug: "user.update", Resource: "user", Action: "update", IsSystem: true},
		{Name: "Delete User", Slug: "user.delete", Resource: "user", Action: "delete", IsSystem: true},
		{Name: "List Users", Slug: "user.list", Resource: "user", Action: "list", IsSystem: true},

		// Role permissions
		{Name: "Create Role", Slug: "role.create", Resource: "role", Action: "create", IsSystem: true},
		{Name: "Read Role", Slug: "role.read", Resource: "role", Action: "read", IsSystem: true},
		{Name: "Update Role", Slug: "role.update", Resource: "role", Action: "update", IsSystem: true},
		{Name: "Delete Role", Slug: "role.delete", Resource: "role", Action: "delete", IsSystem: true},
		{Name: "List Roles", Slug: "role.list", Resource: "role", Action: "list", IsSystem: true},

		// Permission permissions
		{Name: "Create Permission", Slug: "permission.create", Resource: "permission", Action: "create", IsSystem: true},
		{Name: "Read Permission", Slug: "permission.read", Resource: "permission", Action: "read", IsSystem: true},
		{Name: "Update Permission", Slug: "permission.update", Resource: "permission", Action: "update", IsSystem: true},
		{Name: "Delete Permission", Slug: "permission.delete", Resource: "permission", Action: "delete", IsSystem: true},
		{Name: "List Permissions", Slug: "permission.list", Resource: "permission", Action: "list", IsSystem: true},
	}
}
