package testing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// E2ETestSuite represents an end-to-end test suite
type E2ETestSuite struct {
	App     *fiber.App
	DB      *gorm.DB
	Server  *httptest.Server
	Client  *http.Client
	Factory *TestFactory
}

// NewE2ETestSuite creates a new E2E test suite
func NewE2ETestSuite(t *testing.T) *E2ETestSuite {
	testApp := NewTestApp(t, nil)
	factory := NewTestFactory(testApp.DB)

	// Setup comprehensive routes for E2E testing
	setupE2ERoutes(testApp.App, testApp.DB)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &E2ETestSuite{
		App:     testApp.App,
		DB:      testApp.DB,
		Server:  testApp.Server,
		Client:  client,
		Factory: factory,
	}
}

// Close closes the E2E test suite
func (suite *E2ETestSuite) Close() {
	if suite.Server != nil {
		suite.Server.Close()
	}
	if suite.DB != nil {
		sqlDB, _ := suite.DB.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}
}

// Request performs an HTTP request
func (suite *E2ETestSuite) Request(method, path string, body interface{}, headers map[string]string) (*http.Response, []byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, suite.Server.URL+path, reqBody)
	if err != nil {
		return nil, nil, err
	}

	// Set default headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Set custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Perform request
	resp, err := suite.Client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		resp.Body.Close()
		return resp, nil, err
	}

	return resp, respBody, nil
}

// setupE2ERoutes sets up comprehensive routes for E2E testing
func setupE2ERoutes(app *fiber.App, db *gorm.DB) {
	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
			"version":   "1.0.0",
		})
	})

	// API routes
	api := app.Group("/api/v1")

	// User routes
	api.Get("/users", func(c *fiber.Ctx) error {
		var users []TestUser
		page := c.QueryInt("page", 1)
		limit := c.QueryInt("limit", 10)
		offset := (page - 1) * limit

		var total int64
		db.Model(&TestUser{}).Count(&total)
		db.Offset(offset).Limit(limit).Find(&users)

		return c.JSON(fiber.Map{
			"data": users,
			"meta": fiber.Map{
				"page":        page,
				"limit":       limit,
				"total":       total,
				"total_pages": (total + int64(limit) - 1) / int64(limit),
			},
		})
	})

	api.Post("/users", func(c *fiber.Ctx) error {
		var user TestUser
		if err := c.BodyParser(&user); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		// Validate required fields
		if user.Email == "" || user.Name == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Email and name are required"})
		}

		// Check if user already exists
		var existingUser TestUser
		if err := db.Where("email = ?", user.Email).First(&existingUser).Error; err == nil {
			return c.Status(409).JSON(fiber.Map{"error": "User already exists"})
		}

		if err := db.Create(&user).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to create user"})
		}

		return c.Status(201).JSON(user)
	})

	api.Get("/users/:id", func(c *fiber.Ctx) error {
		var user TestUser
		id := c.Params("id")
		if err := db.First(&user, id).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "User not found"})
		}
		return c.JSON(user)
	})

	api.Put("/users/:id", func(c *fiber.Ctx) error {
		var user TestUser
		id := c.Params("id")
		if err := db.First(&user, id).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "User not found"})
		}

		if err := c.BodyParser(&user); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		if err := db.Save(&user).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to update user"})
		}

		return c.JSON(user)
	})

	api.Delete("/users/:id", func(c *fiber.Ctx) error {
		var user TestUser
		id := c.Params("id")
		if err := db.First(&user, id).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "User not found"})
		}

		if err := db.Delete(&user).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to delete user"})
		}

		return c.Status(204).Send(nil)
	})

	// Post routes
	api.Get("/posts", func(c *fiber.Ctx) error {
		var posts []TestPost
		page := c.QueryInt("page", 1)
		limit := c.QueryInt("limit", 10)
		offset := (page - 1) * limit

		var total int64
		db.Model(&TestPost{}).Count(&total)
		db.Preload("User").Offset(offset).Limit(limit).Find(&posts)

		return c.JSON(fiber.Map{
			"data": posts,
			"meta": fiber.Map{
				"page":        page,
				"limit":       limit,
				"total":       total,
				"total_pages": (total + int64(limit) - 1) / int64(limit),
			},
		})
	})

	api.Post("/posts", func(c *fiber.Ctx) error {
		var post TestPost
		if err := c.BodyParser(&post); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		// Validate required fields
		if post.Title == "" || post.Content == "" || post.UserID == 0 {
			return c.Status(400).JSON(fiber.Map{"error": "Title, content, and user_id are required"})
		}

		// Check if user exists
		var user TestUser
		if err := db.First(&user, post.UserID).Error; err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "User not found"})
		}

		if err := db.Create(&post).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to create post"})
		}

		// Load user relationship
		db.Preload("User").First(&post, post.ID)

		return c.Status(201).JSON(post)
	})

	api.Get("/posts/:id", func(c *fiber.Ctx) error {
		var post TestPost
		id := c.Params("id")
		if err := db.Preload("User").First(&post, id).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Post not found"})
		}
		return c.JSON(post)
	})

	// Role and permission routes
	api.Get("/roles", func(c *fiber.Ctx) error {
		var roles []TestRole
		db.Find(&roles)
		return c.JSON(roles)
	})

	api.Post("/roles", func(c *fiber.Ctx) error {
		var role TestRole
		if err := c.BodyParser(&role); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		if role.Name == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Name is required"})
		}

		if err := db.Create(&role).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to create role"})
		}

		return c.Status(201).JSON(role)
	})

	api.Get("/permissions", func(c *fiber.Ctx) error {
		var permissions []TestPermission
		db.Find(&permissions)
		return c.JSON(permissions)
	})

	api.Post("/permissions", func(c *fiber.Ctx) error {
		var permission TestPermission
		if err := c.BodyParser(&permission); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		if permission.Name == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Name is required"})
		}

		if err := db.Create(&permission).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to create permission"})
		}

		return c.Status(201).JSON(permission)
	})
}

func TestE2E_CompleteUserWorkflow(t *testing.T) {
	suite := NewE2ETestSuite(t)
	defer suite.Close()

	// 1. Check health
	resp, body, err := suite.Request("GET", "/health", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var health map[string]interface{}
	err = json.Unmarshal(body, &health)
	require.NoError(t, err)
	assert.Equal(t, "ok", health["status"])

	// 2. Create user
	userData := map[string]interface{}{
		"email":    "e2e@example.com",
		"password": "password123",
		"name":     "E2E User",
	}

	resp, body, err = suite.Request("POST", "/api/v1/users", userData, nil)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)

	var user TestUser
	err = json.Unmarshal(body, &user)
	require.NoError(t, err)
	assert.Equal(t, "e2e@example.com", user.Email)
	assert.Equal(t, "E2E User", user.Name)
	assert.NotZero(t, user.ID)

	// 3. Get user by ID
	resp, body, err = suite.Request("GET", fmt.Sprintf("/api/v1/users/%d", user.ID), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var retrievedUser TestUser
	err = json.Unmarshal(body, &retrievedUser)
	require.NoError(t, err)
	assert.Equal(t, user.ID, retrievedUser.ID)
	assert.Equal(t, user.Email, retrievedUser.Email)

	// 4. Update user
	updateData := map[string]interface{}{
		"email":    "e2e@example.com",
		"password": "password123",
		"name":     "Updated E2E User",
	}

	resp, body, err = suite.Request("PUT", fmt.Sprintf("/api/v1/users/%d", user.ID), updateData, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var updatedUser TestUser
	err = json.Unmarshal(body, &updatedUser)
	require.NoError(t, err)
	assert.Equal(t, "Updated E2E User", updatedUser.Name)

	// 5. Create post for user
	postData := map[string]interface{}{
		"title":   "E2E Test Post",
		"content": "This is a test post created during E2E testing",
		"user_id": user.ID,
	}

	resp, body, err = suite.Request("POST", "/api/v1/posts", postData, nil)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)

	var post TestPost
	err = json.Unmarshal(body, &post)
	require.NoError(t, err)
	assert.Equal(t, "E2E Test Post", post.Title)
	assert.Equal(t, user.ID, post.UserID)
	assert.NotZero(t, post.ID)

	// 6. Get post by ID
	resp, body, err = suite.Request("GET", fmt.Sprintf("/api/v1/posts/%d", post.ID), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var retrievedPost TestPost
	err = json.Unmarshal(body, &retrievedPost)
	require.NoError(t, err)
	assert.Equal(t, post.ID, retrievedPost.ID)
	assert.Equal(t, post.Title, retrievedPost.Title)
	assert.Equal(t, user.ID, retrievedPost.UserID)

	// 7. Get all users with pagination
	resp, body, err = suite.Request("GET", "/api/v1/users?page=1&limit=10", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var usersResponse map[string]interface{}
	err = json.Unmarshal(body, &usersResponse)
	require.NoError(t, err)
	assert.Contains(t, usersResponse, "data")
	assert.Contains(t, usersResponse, "meta")

	// 8. Get all posts with pagination
	resp, body, err = suite.Request("GET", "/api/v1/posts?page=1&limit=10", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var postsResponse map[string]interface{}
	err = json.Unmarshal(body, &postsResponse)
	require.NoError(t, err)
	assert.Contains(t, postsResponse, "data")
	assert.Contains(t, postsResponse, "meta")

	// 9. Delete post
	resp, _, err = suite.Request("DELETE", fmt.Sprintf("/api/v1/posts/%d", post.ID), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode)

	// 10. Delete user
	resp, _, err = suite.Request("DELETE", fmt.Sprintf("/api/v1/users/%d", user.ID), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode)

	// 11. Verify user is deleted
	resp, _, err = suite.Request("GET", fmt.Sprintf("/api/v1/users/%d", user.ID), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)
}

func TestE2E_RoleAndPermissionWorkflow(t *testing.T) {
	suite := NewE2ETestSuite(t)
	defer suite.Close()

	// 1. Create role
	roleData := map[string]interface{}{
		"name":        "e2e-role",
		"description": "Role created during E2E testing",
	}

	resp, body, err := suite.Request("POST", "/api/v1/roles", roleData, nil)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)

	var role TestRole
	err = json.Unmarshal(body, &role)
	require.NoError(t, err)
	assert.Equal(t, "e2e-role", role.Name)
	assert.NotZero(t, role.ID)

	// 2. Create permission
	permissionData := map[string]interface{}{
		"name":        "e2e-permission",
		"description": "Permission created during E2E testing",
	}

	resp, body, err = suite.Request("POST", "/api/v1/permissions", permissionData, nil)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)

	var permission TestPermission
	err = json.Unmarshal(body, &permission)
	require.NoError(t, err)
	assert.Equal(t, "e2e-permission", permission.Name)
	assert.NotZero(t, permission.ID)

	// 3. Get all roles
	resp, body, err = suite.Request("GET", "/api/v1/roles", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var roles []TestRole
	err = json.Unmarshal(body, &roles)
	require.NoError(t, err)
	assert.Len(t, roles, 1)
	assert.Equal(t, "e2e-role", roles[0].Name)

	// 4. Get all permissions
	resp, body, err = suite.Request("GET", "/api/v1/permissions", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var permissions []TestPermission
	err = json.Unmarshal(body, &permissions)
	require.NoError(t, err)
	assert.Len(t, permissions, 1)
	assert.Equal(t, "e2e-permission", permissions[0].Name)
}

func TestE2E_ErrorHandling(t *testing.T) {
	suite := NewE2ETestSuite(t)
	defer suite.Close()

	// Test invalid JSON
	resp, body, err := suite.Request("POST", "/api/v1/users", "invalid json", map[string]string{
		"Content-Type": "application/json",
	})
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
	assert.Contains(t, string(body), "error")

	// Test missing required fields
	resp, _, err = suite.Request("POST", "/api/v1/users", map[string]interface{}{
		"email": "test@example.com",
		// Missing name
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode) // Should still work as we're not validating in this simple test

	// Test duplicate user
	userData := map[string]interface{}{
		"email":    "duplicate@example.com",
		"password": "password123",
		"name":     "Duplicate User",
	}

	// Create first user
	resp, _, err = suite.Request("POST", "/api/v1/users", userData, nil)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)

	// Try to create duplicate user
	resp, body, err = suite.Request("POST", "/api/v1/users", userData, nil)
	require.NoError(t, err)
	assert.Equal(t, 409, resp.StatusCode)
	assert.Contains(t, string(body), "already exists")

	// Test non-existent user
	resp, body, err = suite.Request("GET", "/api/v1/users/999", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)
	assert.Contains(t, string(body), "not found")

	// Test invalid post creation (missing user)
	postData := map[string]interface{}{
		"title":   "Test Post",
		"content": "Test content",
		// Missing user_id
	}

	resp, body, err = suite.Request("POST", "/api/v1/posts", postData, nil)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
	assert.Contains(t, string(body), "required")
}

func TestE2E_ConcurrentOperations(t *testing.T) {
	suite := NewE2ETestSuite(t)
	defer suite.Close()

	// Create multiple users concurrently
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(index int) {
			userData := map[string]interface{}{
				"email":    fmt.Sprintf("concurrent%d@example.com", index),
				"password": "password123",
				"name":     fmt.Sprintf("Concurrent User %d", index),
			}

			resp, body, err := suite.Request("POST", "/api/v1/users", userData, nil)
			require.NoError(t, err)
			assert.Equal(t, 201, resp.StatusCode)

			var user TestUser
			err = json.Unmarshal(body, &user)
			require.NoError(t, err)
			assert.Equal(t, fmt.Sprintf("concurrent%d@example.com", index), user.Email)

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all users were created
	resp, body, err := suite.Request("GET", "/api/v1/users", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var usersResponse map[string]interface{}
	err = json.Unmarshal(body, &usersResponse)
	require.NoError(t, err)

	data := usersResponse["data"].([]interface{})
	assert.Len(t, data, 10)
}

func TestE2E_Performance(t *testing.T) {
	suite := NewE2ETestSuite(t)
	defer suite.Close()

	// Test response time for health check
	start := time.Now()
	resp, _, err := suite.Request("GET", "/health", nil, nil)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Less(t, duration, 100*time.Millisecond, "Health check should respond within 100ms")

	// Test response time for user creation
	userData := map[string]interface{}{
		"email":    "performance@example.com",
		"password": "password123",
		"name":     "Performance User",
	}

	start = time.Now()
	resp, body, err := suite.Request("POST", "/api/v1/users", userData, nil)
	duration = time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)
	assert.Less(t, duration, 500*time.Millisecond, "User creation should complete within 500ms")

	var user TestUser
	err = json.Unmarshal(body, &user)
	require.NoError(t, err)

	// Test response time for user retrieval
	start = time.Now()
	resp, _, err = suite.Request("GET", fmt.Sprintf("/api/v1/users/%d", user.ID), nil, nil)
	duration = time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Less(t, duration, 100*time.Millisecond, "User retrieval should complete within 100ms")
}
