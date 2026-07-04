package acl

import (
	"context"
	"errors"

	"github.com/mithril-framework/mithril/database/models"
	"github.com/mithril-framework/mithril/database/repositories"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const localEffectivePermsKey = "acl_effective_perm_set"

// Service resolves permissions and ownership checks.
type Service struct {
	repo *repositories.ACLRepository
}

// NewService builds an ACL service.
func NewService(r *repositories.ACLRepository) *Service {
	if r == nil {
		return nil
	}
	return &Service{repo: r}
}

func (s *Service) cachedEffectivePerms(ctx context.Context, c *fiber.Ctx, userID uuid.UUID) (map[string]struct{}, error) {
	if c != nil {
		if v := c.Locals(localEffectivePermsKey); v != nil {
			if m, ok := v.(map[string]struct{}); ok {
				return m, nil
			}
		}
	}
	perms, err := s.repo.UserEffectivePermissionCodenames(ctx, userID)
	if err != nil {
		return nil, err
	}
	if c != nil {
		c.Locals(localEffectivePermsKey, perms)
	}
	return perms, nil
}

// HasPermission returns true if the user has the codename (roles + direct) or is superuser.
// jwtIsSuperuser comes from signed JWT claims; when true, DB is not queried for superuser.
func (s *Service) HasPermission(ctx context.Context, c *fiber.Ctx, userID uuid.UUID, codename string, jwtIsSuperuser bool) (bool, error) {
	if s == nil || s.repo == nil {
		return false, nil
	}
	if jwtIsSuperuser {
		return true, nil
	}
	sup, err := s.repo.UserIsSuperuser(ctx, userID)
	if err != nil {
		return false, err
	}
	if sup {
		return true, nil
	}
	perms, err := s.cachedEffectivePerms(ctx, c, userID)
	if err != nil {
		return false, err
	}
	_, ok := perms[codename]
	return ok, nil
}

// HasRole returns true if the user has the named role.
func (s *Service) HasRole(ctx context.Context, userID uuid.UUID, roleName string) (bool, error) {
	if s == nil || s.repo == nil {
		return false, nil
	}
	return s.repo.UserHasRoleName(ctx, userID, roleName)
}

// CanAccessOwnedResource is true for superuser, or holders of manageAnyCodename, or when currentUserID equals ownerUserID.
func (s *Service) CanAccessOwnedResource(ctx context.Context, c *fiber.Ctx, currentUserID, ownerUserID uuid.UUID, manageAnyCodename string, jwtIsSuperuser bool) (bool, error) {
	if s == nil || s.repo == nil {
		return currentUserID == ownerUserID, nil
	}
	if jwtIsSuperuser {
		return true, nil
	}
	sup, err := s.repo.UserIsSuperuser(ctx, currentUserID)
	if err != nil {
		return false, err
	}
	if sup {
		return true, nil
	}
	if manageAnyCodename != "" {
		ok, err := s.HasPermission(ctx, c, currentUserID, manageAnyCodename, false)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return currentUserID == ownerUserID, nil
}

// Repo exposes the underlying repository for admin CLI/API (nil if service nil).
func (s *Service) Repo() *repositories.ACLRepository {
	if s == nil {
		return nil
	}
	return s.repo
}

func aclErrNil() error {
	return errors.New("acl service not configured")
}

// SetSuperuserByEmail sets the superuser flag (programmatic / same as CLI).
func (s *Service) SetSuperuserByEmail(ctx context.Context, email string, super bool) error {
	if s == nil || s.repo == nil {
		return aclErrNil()
	}
	return s.repo.SetSuperuserByEmail(ctx, email, super)
}

// CreatePermission inserts a permission codename.
func (s *Service) CreatePermission(ctx context.Context, codename, description string) (*models.Permission, error) {
	if s == nil || s.repo == nil {
		return nil, aclErrNil()
	}
	return s.repo.CreatePermission(ctx, codename, description)
}

// DeletePermissionByCodename removes a permission.
func (s *Service) DeletePermissionByCodename(ctx context.Context, codename string) error {
	if s == nil || s.repo == nil {
		return aclErrNil()
	}
	return s.repo.DeletePermissionByCodename(ctx, codename)
}

// CreateRole inserts a role.
func (s *Service) CreateRole(ctx context.Context, name, description string) (*models.Role, error) {
	if s == nil || s.repo == nil {
		return nil, aclErrNil()
	}
	return s.repo.CreateRole(ctx, name, description)
}

// DeleteRoleByName removes a role.
func (s *Service) DeleteRoleByName(ctx context.Context, name string) error {
	if s == nil || s.repo == nil {
		return aclErrNil()
	}
	return s.repo.DeleteRoleByName(ctx, name)
}

// AssignRole links a user (by id) to a role (by id).
func (s *Service) AssignRole(ctx context.Context, userID, roleID uuid.UUID) error {
	if s == nil || s.repo == nil {
		return aclErrNil()
	}
	return s.repo.AssignRole(ctx, userID, roleID)
}

// RevokeRole removes a user–role link.
func (s *Service) RevokeRole(ctx context.Context, userID, roleID uuid.UUID) error {
	if s == nil || s.repo == nil {
		return aclErrNil()
	}
	return s.repo.RevokeRole(ctx, userID, roleID)
}

// AssignPermissionToRole attaches a permission to a role.
func (s *Service) AssignPermissionToRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	if s == nil || s.repo == nil {
		return aclErrNil()
	}
	return s.repo.AssignPermissionToRole(ctx, roleID, permissionID)
}

// RevokePermissionFromRole removes a permission from a role.
func (s *Service) RevokePermissionFromRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	if s == nil || s.repo == nil {
		return aclErrNil()
	}
	return s.repo.RevokePermissionFromRole(ctx, roleID, permissionID)
}

// AssignPermissionToUser grants a direct user permission.
func (s *Service) AssignPermissionToUser(ctx context.Context, userID, permissionID uuid.UUID) error {
	if s == nil || s.repo == nil {
		return aclErrNil()
	}
	return s.repo.AssignPermissionToUser(ctx, userID, permissionID)
}

// RevokePermissionFromUser removes a direct user permission.
func (s *Service) RevokePermissionFromUser(ctx context.Context, userID, permissionID uuid.UUID) error {
	if s == nil || s.repo == nil {
		return aclErrNil()
	}
	return s.repo.RevokePermissionFromUser(ctx, userID, permissionID)
}
