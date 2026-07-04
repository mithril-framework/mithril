package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/mithril-framework/mithril/database/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ACLRepository manages permissions, roles, and assignments.
type ACLRepository struct {
	db *pgxpool.Pool
}

// NewACLRepository returns a new ACLRepository.
func NewACLRepository(db *pgxpool.Pool) *ACLRepository {
	return &ACLRepository{db: db}
}

// UserIsSuperuser returns the is_superuser flag for a user.
func (r *ACLRepository) UserIsSuperuser(ctx context.Context, userID uuid.UUID) (bool, error) {
	var sup bool
	err := r.db.QueryRow(ctx, `SELECT is_superuser FROM users WHERE id = $1`, userID).Scan(&sup)
	if err != nil {
		return false, err
	}
	return sup, nil
}

// UserRoleNames returns role names for a user, sorted.
func (r *ACLRepository) UserRoleNames(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT r.name FROM roles r
		INNER JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1
		ORDER BY r.name
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	return names, rows.Err()
}

// UserEffectivePermissionCodenames returns distinct permission codenames for a user (roles + direct).
func (r *ACLRepository) UserEffectivePermissionCodenames(ctx context.Context, userID uuid.UUID) (map[string]struct{}, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT p.codename FROM permissions p
		WHERE p.id IN (
			SELECT permission_id FROM user_permissions WHERE user_id = $1
			UNION
			SELECT rp.permission_id FROM role_permissions rp
			INNER JOIN user_roles ur ON ur.role_id = rp.role_id WHERE ur.user_id = $1
		)
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]struct{})
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out[c] = struct{}{}
	}
	return out, rows.Err()
}

// SetSuperuserByEmail sets is_superuser for a user identified by email.
func (r *ACLRepository) SetSuperuserByEmail(ctx context.Context, email string, super bool) error {
	tag, err := r.db.Exec(ctx, `UPDATE users SET is_superuser = $2, updated_at = NOW() WHERE LOWER(email) = LOWER($1)`, email, super)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user not found: %s", email)
	}
	return nil
}

// CreatePermission inserts a permission by codename.
func (r *ACLRepository) CreatePermission(ctx context.Context, codename, description string) (*models.Permission, error) {
	if codename == "" {
		return nil, errors.New("codename required")
	}
	m := &models.Permission{ID: uuid.New(), Codename: codename, Description: description}
	err := r.db.QueryRow(ctx, `
		INSERT INTO permissions (id, codename, description) VALUES ($1, $2, $3)
		RETURNING created_at
	`, m.ID, m.Codename, m.Description).Scan(&m.CreatedAt)
	return m, err
}

// DeletePermissionByCodename removes a permission by codename.
func (r *ACLRepository) DeletePermissionByCodename(ctx context.Context, codename string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM permissions WHERE codename = $1`, codename)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("permission not found: %s", codename)
	}
	return nil
}

// ListPermissions returns all permissions.
func (r *ACLRepository) ListPermissions(ctx context.Context) ([]models.Permission, error) {
	rows, err := r.db.Query(ctx, `SELECT id, codename, description, created_at FROM permissions ORDER BY codename`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.Permission
	for rows.Next() {
		var m models.Permission
		if err := rows.Scan(&m.ID, &m.Codename, &m.Description, &m.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// GetPermissionIDByCodename returns permission id or ErrNoRows.
func (r *ACLRepository) GetPermissionIDByCodename(ctx context.Context, codename string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `SELECT id FROM permissions WHERE codename = $1`, codename).Scan(&id)
	return id, err
}

// CreateRole inserts a role by name.
func (r *ACLRepository) CreateRole(ctx context.Context, name, description string) (*models.Role, error) {
	if name == "" {
		return nil, errors.New("name required")
	}
	m := &models.Role{ID: uuid.New(), Name: name, Description: description}
	err := r.db.QueryRow(ctx, `
		INSERT INTO roles (id, name, description) VALUES ($1, $2, $3)
		RETURNING created_at
	`, m.ID, m.Name, m.Description).Scan(&m.CreatedAt)
	return m, err
}

// DeleteRoleByName removes a role by name.
func (r *ACLRepository) DeleteRoleByName(ctx context.Context, name string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM roles WHERE name = $1`, name)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("role not found: %s", name)
	}
	return nil
}

// ListRoles returns all roles.
func (r *ACLRepository) ListRoles(ctx context.Context) ([]models.Role, error) {
	rows, err := r.db.Query(ctx, `SELECT id, name, description, created_at FROM roles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.Role
	for rows.Next() {
		var m models.Role
		if err := rows.Scan(&m.ID, &m.Name, &m.Description, &m.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// GetRoleIDByName returns role id or ErrNoRows.
func (r *ACLRepository) GetRoleIDByName(ctx context.Context, name string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `SELECT id FROM roles WHERE name = $1`, name).Scan(&id)
	return id, err
}

// GetUserIDByEmail returns user id or ErrNoRows.
func (r *ACLRepository) GetUserIDByEmail(ctx context.Context, email string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `SELECT id FROM users WHERE LOWER(email) = LOWER($1)`, email).Scan(&id)
	return id, err
}

// AssignRole links a user to a role.
func (r *ACLRepository) AssignRole(ctx context.Context, userID, roleID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)
		ON CONFLICT (user_id, role_id) DO NOTHING
	`, userID, roleID)
	return err
}

// RevokeRole removes user-role link.
func (r *ACLRepository) RevokeRole(ctx context.Context, userID, roleID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`, userID, roleID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("assignment not found")
	}
	return nil
}

// AssignPermissionToRole links permission to role.
func (r *ACLRepository) AssignPermissionToRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2)
		ON CONFLICT (role_id, permission_id) DO NOTHING
	`, roleID, permissionID)
	return err
}

// RevokePermissionFromRole removes link.
func (r *ACLRepository) RevokePermissionFromRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM role_permissions WHERE role_id = $1 AND permission_id = $2`, roleID, permissionID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("assignment not found")
	}
	return nil
}

// AssignPermissionToUser links permission directly to user.
func (r *ACLRepository) AssignPermissionToUser(ctx context.Context, userID, permissionID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO user_permissions (user_id, permission_id) VALUES ($1, $2)
		ON CONFLICT (user_id, permission_id) DO NOTHING
	`, userID, permissionID)
	return err
}

// RevokePermissionFromUser removes direct user permission.
func (r *ACLRepository) RevokePermissionFromUser(ctx context.Context, userID, permissionID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM user_permissions WHERE user_id = $1 AND permission_id = $2`, userID, permissionID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("assignment not found")
	}
	return nil
}

// ListRolePermissionCodenames returns permission codenames for a role.
func (r *ACLRepository) ListRolePermissionCodenames(ctx context.Context, roleID uuid.UUID) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.codename FROM permissions p
		INNER JOIN role_permissions rp ON rp.permission_id = p.id
		WHERE rp.role_id = $1 ORDER BY p.codename
	`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListUserRoleNames returns role names for a user (for admin UI).
func (r *ACLRepository) ListUserRoleNames(ctx context.Context, userID uuid.UUID) ([]string, error) {
	return r.UserRoleNames(ctx, userID)
}

// ListUserDirectPermissionCodenames returns codenames granted directly to user.
func (r *ACLRepository) ListUserDirectPermissionCodenames(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.codename FROM permissions p
		INNER JOIN user_permissions up ON up.permission_id = p.id
		WHERE up.user_id = $1 ORDER BY p.codename
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// UserHasRoleName checks if user has the named role.
func (r *ACLRepository) UserHasRoleName(ctx context.Context, userID uuid.UUID, roleName string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM user_roles ur
			INNER JOIN roles r ON r.id = ur.role_id
			WHERE ur.user_id = $1 AND r.name = $2
		)
	`, userID, roleName).Scan(&ok)
	if err != nil {
		return false, err
	}
	return ok, nil
}
