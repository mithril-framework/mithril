package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
)

// AuthCommands contains all authentication-related commands
type AuthCommands struct{}

// NewAuthCommands creates a new auth commands instance
func NewAuthCommands() *AuthCommands {
	return &AuthCommands{}
}

// Register registers all authentication commands
func (ac *AuthCommands) Register() {
	// make:auth command
	NewCommand("make:auth", "Scaffold authentication system").
		Description("Generate complete authentication system with routes, controllers, and middleware").
		Category("Authentication").
		BoolFlag("views", "Include authentication views").
		BoolFlag("api-only", "API-only authentication").
		BoolFlag("web-only", "Web-only authentication").
		Action(ac.MakeAuth).
		Register()

	// createsuperuser command
	NewCommand("createsuperuser", "Create a superuser account").
		Description("Create a superuser account with administrative privileges").
		Category("Authentication").
		StringFlag("email", "", "Superuser email").
		StringFlag("name", "", "Superuser name").
		StringFlag("password", "", "Superuser password").
		Action(ac.CreateSuperUser).
		Register()

	// make:role command
	NewCommand("make:role", "Create a new role").
		Description("Create a new role with specified permissions").
		Category("Authentication").
		StringFlag("permissions", "", "Comma-separated list of permissions").
		Action(ac.MakeRole).
		Register()

	// make:permission command
	NewCommand("make:permission", "Create a new permission").
		Description("Create a new permission").
		Category("Authentication").
		Action(ac.MakePermission).
		Register()

	// assign:role command
	NewCommand("assign:role", "Assign role to user").
		Description("Assign a role to a user").
		Category("Authentication").
		Action(ac.AssignRole).
		Register()

	// assign:permission command
	NewCommand("assign:permission", "Assign permission to role").
		Description("Assign a permission to a role").
		Category("Authentication").
		Action(ac.AssignPermission).
		Register()

	// revoke:role command
	NewCommand("revoke:role", "Revoke role from user").
		Description("Revoke a role from a user").
		Category("Authentication").
		Action(ac.RevokeRole).
		Register()

	// revoke:permission command
	NewCommand("revoke:permission", "Revoke permission from role").
		Description("Revoke a permission from a role").
		Category("Authentication").
		Action(ac.RevokePermission).
		Register()

	// list:roles command
	NewCommand("list:roles", "List all roles").
		Description("List all roles in the system").
		Category("Authentication").
		Action(ac.ListRoles).
		Register()

	// list:permissions command
	NewCommand("list:permissions", "List all permissions").
		Description("List all permissions in the system").
		Category("Authentication").
		Action(ac.ListPermissions).
		Register()

	// list:users command
	NewCommand("list:users", "List all users").
		Description("List all users in the system").
		Category("Authentication").
		Action(ac.ListUsers).
		Register()
}

// MakeAuth scaffolds authentication system
func (ac *AuthCommands) MakeAuth(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	includeViews := cc.GetBoolFlag("views")
	apiOnly := cc.GetBoolFlag("api-only")
	webOnly := cc.GetBoolFlag("web-only")

	cc.PrintInfo("Scaffolding authentication system...")

	// Determine authentication type
	authType := "full"
	if apiOnly {
		authType = "api-only"
	} else if webOnly {
		authType = "web-only"
	}

	cc.PrintInfo(fmt.Sprintf("Authentication type: %s", authType))

	// Create authentication directory structure
	authDir := "app/auth"
	directories := []string{
		authDir,
		filepath.Join(authDir, "controllers"),
		filepath.Join(authDir, "middleware"),
		filepath.Join(authDir, "services"),
		filepath.Join(authDir, "models"),
		filepath.Join(authDir, "schemas"),
	}

	if includeViews {
		directories = append(directories, filepath.Join(authDir, "views"))
	}

	for _, dir := range directories {
		if err := cc.CreateDirectory(dir); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create authentication files
	if err := ac.createAuthFiles(cc, authDir, authType, includeViews); err != nil {
		return err
	}

	// Create user model
	if err := ac.createUserModel(cc); err != nil {
		return err
	}

	// Create role and permission models
	if err := ac.createRBACModels(cc); err != nil {
		return err
	}

	// Create authentication routes
	if err := ac.createAuthRoutes(cc, authType); err != nil {
		return err
	}

	// Create authentication middleware
	if err := ac.createAuthMiddleware(cc); err != nil {
		return err
	}

	// Create authentication services
	if err := ac.createAuthServices(cc); err != nil {
		return err
	}

	// Create authentication schemas
	if err := ac.createAuthSchemas(cc); err != nil {
		return err
	}

	// Create views if requested
	if includeViews {
		if err := ac.createAuthViews(cc); err != nil {
			return err
		}
	}

	// Create migration for authentication tables
	if err := ac.createAuthMigration(cc); err != nil {
		return err
	}

	cc.PrintSuccess("Authentication system scaffolded successfully")
	cc.PrintInfo("Next steps:")
	cc.PrintInfo("1. Run migrations: go run . artisan migrate")
	cc.PrintInfo("2. Create superuser: go run . artisan createsuperuser")
	cc.PrintInfo("3. Start the application: go run . artisan serve")

	return nil
}

// CreateSuperUser creates a superuser account
func (ac *AuthCommands) CreateSuperUser(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	email := cc.GetStringFlag("email")
	name := cc.GetStringFlag("name")
	password := cc.GetStringFlag("password")

	cc.PrintInfo("Creating superuser account...")

	// Prompt for email if not provided
	if email == "" {
		fmt.Print("Email: ")
		_, _ = fmt.Scanln(&email)
	}

	// Prompt for name if not provided
	if name == "" {
		fmt.Print("Name: ")
		_, _ = fmt.Scanln(&name)
	}

	// Prompt for password if not provided
	if password == "" {
		fmt.Print("Password: ")
		_, _ = fmt.Scanln(&password)
	}

	// Validate inputs
	if email == "" {
		return fmt.Errorf("email is required")
	}
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if password == "" {
		return fmt.Errorf("password is required")
	}

	// Validate email format
	if !isValidEmail(email) {
		return fmt.Errorf("invalid email format")
	}

	// Validate password strength
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	// Create superuser
	if err := ac.createSuperUser(cc, email, name, password); err != nil {
		return fmt.Errorf("failed to create superuser: %w", err)
	}

	cc.PrintSuccess(fmt.Sprintf("Superuser created successfully: %s", email))
	return nil
}

// MakeRole creates a new role
func (ac *AuthCommands) MakeRole(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	if len(cc.Args) == 0 {
		return fmt.Errorf("role name is required")
	}

	roleName := cc.GetStringArg(0)
	permissions := cc.GetStringFlag("permissions")

	// Validate role name
	if !isValidRoleName(roleName) {
		return fmt.Errorf("invalid role name: %s", roleName)
	}

	cc.PrintInfo(fmt.Sprintf("Creating role: %s", roleName))

	// Create role
	if err := ac.createRole(cc, roleName, permissions); err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	cc.PrintSuccess(fmt.Sprintf("Role '%s' created successfully", roleName))
	return nil
}

// MakePermission creates a new permission
func (ac *AuthCommands) MakePermission(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	if len(cc.Args) == 0 {
		return fmt.Errorf("permission name is required")
	}

	permissionName := cc.GetStringArg(0)

	// Validate permission name
	if !isValidPermissionName(permissionName) {
		return fmt.Errorf("invalid permission name: %s", permissionName)
	}

	cc.PrintInfo(fmt.Sprintf("Creating permission: %s", permissionName))

	// Create permission
	if err := ac.createPermission(cc, permissionName); err != nil {
		return fmt.Errorf("failed to create permission: %w", err)
	}

	cc.PrintSuccess(fmt.Sprintf("Permission '%s' created successfully", permissionName))
	return nil
}

// AssignRole assigns a role to a user
func (ac *AuthCommands) AssignRole(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	if len(cc.Args) < 2 {
		return fmt.Errorf("role name and user identifier are required")
	}

	roleName := cc.GetStringArg(0)
	userIdentifier := cc.GetStringArg(1)

	cc.PrintInfo(fmt.Sprintf("Assigning role '%s' to user '%s'", roleName, userIdentifier))

	// Assign role
	if err := ac.assignRole(cc, roleName, userIdentifier); err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	cc.PrintSuccess(fmt.Sprintf("Role '%s' assigned to user '%s'", roleName, userIdentifier))
	return nil
}

// AssignPermission assigns a permission to a role
func (ac *AuthCommands) AssignPermission(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	if len(cc.Args) < 2 {
		return fmt.Errorf("permission name and role name are required")
	}

	permissionName := cc.GetStringArg(0)
	roleName := cc.GetStringArg(1)

	cc.PrintInfo(fmt.Sprintf("Assigning permission '%s' to role '%s'", permissionName, roleName))

	// Assign permission
	if err := ac.assignPermission(cc, permissionName, roleName); err != nil {
		return fmt.Errorf("failed to assign permission: %w", err)
	}

	cc.PrintSuccess(fmt.Sprintf("Permission '%s' assigned to role '%s'", permissionName, roleName))
	return nil
}

// RevokeRole revokes a role from a user
func (ac *AuthCommands) RevokeRole(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	if len(cc.Args) < 2 {
		return fmt.Errorf("role name and user identifier are required")
	}

	roleName := cc.GetStringArg(0)
	userIdentifier := cc.GetStringArg(1)

	cc.PrintInfo(fmt.Sprintf("Revoking role '%s' from user '%s'", roleName, userIdentifier))

	// Revoke role
	if err := ac.revokeRole(cc, roleName, userIdentifier); err != nil {
		return fmt.Errorf("failed to revoke role: %w", err)
	}

	cc.PrintSuccess(fmt.Sprintf("Role '%s' revoked from user '%s'", roleName, userIdentifier))
	return nil
}

// RevokePermission revokes a permission from a role
func (ac *AuthCommands) RevokePermission(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	if len(cc.Args) < 2 {
		return fmt.Errorf("permission name and role name are required")
	}

	permissionName := cc.GetStringArg(0)
	roleName := cc.GetStringArg(1)

	cc.PrintInfo(fmt.Sprintf("Revoking permission '%s' from role '%s'", permissionName, roleName))

	// Revoke permission
	if err := ac.revokePermission(cc, permissionName, roleName); err != nil {
		return fmt.Errorf("failed to revoke permission: %w", err)
	}

	cc.PrintSuccess(fmt.Sprintf("Permission '%s' revoked from role '%s'", permissionName, roleName))
	return nil
}

// ListRoles lists all roles
func (ac *AuthCommands) ListRoles(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	cc.PrintInfo("Listing all roles...")

	// Get roles
	roles, err := ac.getRoles(cc)
	if err != nil {
		return fmt.Errorf("failed to get roles: %w", err)
	}

	if len(roles) == 0 {
		cc.PrintInfo("No roles found")
		return nil
	}

	// Display roles
	headers := []string{"ID", "Name", "Description", "Created At"}
	var rows [][]string

	for _, role := range roles {
		rows = append(rows, []string{
			role.ID,
			role.Name,
			role.Description,
			role.CreatedAt,
		})
	}

	cc.PrintTable(headers, rows)
	return nil
}

// ListPermissions lists all permissions
func (ac *AuthCommands) ListPermissions(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	cc.PrintInfo("Listing all permissions...")

	// Get permissions
	permissions, err := ac.getPermissions(cc)
	if err != nil {
		return fmt.Errorf("failed to get permissions: %w", err)
	}

	if len(permissions) == 0 {
		cc.PrintInfo("No permissions found")
		return nil
	}

	// Display permissions
	headers := []string{"ID", "Name", "Description", "Created At"}
	var rows [][]string

	for _, permission := range permissions {
		rows = append(rows, []string{
			permission.ID,
			permission.Name,
			permission.Description,
			permission.CreatedAt,
		})
	}

	cc.PrintTable(headers, rows)
	return nil
}

// ListUsers lists all users
func (ac *AuthCommands) ListUsers(ctx *cli.Context) error {
	cc := NewCommandContext(ctx.App, ctx)

	cc.PrintInfo("Listing all users...")

	// Get users
	users, err := ac.getUsers(cc)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	if len(users) == 0 {
		cc.PrintInfo("No users found")
		return nil
	}

	// Display users
	headers := []string{"ID", "Email", "Name", "Roles", "Created At"}
	var rows [][]string

	for _, user := range users {
		rows = append(rows, []string{
			user.ID,
			user.Email,
			user.Name,
			strings.Join(user.Roles, ", "),
			user.CreatedAt,
		})
	}

	cc.PrintTable(headers, rows)
	return nil
}

// Helper functions for creating authentication files

func (ac *AuthCommands) createAuthFiles(cc *CommandContext, authDir, authType string, includeViews bool) error {
	// Create authentication controller
	controllerFile := filepath.Join(authDir, "controllers", "auth_controller.go")
	content := ac.generateAuthController(authType, includeViews)
	if err := cc.WriteFile(controllerFile, []byte(content)); err != nil {
		return err
	}

	// Create JWT service
	jwtFile := filepath.Join(authDir, "services", "jwt_service.go")
	content = ac.generateJWTService()
	if err := cc.WriteFile(jwtFile, []byte(content)); err != nil {
		return err
	}

	// Create password service
	passwordFile := filepath.Join(authDir, "services", "password_service.go")
	content = ac.generatePasswordService()
	if err := cc.WriteFile(passwordFile, []byte(content)); err != nil {
		return err
	}

	// Create OTP service
	otpFile := filepath.Join(authDir, "services", "otp_service.go")
	content = ac.generateOTPService()
	if err := cc.WriteFile(otpFile, []byte(content)); err != nil {
		return err
	}

	// Create 2FA service
	twoFactorFile := filepath.Join(authDir, "services", "two_factor_service.go")
	content = ac.generateTwoFactorService()
	if err := cc.WriteFile(twoFactorFile, []byte(content)); err != nil {
		return err
	}

	return nil
}

func (ac *AuthCommands) createUserModel(cc *CommandContext) error {
	userFile := filepath.Join("app", "models", "user.go")
	content := ac.generateUserModel()
	return cc.WriteFile(userFile, []byte(content))
}

func (ac *AuthCommands) createRBACModels(cc *CommandContext) error {
	// Create role model
	roleFile := filepath.Join("app", "models", "role.go")
	content := ac.generateRoleModel()
	if err := cc.WriteFile(roleFile, []byte(content)); err != nil {
		return err
	}

	// Create permission model
	permissionFile := filepath.Join("app", "models", "permission.go")
	content = ac.generatePermissionModel()
	if err := cc.WriteFile(permissionFile, []byte(content)); err != nil {
		return err
	}

	// Create user role model
	userRoleFile := filepath.Join("app", "models", "user_role.go")
	content = ac.generateUserRoleModel()
	if err := cc.WriteFile(userRoleFile, []byte(content)); err != nil {
		return err
	}

	// Create role permission model
	rolePermissionFile := filepath.Join("app", "models", "role_permission.go")
	content = ac.generateRolePermissionModel()
	if err := cc.WriteFile(rolePermissionFile, []byte(content)); err != nil {
		return err
	}

	return nil
}

func (ac *AuthCommands) createAuthRoutes(cc *CommandContext, authType string) error {
	routesFile := filepath.Join("routes", "auth.go")
	content := ac.generateAuthRoutes(authType)
	return cc.WriteFile(routesFile, []byte(content))
}

func (ac *AuthCommands) createAuthMiddleware(cc *CommandContext) error {
	// Create auth middleware
	authMiddlewareFile := filepath.Join("app", "middleware", "auth.go")
	content := ac.generateAuthMiddleware()
	if err := cc.WriteFile(authMiddlewareFile, []byte(content)); err != nil {
		return err
	}

	// Create RBAC middleware
	rbacMiddlewareFile := filepath.Join("app", "middleware", "rbac.go")
	content = ac.generateRBACMiddleware()
	if err := cc.WriteFile(rbacMiddlewareFile, []byte(content)); err != nil {
		return err
	}

	return nil
}

func (ac *AuthCommands) createAuthServices(cc *CommandContext) error {
	// Create auth service
	authServiceFile := filepath.Join("app", "auth", "services", "auth_service.go")
	content := ac.generateAuthService()
	return cc.WriteFile(authServiceFile, []byte(content))
}

func (ac *AuthCommands) createAuthSchemas(cc *CommandContext) error {
	// Create auth schemas
	schemasFile := filepath.Join("app", "auth", "schemas", "auth_schemas.go")
	content := ac.generateAuthSchemas()
	return cc.WriteFile(schemasFile, []byte(content))
}

func (ac *AuthCommands) createAuthViews(cc *CommandContext) error {
	// Create login view
	loginViewFile := filepath.Join("app", "auth", "views", "login.html")
	content := ac.generateLoginView()
	if err := cc.WriteFile(loginViewFile, []byte(content)); err != nil {
		return err
	}

	// Create register view
	registerViewFile := filepath.Join("app", "auth", "views", "register.html")
	content = ac.generateRegisterView()
	if err := cc.WriteFile(registerViewFile, []byte(content)); err != nil {
		return err
	}

	return nil
}

func (ac *AuthCommands) createAuthMigration(cc *CommandContext) error {
	// Create migration for authentication tables
	migrationFile := filepath.Join("database", "migrations", "create_auth_tables.go")
	content := ac.generateAuthMigration()
	return cc.WriteFile(migrationFile, []byte(content))
}

// Database operations

func (ac *AuthCommands) createSuperUser(cc *CommandContext, email, name, password string) error {
	// This would create a superuser in the database
	cc.PrintInfo(fmt.Sprintf("Creating superuser: %s", email))
	return nil
}

func (ac *AuthCommands) createRole(cc *CommandContext, name, permissions string) error {
	// This would create a role in the database
	cc.PrintInfo(fmt.Sprintf("Creating role: %s", name))
	return nil
}

func (ac *AuthCommands) createPermission(cc *CommandContext, name string) error {
	// This would create a permission in the database
	cc.PrintInfo(fmt.Sprintf("Creating permission: %s", name))
	return nil
}

func (ac *AuthCommands) assignRole(cc *CommandContext, roleName, userIdentifier string) error {
	// This would assign a role to a user
	cc.PrintInfo(fmt.Sprintf("Assigning role %s to user %s", roleName, userIdentifier))
	return nil
}

func (ac *AuthCommands) assignPermission(cc *CommandContext, permissionName, roleName string) error {
	// This would assign a permission to a role
	cc.PrintInfo(fmt.Sprintf("Assigning permission %s to role %s", permissionName, roleName))
	return nil
}

func (ac *AuthCommands) revokeRole(cc *CommandContext, roleName, userIdentifier string) error {
	// This would revoke a role from a user
	cc.PrintInfo(fmt.Sprintf("Revoking role %s from user %s", roleName, userIdentifier))
	return nil
}

func (ac *AuthCommands) revokePermission(cc *CommandContext, permissionName, roleName string) error {
	// This would revoke a permission from a role
	cc.PrintInfo(fmt.Sprintf("Revoking permission %s from role %s", permissionName, roleName))
	return nil
}

// Data structures for listing

type Role struct {
	ID          string
	Name        string
	Description string
	CreatedAt   string
}

type Permission struct {
	ID          string
	Name        string
	Description string
	CreatedAt   string
}

type User struct {
	ID        string
	Email     string
	Name      string
	Roles     []string
	CreatedAt string
}

func (ac *AuthCommands) getRoles(cc *CommandContext) ([]Role, error) {
	// This would query the database for roles
	return []Role{}, nil
}

func (ac *AuthCommands) getPermissions(cc *CommandContext) ([]Permission, error) {
	// This would query the database for permissions
	return []Permission{}, nil
}

func (ac *AuthCommands) getUsers(cc *CommandContext) ([]User, error) {
	// This would query the database for users
	return []User{}, nil
}

// Template generation functions

func (ac *AuthCommands) generateAuthController(authType string, includeViews bool) string {
	_ = authType      // authType parameter reserved for future use
	_ = includeViews  // includeViews parameter reserved for future use
	return `package controllers

import (
	"github.com/gofiber/fiber/v2"
)

type AuthController struct {
	// Add dependencies here
}

func NewAuthController() *AuthController {
	return &AuthController{}
}

// Register handles user registration
func (c *AuthController) Register(ctx *fiber.Ctx) error {
	// Implement registration logic
	return ctx.JSON(fiber.Map{"message": "Registration endpoint"})
}

// Login handles user login
func (c *AuthController) Login(ctx *fiber.Ctx) error {
	// Implement login logic
	return ctx.JSON(fiber.Map{"message": "Login endpoint"})
}

// Logout handles user logout
func (c *AuthController) Logout(ctx *fiber.Ctx) error {
	// Implement logout logic
	return ctx.JSON(fiber.Map{"message": "Logout endpoint"})
}

// Refresh handles token refresh
func (c *AuthController) Refresh(ctx *fiber.Ctx) error {
	// Implement token refresh logic
	return ctx.JSON(fiber.Map{"message": "Refresh endpoint"})
}

// ForgotPassword handles password reset request
func (c *AuthController) ForgotPassword(ctx *fiber.Ctx) error {
	// Implement forgot password logic
	return ctx.JSON(fiber.Map{"message": "Forgot password endpoint"})
}

// ResetPassword handles password reset
func (c *AuthController) ResetPassword(ctx *fiber.Ctx) error {
	// Implement reset password logic
	return ctx.JSON(fiber.Map{"message": "Reset password endpoint"})
}

// VerifyEmail handles email verification
func (c *AuthController) VerifyEmail(ctx *fiber.Ctx) error {
	// Implement email verification logic
	return ctx.JSON(fiber.Map{"message": "Verify email endpoint"})
}

// SendOTP handles OTP sending
func (c *AuthController) SendOTP(ctx *fiber.Ctx) error {
	// Implement OTP sending logic
	return ctx.JSON(fiber.Map{"message": "Send OTP endpoint"})
}

// VerifyOTP handles OTP verification
func (c *AuthController) VerifyOTP(ctx *fiber.Ctx) error {
	// Implement OTP verification logic
	return ctx.JSON(fiber.Map{"message": "Verify OTP endpoint"})
}

// Enable2FA handles 2FA enabling
func (c *AuthController) Enable2FA(ctx *fiber.Ctx) error {
	// Implement 2FA enabling logic
	return ctx.JSON(fiber.Map{"message": "Enable 2FA endpoint"})
}

// Verify2FA handles 2FA verification
func (c *AuthController) Verify2FA(ctx *fiber.Ctx) error {
	// Implement 2FA verification logic
	return ctx.JSON(fiber.Map{"message": "Verify 2FA endpoint"})
}

// Disable2FA handles 2FA disabling
func (c *AuthController) Disable2FA(ctx *fiber.Ctx) error {
	// Implement 2FA disabling logic
	return ctx.JSON(fiber.Map{"message": "Disable 2FA endpoint"})
}
`
}

func (ac *AuthCommands) generateJWTService() string {
	return `package services

import (
	"time"
	"github.com/golang-jwt/jwt/v5"
)

type JWTService struct {
	secretKey string
}

func NewJWTService(secretKey string) *JWTService {
	return &JWTService{
		secretKey: secretKey,
	}
}

// GenerateToken generates a JWT token
func (j *JWTService) GenerateToken(userID string, claims map[string]interface{}) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
		"iat":     time.Now().Unix(),
	})
	
	// Add custom claims
	for key, value := range claims {
		token.Claims.(jwt.MapClaims)[key] = value
	}
	
	return token.SignedString([]byte(j.secretKey))
}

// ValidateToken validates a JWT token
func (j *JWTService) ValidateToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(j.secretKey), nil
	})
}

// ExtractClaims extracts claims from a token
func (j *JWTService) ExtractClaims(token *jwt.Token) (map[string]interface{}, error) {
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return map[string]interface{}(claims), nil
	}
	return nil, jwt.ErrInvalidKey
}
`
}

func (ac *AuthCommands) generatePasswordService() string {
	return `package services

import (
	"golang.org/x/crypto/bcrypt"
)

type PasswordService struct{}

func NewPasswordService() *PasswordService {
	return &PasswordService{}
}

// HashPassword hashes a password
func (p *PasswordService) HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// CheckPassword checks a password against its hash
func (p *PasswordService) CheckPassword(password, hashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}
`
}

func (ac *AuthCommands) generateOTPService() string {
	return `package services

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

type OTPService struct {
	expiry time.Duration
}

func NewOTPService() *OTPService {
	return &OTPService{
		expiry: time.Minute * 5, // 5 minutes
	}
}

// GenerateOTP generates a 6-digit OTP
func (o *OTPService) GenerateOTP() (string, error) {
	otp, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", otp.Int64()), nil
}

// ValidateOTP validates an OTP
func (o *OTPService) ValidateOTP(otp, storedOTP string, createdAt time.Time) bool {
	if otp != storedOTP {
		return false
	}
	
	if time.Since(createdAt) > o.expiry {
		return false
	}
	
	return true
}
`
}

func (ac *AuthCommands) generateTwoFactorService() string {
	return `package services

import (
	"crypto/rand"
	"encoding/base32"
	"time"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

type TwoFactorService struct{}

func NewTwoFactorService() *TwoFactorService {
	return &TwoFactorService{}
}

// GenerateSecret generates a 2FA secret
func (t *TwoFactorService) GenerateSecret() (string, error) {
	secret := make([]byte, 20)
	_, err := rand.Read(secret)
	if err != nil {
		return "", err
	}
	return base32.StdEncoding.EncodeToString(secret), nil
}

// GenerateQRCode generates a QR code for 2FA setup
func (t *TwoFactorService) GenerateQRCode(secret, email string) (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Mithril App",
		AccountName: email,
		Secret:      []byte(secret),
	})
	if err != nil {
		return "", err
	}
	
	return key.URL(), nil
}

// ValidateCode validates a 2FA code
func (t *TwoFactorService) ValidateCode(secret, code string) bool {
	return totp.Validate(code, secret)
}
`
}

func (ac *AuthCommands) generateUserModel() string {
	return "package models\n\n" +
		"import (\n" +
		"\t\"time\"\n" +
		"\t\"github.com/google/uuid\"\n" +
		"\t\"gorm.io/gorm\"\n" +
		")\n\n" +
		"type User struct {\n" +
		"\tID                uuid.UUID      `json:\"id\" gorm:\"type:uuid;primary_key;default:gen_random_uuid()\"`\n" +
		"\tEmail             string         `json:\"email\" gorm:\"uniqueIndex;not null\"`\n" +
		"\tPassword          string         `json:\"-\" gorm:\"not null\"`\n" +
		"\tName              string         `json:\"name\" gorm:\"not null\"`\n" +
		"\tPhone             string         `json:\"phone\" gorm:\"uniqueIndex\"`\n" +
		"\tEmailVerifiedAt   *time.Time     `json:\"email_verified_at\"`\n" +
		"\tPhoneVerifiedAt   *time.Time     `json:\"phone_verified_at\"`\n" +
		"\tTwoFactorSecret   string         `json:\"-\"`\n" +
		"\tTwoFactorEnabled  bool           `json:\"two_factor_enabled\" gorm:\"default:false\"`\n" +
		"\tIsActive          bool           `json:\"is_active\" gorm:\"default:true\"`\n" +
		"\tIsSuperUser       bool           `json:\"is_super_user\" gorm:\"default:false\"`\n" +
		"\tLastLoginAt       *time.Time     `json:\"last_login_at\"`\n" +
		"\tCreatedAt         time.Time      `json:\"created_at\"`\n" +
		"\tUpdatedAt         time.Time      `json:\"updated_at\"`\n" +
		"\tDeletedAt         gorm.DeletedAt `json:\"deleted_at\" gorm:\"index\"`\n\n" +
		"\t// Relationships\n" +
		"\tRoles       []Role       `json:\"roles\" gorm:\"many2many:user_roles;\"`\n" +
		"\tPermissions []Permission `json:\"permissions\" gorm:\"many2many:user_permissions;\"`\n" +
		"}\n\n" +
		"func (User) TableName() string {\n" +
		"\treturn \"users\"\n" +
		"}\n"
}

func (ac *AuthCommands) generateRoleModel() string {
	return "package models\n\n" +
		"import (\n" +
		"\t\"time\"\n" +
		"\t\"github.com/google/uuid\"\n" +
		"\t\"gorm.io/gorm\"\n" +
		")\n\n" +
		"type Role struct {\n" +
		"\tID          uuid.UUID      `json:\"id\" gorm:\"type:uuid;primary_key;default:gen_random_uuid()\"`\n" +
		"\tName        string         `json:\"name\" gorm:\"uniqueIndex;not null\"`\n" +
		"\tDescription string         `json:\"description\"`\n" +
		"\tIsActive    bool           `json:\"is_active\" gorm:\"default:true\"`\n" +
		"\tCreatedAt   time.Time      `json:\"created_at\"`\n" +
		"\tUpdatedAt   time.Time      `json:\"updated_at\"`\n" +
		"\tDeletedAt   gorm.DeletedAt `json:\"deleted_at\" gorm:\"index\"`\n\n" +
		"\t// Relationships\n" +
		"\tUsers       []User       `json:\"users\" gorm:\"many2many:user_roles;\"`\n" +
		"\tPermissions []Permission `json:\"permissions\" gorm:\"many2many:role_permissions;\"`\n" +
		"}\n\n" +
		"func (Role) TableName() string {\n" +
		"\treturn \"roles\"\n" +
		"}\n"
}

func (ac *AuthCommands) generatePermissionModel() string {
	return "package models\n\n" +
		"import (\n" +
		"\t\"time\"\n" +
		"\t\"github.com/google/uuid\"\n" +
		"\t\"gorm.io/gorm\"\n" +
		")\n\n" +
		"type Permission struct {\n" +
		"\tID          uuid.UUID      `json:\"id\" gorm:\"type:uuid;primary_key;default:gen_random_uuid()\"`\n" +
		"\tName        string         `json:\"name\" gorm:\"uniqueIndex;not null\"`\n" +
		"\tDescription string         `json:\"description\"`\n" +
		"\tIsActive    bool           `json:\"is_active\" gorm:\"default:true\"`\n" +
		"\tCreatedAt   time.Time      `json:\"created_at\"`\n" +
		"\tUpdatedAt   time.Time      `json:\"updated_at\"`\n" +
		"\tDeletedAt   gorm.DeletedAt `json:\"deleted_at\" gorm:\"index\"`\n\n" +
		"\t// Relationships\n" +
		"\tUsers []User `json:\"users\" gorm:\"many2many:user_permissions;\"`\n" +
		"\tRoles []Role `json:\"roles\" gorm:\"many2many:role_permissions;\"`\n" +
		"}\n\n" +
		"func (Permission) TableName() string {\n" +
		"\treturn \"permissions\"\n" +
		"}\n"
}

func (ac *AuthCommands) generateUserRoleModel() string {
	return "package models\n\n" +
		"import (\n" +
		"\t\"time\"\n" +
		"\t\"github.com/google/uuid\"\n" +
		")\n\n" +
		"type UserRole struct {\n" +
		"\tID        uuid.UUID `json:\"id\" gorm:\"type:uuid;primary_key;default:gen_random_uuid()\"`\n" +
		"\tUserID    uuid.UUID `json:\"user_id\" gorm:\"not null\"`\n" +
		"\tRoleID    uuid.UUID `json:\"role_id\" gorm:\"not null\"`\n" +
		"\tCreatedAt time.Time `json:\"created_at\"`\n\n" +
		"\t// Relationships\n" +
		"\tUser User `json:\"user\" gorm:\"foreignKey:UserID\"`\n" +
		"\tRole Role `json:\"role\" gorm:\"foreignKey:RoleID\"`\n" +
		"}\n\n" +
		"func (UserRole) TableName() string {\n" +
		"\treturn \"user_roles\"\n" +
		"}\n"
}

func (ac *AuthCommands) generateRolePermissionModel() string {
	return "package models\n\n" +
		"import (\n" +
		"\t\"time\"\n" +
		"\t\"github.com/google/uuid\"\n" +
		")\n\n" +
		"type RolePermission struct {\n" +
		"\tID           uuid.UUID `json:\"id\" gorm:\"type:uuid;primary_key;default:gen_random_uuid()\"`\n" +
		"\tRoleID       uuid.UUID `json:\"role_id\" gorm:\"not null\"`\n" +
		"\tPermissionID uuid.UUID `json:\"permission_id\" gorm:\"not null\"`\n" +
		"\tCreatedAt    time.Time `json:\"created_at\"`\n\n" +
		"\t// Relationships\n" +
		"\tRole       Role       `json:\"role\" gorm:\"foreignKey:RoleID\"`\n" +
		"\tPermission Permission `json:\"permission\" gorm:\"foreignKey:PermissionID\"`\n" +
		"}\n\n" +
		"func (RolePermission) TableName() string {\n" +
		"\treturn \"role_permissions\"\n" +
		"}\n"
}

func (ac *AuthCommands) generateAuthRoutes(authType string) string {
	return `package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/mithril-framework/mithril/app/auth/controllers"
)

// RegisterAuthRoutes registers authentication routes
func RegisterAuthRoutes(app *fiber.App) {
	authController := controllers.NewAuthController()
	
	// Authentication routes
	auth := app.Group("/auth")
	
	// Registration and login
	auth.Post("/register", authController.Register)
	auth.Post("/login", authController.Login)
	auth.Post("/logout", authController.Logout)
	auth.Post("/refresh", authController.Refresh)
	
	// Password reset
	auth.Post("/forgot-password", authController.ForgotPassword)
	auth.Post("/reset-password", authController.ResetPassword)
	
	// Email verification
	auth.Post("/verify-email", authController.VerifyEmail)
	
	// OTP
	auth.Post("/send-otp", authController.SendOTP)
	auth.Post("/verify-otp", authController.VerifyOTP)
	
	// 2FA
	auth.Post("/enable-2fa", authController.Enable2FA)
	auth.Post("/verify-2fa", authController.Verify2FA)
	auth.Post("/disable-2fa", authController.Disable2FA)
}
`
}

func (ac *AuthCommands) generateAuthMiddleware() string {
	return `package middleware

import (
	"strings"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware validates JWT tokens
func AuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(401).JSON(fiber.Map{
				"error": "Authorization header required",
			})
		}
		
		// Extract token from "Bearer <token>"
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			return c.Status(401).JSON(fiber.Map{
				"error": "Invalid authorization header format",
			})
		}
		
		tokenString := tokenParts[1]
		
		// Validate token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrInvalidKey
			}
			return []byte("your-secret-key"), nil
		})
		
		if err != nil || !token.Valid {
			return c.Status(401).JSON(fiber.Map{
				"error": "Invalid token",
			})
		}
		
		// Set user ID in context
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if userID, ok := claims["user_id"].(string); ok {
				c.Locals("user_id", userID)
			}
		}
		
		return c.Next()
	}
}
`
}

func (ac *AuthCommands) generateRBACMiddleware() string {
	return `package middleware

import (
	"github.com/gofiber/fiber/v2"
)

// RequireRole middleware checks if user has required role
func RequireRole(role string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get user ID from context
		userID := c.Locals("user_id")
		if userID == nil {
			return c.Status(401).JSON(fiber.Map{
				"error": "User not authenticated",
			})
		}
		
		// Check if user has required role
		// This would query the database to check user roles
		// For now, just return next
		return c.Next()
	}
}

// RequirePermission middleware checks if user has required permission
func RequirePermission(permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get user ID from context
		userID := c.Locals("user_id")
		if userID == nil {
			return c.Status(401).JSON(fiber.Map{
				"error": "User not authenticated",
			})
		}
		
		// Check if user has required permission
		// This would query the database to check user permissions
		// For now, just return next
		return c.Next()
	}
}
`
}

func (ac *AuthCommands) generateAuthService() string {
	return `package services

import (
	"context"
)

type AuthService struct {
	// Add dependencies here
}

func NewAuthService() *AuthService {
	return &AuthService{}
}

// Register registers a new user
func (a *AuthService) Register(ctx context.Context, email, password, name string) error {
	// Implement registration logic
	return nil
}

// Login authenticates a user
func (a *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	// Implement login logic
	return "", nil
}

// Logout logs out a user
func (a *AuthService) Logout(ctx context.Context, userID string) error {
	// Implement logout logic
	return nil
}

// RefreshToken refreshes a user's token
func (a *AuthService) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	// Implement token refresh logic
	return "", nil
}
`
}

func (ac *AuthCommands) generateAuthSchemas() string {
	return "package schemas\n\n" +
		"// RegisterRequest represents user registration request\n" +
		"type RegisterRequest struct {\n" +
		"\tEmail    string `json:\"email\" validate:\"required,email\"`\n" +
		"\tPassword string `json:\"password\" validate:\"required,min=8\"`\n" +
		"\tName     string `json:\"name\" validate:\"required\"`\n" +
		"\tPhone    string `json:\"phone\" validate:\"omitempty\"`\n" +
		"}\n\n" +
		"// LoginRequest represents user login request\n" +
		"type LoginRequest struct {\n" +
		"\tEmail    string `json:\"email\" validate:\"required,email\"`\n" +
		"\tPassword string `json:\"password\" validate:\"required\"`\n" +
		"}\n\n" +
		"// ForgotPasswordRequest represents forgot password request\n" +
		"type ForgotPasswordRequest struct {\n" +
		"\tEmail string `json:\"email\" validate:\"required,email\"`\n" +
		"}\n\n" +
		"// ResetPasswordRequest represents reset password request\n" +
		"type ResetPasswordRequest struct {\n" +
		"\tToken    string `json:\"token\" validate:\"required\"`\n" +
		"\tPassword string `json:\"password\" validate:\"required,min=8\"`\n" +
		"}\n\n" +
		"// VerifyEmailRequest represents email verification request\n" +
		"type VerifyEmailRequest struct {\n" +
		"\tToken string `json:\"token\" validate:\"required\"`\n" +
		"}\n\n" +
		"// SendOTPRequest represents send OTP request\n" +
		"type SendOTPRequest struct {\n" +
		"\tPhone string `json:\"phone\" validate:\"required\"`\n" +
		"}\n\n" +
		"// VerifyOTPRequest represents verify OTP request\n" +
		"type VerifyOTPRequest struct {\n" +
		"\tPhone string `json:\"phone\" validate:\"required\"`\n" +
		"\tOTP   string `json:\"otp\" validate:\"required,len=6\"`\n" +
		"}\n\n" +
		"// Enable2FARequest represents enable 2FA request\n" +
		"type Enable2FARequest struct {\n" +
		"\tPassword string `json:\"password\" validate:\"required\"`\n" +
		"}\n\n" +
		"// Verify2FARequest represents verify 2FA request\n" +
		"type Verify2FARequest struct {\n" +
		"\tCode string `json:\"code\" validate:\"required,len=6\"`\n" +
		"}\n\n" +
		"// Disable2FARequest represents disable 2FA request\n" +
		"type Disable2FARequest struct {\n" +
		"\tPassword string `json:\"password\" validate:\"required\"`\n" +
		"\tCode     string `json:\"code\" validate:\"required,len=6\"`\n" +
		"}\n\n" +
		"// AuthResponse represents authentication response\n" +
		"type AuthResponse struct {\n" +
		"\tUser         UserResponse `json:\"user\"`\n" +
		"\tAccessToken  string       `json:\"access_token\"`\n" +
		"\tRefreshToken string       `json:\"refresh_token\"`\n" +
		"\tExpiresIn    int          `json:\"expires_in\"`\n" +
		"}\n\n" +
		"// UserResponse represents user response\n" +
		"type UserResponse struct {\n" +
		"\tID               string `json:\"id\"`\n" +
		"\tEmail            string `json:\"email\"`\n" +
		"\tName             string `json:\"name\"`\n" +
		"\tPhone            string `json:\"phone\"`\n" +
		"\tEmailVerifiedAt  string `json:\"email_verified_at,omitempty\"`\n" +
		"\tPhoneVerifiedAt  string `json:\"phone_verified_at,omitempty\"`\n" +
		"\tTwoFactorEnabled bool   `json:\"two_factor_enabled\"`\n" +
		"\tIsActive         bool   `json:\"is_active\"`\n" +
		"\tIsSuperUser      bool   `json:\"is_super_user\"`\n" +
		"\tLastLoginAt      string `json:\"last_login_at,omitempty\"`\n" +
		"\tCreatedAt        string `json:\"created_at\"`\n" +
		"\tUpdatedAt        string `json:\"updated_at\"`\n" +
		"}\n"
}

func (ac *AuthCommands) generateLoginView() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Login - Mithril App</title>
</head>
<body>
    <h1>Login</h1>
    <form method="POST" action="/auth/login">
        <div>
            <label for="email">Email:</label>
            <input type="email" id="email" name="email" required>
        </div>
        <div>
            <label for="password">Password:</label>
            <input type="password" id="password" name="password" required>
        </div>
        <button type="submit">Login</button>
    </form>
    <p><a href="/auth/register">Don't have an account? Register</a></p>
</body>
</html>
`
}

func (ac *AuthCommands) generateRegisterView() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Register - Mithril App</title>
</head>
<body>
    <h1>Register</h1>
    <form method="POST" action="/auth/register">
        <div>
            <label for="name">Name:</label>
            <input type="text" id="name" name="name" required>
        </div>
        <div>
            <label for="email">Email:</label>
            <input type="email" id="email" name="email" required>
        </div>
        <div>
            <label for="password">Password:</label>
            <input type="password" id="password" name="password" required>
        </div>
        <div>
            <label for="phone">Phone (optional):</label>
            <input type="tel" id="phone" name="phone">
        </div>
        <button type="submit">Register</button>
    </form>
    <p><a href="/auth/login">Already have an account? Login</a></p>
</body>
</html>
`
}

func (ac *AuthCommands) generateAuthMigration() string {
	return `package migrations

import (
	"gorm.io/gorm"
)

type CreateAuthTables struct{}

func (m *CreateAuthTables) Up(db *gorm.DB) error {
	// This would create all authentication tables
	// For now, just return nil
	return nil
}

func (m *CreateAuthTables) Down(db *gorm.DB) error {
	// This would drop all authentication tables
	// For now, just return nil
	return nil
}
`
}

// Validation functions

func isValidEmail(email string) bool {
	// Basic email validation
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

func isValidRoleName(name string) bool {
	if name == "" {
		return false
	}
	// Check for valid characters (alphanumeric and hyphens)
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

func isValidPermissionName(name string) bool {
	return isValidRoleName(name)
}
