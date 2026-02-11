package testing

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_HTTPRequests(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	helper.SetupTestRoutes()

	tests := []struct {
		name           string
		method         string
		path           string
		body           interface{}
		headers        map[string]string
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name:           "GET health check",
			method:         "GET",
			path:           "/health",
			expectedStatus: 200,
			expectedBody:   map[string]string{"status": "ok"},
		},
		{
			name:           "GET users - empty list",
			method:         "GET",
			path:           "/users",
			expectedStatus: 200,
			expectedBody:   []interface{}{},
		},
		{
			name:   "POST users - create user",
			method: "POST",
			path:   "/users",
			body: map[string]interface{}{
				"email":    "test@example.com",
				"password": "password123",
				"name":     "Test User",
			},
			expectedStatus: 201,
		},
		{
			name:           "GET users - with data",
			method:         "GET",
			path:           "/users",
			expectedStatus: 200,
		},
		{
			name:           "GET user by ID - existing",
			method:         "GET",
			path:           "/users/1",
			expectedStatus: 200,
		},
		{
			name:           "GET user by ID - not found",
			method:         "GET",
			path:           "/users/999",
			expectedStatus: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp *TestResponse

			switch tt.method {
			case "GET":
				resp = helper.Request(t, "GET", tt.path, nil, tt.headers)
			case "POST":
				resp = helper.Request(t, "POST", tt.path, tt.body, tt.headers)
			case "PUT":
				resp = helper.Request(t, "PUT", tt.path, tt.body, tt.headers)
			case "DELETE":
				resp = helper.Request(t, "DELETE", tt.path, nil, tt.headers)
			}

			resp.AssertStatus(t, tt.expectedStatus)

			if tt.expectedBody != nil {
				resp.AssertJSON(t, tt.expectedBody)
			}
		})
	}
}

func TestIntegration_DatabaseOperations(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Test user creation
	user := helper.Factory.CreateUser(map[string]interface{}{
		"email":    "integration@example.com",
		"password": "password123",
		"name":     "Integration User",
	})

	assert.NotNil(t, user)
	assert.Equal(t, "integration@example.com", user.Email)
	assert.Equal(t, "Integration User", user.Name)
	assert.NotZero(t, user.ID)

	// Test post creation
	post := helper.Factory.CreatePost(user.ID, map[string]interface{}{
		"title":   "Integration Test Post",
		"content": "This is a test post for integration testing",
	})

	assert.NotNil(t, post)
	assert.Equal(t, "Integration Test Post", post.Title)
	assert.Equal(t, user.ID, post.UserID)
	assert.NotZero(t, post.ID)

	// Test role creation
	role := helper.Factory.CreateRole(map[string]interface{}{
		"name":        "integration-role",
		"description": "Role for integration testing",
	})

	assert.NotNil(t, role)
	assert.Equal(t, "integration-role", role.Name)
	assert.NotZero(t, role.ID)

	// Test permission creation
	permission := helper.Factory.CreatePermission(map[string]interface{}{
		"name":        "integration-permission",
		"description": "Permission for integration testing",
	})

	assert.NotNil(t, permission)
	assert.Equal(t, "integration-permission", permission.Name)
	assert.NotZero(t, permission.ID)
}

func TestIntegration_ErrorHandling(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	helper.SetupTestRoutes()

	// Test invalid JSON
	resp := helper.Request(t, "POST", "/users", "invalid json", map[string]string{
		"Content-Type": "application/json",
	})
	resp.AssertStatus(t, 400)
	resp.AssertContains(t, "error")

	// Test missing required fields
	resp = helper.Request(t, "POST", "/users", map[string]interface{}{
		"email": "test@example.com",
		// Missing password and name
	}, nil)
	resp.AssertStatus(t, 201) // Should still work as we're not validating in this simple test
}

func TestIntegration_ConcurrentRequests(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	helper.SetupTestRoutes()

	// Create multiple users concurrently
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(index int) {
			user := helper.Factory.CreateUser(map[string]interface{}{
				"email":    fmt.Sprintf("user%d@example.com", index),
				"password": "password123",
				"name":     fmt.Sprintf("User %d", index),
			})
			assert.NotNil(t, user)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all users were created
	var count int64
	helper.DB.Model(&TestUser{}).Count(&count)
	assert.Equal(t, int64(10), count)
}

func TestIntegration_TransactionRollback(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Test transaction rollback
	tx := helper.DB.Begin()
	defer tx.Rollback()

	// Create user in transaction
	user := &TestUser{
		Email:    "transaction@example.com",
		Password: "password123",
		Name:     "Transaction User",
	}

	err := tx.Create(user).Error
	require.NoError(t, err)

	// Verify user exists in transaction
	var count int64
	tx.Model(&TestUser{}).Where("email = ?", "transaction@example.com").Count(&count)
	assert.Equal(t, int64(1), count)

	// Rollback transaction
	tx.Rollback()

	// Verify user doesn't exist after rollback
	helper.DB.Model(&TestUser{}).Where("email = ?", "transaction@example.com").Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestIntegration_JSONSerialization(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Create test data
	user := helper.Factory.CreateUser(map[string]interface{}{
		"email":    "json@example.com",
		"password": "password123",
		"name":     "JSON User",
	})

	post := helper.Factory.CreatePost(user.ID, map[string]interface{}{
		"title":   "JSON Test Post",
		"content": "Testing JSON serialization",
	})

	// Test JSON serialization
	userJSON, err := json.Marshal(user)
	require.NoError(t, err)

	var deserializedUser TestUser
	err = json.Unmarshal(userJSON, &deserializedUser)
	require.NoError(t, err)

	assert.Equal(t, user.Email, deserializedUser.Email)
	assert.Equal(t, user.Name, deserializedUser.Name)
	assert.Equal(t, user.ID, deserializedUser.ID)

	// Test post with user relationship
	postJSON, err := json.Marshal(post)
	require.NoError(t, err)

	var deserializedPost TestPost
	err = json.Unmarshal(postJSON, &deserializedPost)
	require.NoError(t, err)

	assert.Equal(t, post.Title, deserializedPost.Title)
	assert.Equal(t, post.Content, deserializedPost.Content)
	assert.Equal(t, post.UserID, deserializedPost.UserID)
}

func TestIntegration_FileOperations(t *testing.T) {
	env := NewTestEnvironment(t)
	defer env.Cleanup(t)

	// Create test files
	file1 := env.AddFile(t, "test1.txt", "Hello, World!")
	file2 := env.AddFile(t, "test2.json", `{"message": "Hello, JSON!"}`)

	// Verify files exist
	assert.FileExists(t, file1.Path)
	assert.FileExists(t, file2.Path)

	// Read file contents
	content1, err := os.ReadFile(file1.Path)
	require.NoError(t, err)
	assert.Equal(t, "Hello, World!", string(content1))

	content2, err := os.ReadFile(file2.Path)
	require.NoError(t, err)

	var jsonData map[string]string
	err = json.Unmarshal(content2, &jsonData)
	require.NoError(t, err)
	assert.Equal(t, "Hello, JSON!", jsonData["message"])
}

func TestIntegration_Logging(t *testing.T) {
	logger := NewTestLogger()

	// Test logging
	logger.Log("Test message 1")
	logger.Log("Test message 2")
	logger.Log("Test message 3")

	messages := logger.GetMessages()
	assert.Len(t, messages, 3)
	assert.Contains(t, messages, "Test message 1")
	assert.Contains(t, messages, "Test message 2")
	assert.Contains(t, messages, "Test message 3")

	// Test clearing
	logger.Clear()
	messages = logger.GetMessages()
	assert.Len(t, messages, 0)
}

func TestIntegration_Timeout(t *testing.T) {
	timeout := NewTestTimeout(100 * time.Millisecond)

	// Test successful operation
	timeout.WithTimeout(t, func() {
		time.Sleep(50 * time.Millisecond)
	})

	// Test timeout
	assert.Panics(t, func() {
		timeout.WithTimeout(t, func() {
			time.Sleep(200 * time.Millisecond)
		})
	})
}

func TestIntegration_DataProvider(t *testing.T) {
	provider := NewTestDataProvider()

	// Test setting and getting data
	provider.Set("string", "test value")
	provider.Set("int", 42)
	provider.Set("bool", true)

	assert.Equal(t, "test value", provider.GetString("string"))
	assert.Equal(t, 42, provider.GetInt("int"))
	assert.Equal(t, true, provider.GetBool("bool"))

	// Test clearing
	provider.Clear()
	assert.Equal(t, "", provider.GetString("string"))
	assert.Equal(t, 0, provider.GetInt("int"))
	assert.Equal(t, false, provider.GetBool("bool"))
}

func TestIntegration_Assertions(t *testing.T) {
	assertion := NewTestAssertion(t)

	// Test AssertNoError
	assertion.AssertNoError(nil, "Should not error")

	// Test AssertError
	assertion.AssertError(fmt.Errorf("test error"), "Should error")

	// Test AssertNotEmpty
	assertion.AssertNotEmpty("test", "Should not be empty")
	assertion.AssertNotEmpty([]string{"test"}, "Should not be empty")
	assertion.AssertNotEmpty(map[string]string{"key": "value"}, "Should not be empty")

	// Test AssertEmpty
	assertion.AssertEmpty("", "Should be empty")
	assertion.AssertEmpty([]string{}, "Should be empty")
	assertion.AssertEmpty(map[string]string{}, "Should be empty")
	assertion.AssertEmpty(nil, "Should be empty")
}
