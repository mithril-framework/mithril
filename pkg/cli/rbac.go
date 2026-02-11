package cli

import (
	"fmt"
	"strings"

	"github.com/mithril-framework/mithril/app/models"
	"github.com/mithril-framework/mithril/pkg/acl"
	"gorm.io/gorm"
)

// RBACCommands provides commands for RBAC management
type RBACCommands struct {
	db   *gorm.DB
	rbac *acl.RBAC
}

// NewRBACCommands creates a new RBACCommands instance
func NewRBACCommands(db *gorm.DB) *RBACCommands {
	return &RBACCommands{
		db:   db,
		rbac: acl.NewRBAC(db),
	}
}

// CreateRole creates a new role
func (r *RBACCommands) CreateRole(name, slug, description string) error {
	role := &models.Role{
		Name:        name,
		Slug:        slug,
		Description: description,
		IsSystem:    false,
	}

	if err := r.rbac.CreateRole(role); err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	fmt.Printf("✅ Role '%s' created successfully (ID: %s)\n", name, role.ID)
	return nil
}

// CreatePermission creates a new permission
func (r *RBACCommands) CreatePermission(name, slug, resource, action, description string) error {
	permission := &models.Permission{
		Name:        name,
		Slug:        slug,
		Resource:    resource,
		Action:      action,
		Description: description,
		IsSystem:    false,
	}

	if err := r.rbac.CreatePermission(permission); err != nil {
		return fmt.Errorf("failed to create permission: %w", err)
	}

	fmt.Printf("✅ Permission '%s' created successfully (ID: %s)\n", name, permission.ID)
	return nil
}

// AssignRole assigns a role to a user
func (r *RBACCommands) AssignRole(userEmail, roleSlug string) error {
	// Find user by email
	var user models.User
	if err := r.db.Where("email = ?", userEmail).First(&user).Error; err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Find role by slug
	role, err := r.rbac.GetRoleBySlug(roleSlug)
	if err != nil {
		return fmt.Errorf("role not found: %w", err)
	}

	// Assign role
	if err := r.rbac.AssignRoleToUser(user.ID.String(), role.ID.String()); err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	fmt.Printf("✅ Role '%s' assigned to user '%s'\n", role.Name, user.Email)
	return nil
}

// AssignPermission assigns a permission to a role
func (r *RBACCommands) AssignPermission(permissionSlug, roleSlug string) error {
	// Find permission by slug
	permission, err := r.rbac.GetPermissionBySlug(permissionSlug)
	if err != nil {
		return fmt.Errorf("permission not found: %w", err)
	}

	// Find role by slug
	role, err := r.rbac.GetRoleBySlug(roleSlug)
	if err != nil {
		return fmt.Errorf("role not found: %w", err)
	}

	// Assign permission to role
	if err := r.rbac.AssignPermissionToRole(role.ID.String(), permission.ID.String()); err != nil {
		return fmt.Errorf("failed to assign permission: %w", err)
	}

	fmt.Printf("✅ Permission '%s' assigned to role '%s'\n", permission.Name, role.Name)
	return nil
}

// ListRoles lists all roles
func (r *RBACCommands) ListRoles() error {
	roles, err := r.rbac.ListRoles()
	if err != nil {
		return fmt.Errorf("failed to list roles: %w", err)
	}

	if len(roles) == 0 {
		fmt.Println("No roles found")
		return nil
	}

	fmt.Println("\n📋 Roles:")
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("%-36s %-20s %-15s %s\n", "ID", "Name", "Slug", "Permissions")
	fmt.Println(strings.Repeat("-", 80))

	for _, role := range roles {
		permCount := len(role.Permissions)
		systemBadge := ""
		if role.IsSystem {
			systemBadge = " [System]"
		}
		fmt.Printf("%-36s %-20s %-15s %d permissions%s\n",
			role.ID.String(), role.Name, role.Slug, permCount, systemBadge)
	}
	fmt.Println()

	return nil
}

// ListPermissions lists all permissions
func (r *RBACCommands) ListPermissions() error {
	permissions, err := r.rbac.ListPermissions()
	if err != nil {
		return fmt.Errorf("failed to list permissions: %w", err)
	}

	if len(permissions) == 0 {
		fmt.Println("No permissions found")
		return nil
	}

	fmt.Println("\n📋 Permissions:")
	fmt.Println(strings.Repeat("-", 100))
	fmt.Printf("%-36s %-30s %-20s %-15s %s\n", "ID", "Name", "Slug", "Resource", "Action")
	fmt.Println(strings.Repeat("-", 100))

	for _, perm := range permissions {
		systemBadge := ""
		if perm.IsSystem {
			systemBadge = " [System]"
		}
		fmt.Printf("%-36s %-30s %-20s %-15s %s%s\n",
			perm.ID.String(), perm.Name, perm.Slug, perm.Resource, perm.Action, systemBadge)
	}
	fmt.Println()

	return nil
}

// ShowUserPermissions shows all permissions for a user
func (r *RBACCommands) ShowUserPermissions(userEmail string) error {
	// Find user by email
	var user models.User
	if err := r.db.Preload("Roles.Permissions").Preload("Permissions").Where("email = ?", userEmail).First(&user).Error; err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	fmt.Printf("\n👤 User: %s %s (%s)\n", user.FirstName, user.LastName, user.Email)

	// Show roles
	fmt.Println("\n📌 Roles:")
	if len(user.Roles) == 0 {
		fmt.Println("  No roles assigned")
	} else {
		for _, role := range user.Roles {
			fmt.Printf("  - %s (%s)\n", role.Name, role.Slug)
		}
	}

	// Show all permissions
	allPerms := user.GetAllPermissions()
	fmt.Println("\n🔐 Permissions:")
	if len(allPerms) == 0 {
		fmt.Println("  No permissions")
	} else {
		for _, perm := range allPerms {
			fmt.Printf("  - %s\n", perm)
		}
	}
	fmt.Println()

	return nil
}

// SeedDefaultRBACData seeds default roles and permissions
func (r *RBACCommands) SeedDefaultRBACData() error {
	fmt.Println("🌱 Seeding default roles and permissions...")

	if err := r.rbac.SeedDefaultData(); err != nil {
		return fmt.Errorf("failed to seed data: %w", err)
	}

	fmt.Println("✅ Default roles and permissions seeded successfully")
	return nil
}
