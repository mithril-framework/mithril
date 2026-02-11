package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Email           string         `gorm:"uniqueIndex;not null" json:"email" validate:"required,email"`
	Phone           string         `gorm:"uniqueIndex" json:"phone" validate:"omitempty,phone"`
	Password        string         `gorm:"not null" json:"-" validate:"required,password"`
	FirstName       string         `gorm:"not null" json:"first_name" validate:"required,min=2,max=50"`
	LastName        string         `gorm:"not null" json:"last_name" validate:"required,min=2,max=50"`
	IsActive        bool           `gorm:"default:true" json:"is_active"`
	IsEmailVerified bool           `gorm:"default:false" json:"is_email_verified"`
	IsPhoneVerified bool           `gorm:"default:false" json:"is_phone_verified"`
	Is2FAEnabled    bool           `gorm:"default:false" json:"is_2fa_enabled"`
	TwoFactorSecret string         `gorm:"-" json:"-"` // Not stored in DB for security
	LastLoginAt     *time.Time     `json:"last_login_at"`
	EmailVerifiedAt *time.Time     `json:"email_verified_at"`
	PhoneVerifiedAt *time.Time     `json:"phone_verified_at"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Roles       []*Role       `gorm:"many2many:user_roles;" json:"roles,omitempty"`
	Permissions []*Permission `gorm:"many2many:user_permissions;" json:"permissions,omitempty"`
	Sessions    []Session     `json:"sessions,omitempty"`
}

// BeforeCreate hook to set UUID if not already set
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return
}

// TableName returns the table name for User
func (User) TableName() string {
	return "users"
}

// GetFullName returns the user's full name
func (u *User) GetFullName() string {
	return u.FirstName + " " + u.LastName
}

// IsAdmin checks if user has admin role
func (u *User) IsAdmin() bool {
	for _, role := range u.Roles {
		if role.Slug == "admin" {
			return true
		}
	}
	return false
}

// HasRole checks if user has a specific role (by slug)
func (u *User) HasRole(roleSlug string) bool {
	for _, role := range u.Roles {
		if role.Slug == roleSlug {
			return true
		}
	}
	return false
}

// HasPermission checks if user has a specific permission (by slug)
func (u *User) HasPermission(permissionSlug string) bool {
	// Check direct permissions
	for _, permission := range u.Permissions {
		if permission.Slug == permissionSlug {
			return true
		}
	}

	// Check role permissions
	for _, role := range u.Roles {
		for _, permission := range role.Permissions {
			if permission.Slug == permissionSlug {
				return true
			}
		}
	}

	return false
}

// AssignRole assigns a role to the user
func (u *User) AssignRole(db *gorm.DB, role *Role) error {
	return db.Model(u).Association("Roles").Append(role)
}

// RemoveRole removes a role from the user
func (u *User) RemoveRole(db *gorm.DB, role *Role) error {
	return db.Model(u).Association("Roles").Delete(role)
}

// GivePermission gives a direct permission to the user
func (u *User) GivePermission(db *gorm.DB, permission *Permission) error {
	return db.Model(u).Association("Permissions").Append(permission)
}

// RevokePermission revokes a direct permission from the user
func (u *User) RevokePermission(db *gorm.DB, permission *Permission) error {
	return db.Model(u).Association("Permissions").Delete(permission)
}

// GetAllPermissions returns all permissions (direct + from roles)
func (u *User) GetAllPermissions() []string {
	permSet := make(map[string]bool)

	// Direct permissions
	for _, perm := range u.Permissions {
		permSet[perm.Slug] = true
	}

	// Role permissions
	for _, role := range u.Roles {
		for _, perm := range role.Permissions {
			permSet[perm.Slug] = true
		}
	}

	perms := make([]string, 0, len(permSet))
	for perm := range permSet {
		perms = append(perms, perm)
	}
	return perms
}

// Role represents a user role
type Role struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string         `gorm:"size:100;not null;unique;index" json:"name"`
	Slug        string         `gorm:"size:100;not null;unique;index" json:"slug"`
	Description string         `gorm:"type:text" json:"description"`
	IsSystem    bool           `gorm:"default:false" json:"is_system"` // System roles can't be deleted
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Permissions []*Permission `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
	Users       []*User       `gorm:"many2many:user_roles;" json:"users,omitempty"`
}

// TableName returns the table name for Role
func (Role) TableName() string {
	return "roles"
}

// BeforeCreate hook to set UUID if not already set
func (r *Role) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return
}

// HasPermission checks if the role has a specific permission
func (r *Role) HasPermission(permissionSlug string) bool {
	for _, permission := range r.Permissions {
		if permission.Slug == permissionSlug {
			return true
		}
	}
	return false
}

// AddPermission adds a permission to the role
func (r *Role) AddPermission(db *gorm.DB, permission *Permission) error {
	return db.Model(r).Association("Permissions").Append(permission)
}

// RemovePermission removes a permission from the role
func (r *Role) RemovePermission(db *gorm.DB, permission *Permission) error {
	return db.Model(r).Association("Permissions").Delete(permission)
}

// SyncPermissions replaces all permissions with the given ones
func (r *Role) SyncPermissions(db *gorm.DB, permissions []*Permission) error {
	return db.Model(r).Association("Permissions").Replace(permissions)
}

// Permission represents a system permission
type Permission struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string         `gorm:"size:100;not null" json:"name"`
	Slug        string         `gorm:"size:100;not null;unique;index" json:"slug"`
	Description string         `gorm:"type:text" json:"description"`
	Resource    string         `gorm:"size:100;index" json:"resource"` // e.g., "users", "posts"
	Action      string         `gorm:"size:50;index" json:"action"`    // e.g., "create", "read", "update", "delete"
	IsSystem    bool           `gorm:"default:false" json:"is_system"` // System permissions can't be deleted
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Roles []*Role `gorm:"many2many:role_permissions;" json:"roles,omitempty"`
}

// TableName returns the table name for Permission
func (Permission) TableName() string {
	return "permissions"
}

// BeforeCreate hook to set UUID if not already set
func (p *Permission) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return
}

// Session represents a user session
type Session struct {
	ID           string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID       string    `gorm:"not null;index" json:"user_id"`
	Token        string    `gorm:"uniqueIndex;not null" json:"token"`
	RefreshToken string    `gorm:"uniqueIndex" json:"refresh_token"`
	UserAgent    string    `json:"user_agent"`
	IPAddress    string    `json:"ip_address"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	ExpiresAt    time.Time `json:"expires_at"`
	LastUsedAt   time.Time `json:"last_used_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Relationships
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName returns the table name for Session
func (Session) TableName() string {
	return "sessions"
}

// IsExpired checks if the session is expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// PasswordReset represents a password reset token
type PasswordReset struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    string    `gorm:"not null;index" json:"user_id"`
	Token     string    `gorm:"uniqueIndex;not null" json:"token"`
	Email     string    `gorm:"not null" json:"email"`
	IsUsed    bool      `gorm:"default:false" json:"is_used"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`

	// Relationships
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName returns the table name for PasswordReset
func (PasswordReset) TableName() string {
	return "password_resets"
}

// IsExpired checks if the password reset token is expired
func (pr *PasswordReset) IsExpired() bool {
	return time.Now().After(pr.ExpiresAt)
}

// EmailVerification represents an email verification token
type EmailVerification struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    string    `gorm:"not null;index" json:"user_id"`
	Token     string    `gorm:"uniqueIndex;not null" json:"token"`
	Email     string    `gorm:"not null" json:"email"`
	IsUsed    bool      `gorm:"default:false" json:"is_used"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`

	// Relationships
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName returns the table name for EmailVerification
func (EmailVerification) TableName() string {
	return "email_verifications"
}

// IsExpired checks if the email verification token is expired
func (ev *EmailVerification) IsExpired() bool {
	return time.Now().After(ev.ExpiresAt)
}

// OTPVerification represents an OTP verification
type OTPVerification struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    string    `gorm:"not null;index" json:"user_id"`
	Phone     string    `gorm:"not null" json:"phone"`
	Code      string    `gorm:"not null" json:"code"`
	IsUsed    bool      `gorm:"default:false" json:"is_used"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`

	// Relationships
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName returns the table name for OTPVerification
func (OTPVerification) TableName() string {
	return "otp_verifications"
}

// IsExpired checks if the OTP is expired
func (otp *OTPVerification) IsExpired() bool {
	return time.Now().After(otp.ExpiresAt)
}
