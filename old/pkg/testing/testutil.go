package testing

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestApp represents a test application instance
type TestApp struct {
	App    *fiber.App
	DB     *gorm.DB
	Server *httptest.Server
}

// TestConfig holds test configuration
type TestConfig struct {
	DatabaseDriver string
	DatabaseDSN    string
	Port           int
	Debug          bool
}

// DefaultTestConfig returns default test configuration
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		DatabaseDriver: "sqlite",
		DatabaseDSN:    ":memory:",
		Port:           3001,
		Debug:          true,
	}
}

// NewTestApp creates a new test application
func NewTestApp(t *testing.T, config *TestConfig) *TestApp {
	if config == nil {
		config = DefaultTestConfig()
	}

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Setup database
	var db *gorm.DB
	var err error

	switch config.DatabaseDriver {
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(config.DatabaseDSN), &gorm.Config{})
		require.NoError(t, err)
	default:
		t.Fatalf("Unsupported database driver: %s", config.DatabaseDriver)
	}

	// Auto-migrate common tables
	err = db.AutoMigrate(
		&TestUser{},
		&TestPost{},
		&TestRole{},
		&TestPermission{},
	)
	require.NoError(t, err)

	// Create test server using Fiber's Test method instead of httptest
	// Fiber v2 doesn't implement http.Handler directly
	// We'll use app.Test() for testing instead

	return &TestApp{
		App:    app,
		DB:     db,
		Server: nil, // Use app.Test() method instead of httptest.Server
	}
}

// Close closes the test app and cleans up resources
func (ta *TestApp) Close() {
	if ta.Server != nil {
		ta.Server.Close()
	}
	if ta.DB != nil {
		sqlDB, _ := ta.DB.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}
}

// Get performs a GET request
func (ta *TestApp) Get(t *testing.T, path string) *TestResponse {
	return ta.Request(t, "GET", path, nil, nil)
}

// Post performs a POST request
func (ta *TestApp) Post(t *testing.T, path string, body interface{}, headers map[string]string) *TestResponse {
	return ta.Request(t, "POST", path, body, headers)
}

// Put performs a PUT request
func (ta *TestApp) Put(t *testing.T, path string, body interface{}, headers map[string]string) *TestResponse {
	return ta.Request(t, "PUT", path, body, headers)
}

// Delete performs a DELETE request
func (ta *TestApp) Delete(t *testing.T, path string, headers map[string]string) *TestResponse {
	return ta.Request(t, "DELETE", path, nil, headers)
}

// Request performs an HTTP request
func (ta *TestApp) Request(t *testing.T, method, path string, body interface{}, headers map[string]string) *TestResponse {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, ta.Server.URL+path, reqBody)
	require.NoError(t, err)

	// Set default headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Set custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Perform request
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	return &TestResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       respBody,
	}
}

// TestResponse represents an HTTP test response
type TestResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// AssertStatus asserts the response status code
func (tr *TestResponse) AssertStatus(t *testing.T, expected int) {
	assert.Equal(t, expected, tr.StatusCode, "Expected status %d, got %d. Body: %s", expected, tr.StatusCode, string(tr.Body))
}

// AssertJSON asserts the response body matches expected JSON
func (tr *TestResponse) AssertJSON(t *testing.T, expected interface{}) {
	var actual interface{}
	err := json.Unmarshal(tr.Body, &actual)
	require.NoError(t, err, "Failed to unmarshal response body: %s", string(tr.Body))

	assert.Equal(t, expected, actual)
}

// AssertHeader asserts a response header value
func (tr *TestResponse) AssertHeader(t *testing.T, key, expected string) {
	actual := tr.Headers.Get(key)
	assert.Equal(t, expected, actual, "Expected header %s to be %s, got %s", key, expected, actual)
}

// AssertContains asserts the response body contains a substring
func (tr *TestResponse) AssertContains(t *testing.T, expected string) {
	body := string(tr.Body)
	assert.Contains(t, body, expected, "Expected response body to contain %s, got: %s", expected, body)
}

// GetJSON unmarshals the response body into the target
func (tr *TestResponse) GetJSON(t *testing.T, target interface{}) {
	err := json.Unmarshal(tr.Body, target)
	require.NoError(t, err, "Failed to unmarshal response body: %s", string(tr.Body))
}

// TestUser represents a test user model
type TestUser struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Email     string    `json:"email" gorm:"uniqueIndex"`
	Password  string    `json:"-"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TestPost represents a test post model
type TestPost struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	UserID    uint      `json:"user_id"`
	User      TestUser  `json:"user" gorm:"foreignKey:UserID"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TestRole represents a test role model
type TestRole struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"uniqueIndex"`
	Description string         `json:"description"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

// TestPermission represents a test permission model
type TestPermission struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"uniqueIndex"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TestFactory provides factory methods for creating test data
type TestFactory struct {
	DB *gorm.DB
}

// NewTestFactory creates a new test factory
func NewTestFactory(db *gorm.DB) *TestFactory {
	return &TestFactory{DB: db}
}

// CreateUser creates a test user
func (tf *TestFactory) CreateUser(overrides ...map[string]interface{}) *TestUser {
	user := &TestUser{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}

	// Apply overrides
	for _, override := range overrides {
		for key, value := range override {
			switch key {
			case "email":
				user.Email = value.(string)
			case "password":
				user.Password = value.(string)
			case "name":
				user.Name = value.(string)
			}
		}
	}

	tf.DB.Create(user)
	return user
}

// CreatePost creates a test post
func (tf *TestFactory) CreatePost(userID uint, overrides ...map[string]interface{}) *TestPost {
	post := &TestPost{
		Title:   "Test Post",
		Content: "This is a test post content",
		UserID:  userID,
	}

	// Apply overrides
	for _, override := range overrides {
		for key, value := range override {
			switch key {
			case "title":
				post.Title = value.(string)
			case "content":
				post.Content = value.(string)
			case "user_id":
				post.UserID = value.(uint)
			}
		}
	}

	tf.DB.Create(post)
	return post
}

// CreateRole creates a test role
func (tf *TestFactory) CreateRole(overrides ...map[string]interface{}) *TestRole {
	role := &TestRole{
		Name:        "test-role",
		Description: "Test role description",
	}

	// Apply overrides
	for _, override := range overrides {
		for key, value := range override {
			switch key {
			case "name":
				role.Name = value.(string)
			case "description":
				role.Description = value.(string)
			}
		}
	}

	tf.DB.Create(role)
	return role
}

// CreatePermission creates a test permission
func (tf *TestFactory) CreatePermission(overrides ...map[string]interface{}) *TestPermission {
	permission := &TestPermission{
		Name:        "test-permission",
		Description: "Test permission description",
	}

	// Apply overrides
	for _, override := range overrides {
		for key, value := range override {
			switch key {
			case "name":
				permission.Name = value.(string)
			case "description":
				permission.Description = value.(string)
			}
		}
	}

	tf.DB.Create(permission)
	return permission
}

// CleanupDatabase cleans up test database
func (tf *TestFactory) CleanupDatabase() {
	tf.DB.Exec("DELETE FROM test_users")
	tf.DB.Exec("DELETE FROM test_posts")
	tf.DB.Exec("DELETE FROM test_roles")
	tf.DB.Exec("DELETE FROM test_permissions")
}

// TestHelper provides common test utilities
type TestHelper struct {
	App     *fiber.App
	DB      *gorm.DB
	Factory *TestFactory
}

// NewTestHelper creates a new test helper
func NewTestHelper(t *testing.T) *TestHelper {
	testApp := NewTestApp(t, nil)
	factory := NewTestFactory(testApp.DB)

	return &TestHelper{
		App:     testApp.App,
		DB:      testApp.DB,
		Factory: factory,
	}
}

// SetupTestRoutes sets up common test routes
func (th *TestHelper) SetupTestRoutes() {
	// Health check
	th.App.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Test user routes
	th.App.Get("/users", func(c *fiber.Ctx) error {
		var users []TestUser
		th.DB.Find(&users)
		return c.JSON(users)
	})

	th.App.Post("/users", func(c *fiber.Ctx) error {
		var user TestUser
		if err := c.BodyParser(&user); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		th.DB.Create(&user)
		return c.Status(201).JSON(user)
	})

	th.App.Get("/users/:id", func(c *fiber.Ctx) error {
		var user TestUser
		id := c.Params("id")
		if err := th.DB.First(&user, id).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "User not found"})
		}
		return c.JSON(user)
	})
}

// Request performs an HTTP request using the Fiber app
func (th *TestHelper) Request(t *testing.T, method, path string, body interface{}, headers map[string]string) *TestResponse {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req := httptest.NewRequest(method, path, reqBody)

	// Set default headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Set custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Perform request using Fiber's Test method
	resp, err := th.App.Test(req)
	require.NoError(t, err)

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	return &TestResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       respBody,
	}
}

// Cleanup cleans up test resources
func (th *TestHelper) Cleanup() {
	th.Factory.CleanupDatabase()
}

// TestFile represents a test file
type TestFile struct {
	Name    string
	Content []byte
	Path    string
}

// CreateTestFile creates a test file
func CreateTestFile(t *testing.T, name, content string) *TestFile {
	return &TestFile{
		Name:    name,
		Content: []byte(content),
		Path:    filepath.Join(os.TempDir(), name),
	}
}

// Write writes the test file to disk
func (tf *TestFile) Write(t *testing.T) {
	err := os.WriteFile(tf.Path, tf.Content, 0644)
	require.NoError(t, err)
}

// Remove removes the test file from disk
func (tf *TestFile) Remove(t *testing.T) {
	err := os.Remove(tf.Path)
	require.NoError(t, err)
}

// TestEnvironment represents a test environment
type TestEnvironment struct {
	TempDir string
	Files   []*TestFile
}

// NewTestEnvironment creates a new test environment
func NewTestEnvironment(t *testing.T) *TestEnvironment {
	tempDir, err := os.MkdirTemp("", "mithril-test-*")
	require.NoError(t, err)

	return &TestEnvironment{
		TempDir: tempDir,
		Files:   make([]*TestFile, 0),
	}
}

// AddFile adds a file to the test environment
func (te *TestEnvironment) AddFile(t *testing.T, name, content string) *TestFile {
	file := &TestFile{
		Name:    name,
		Content: []byte(content),
		Path:    filepath.Join(te.TempDir, name),
	}
	te.Files = append(te.Files, file)
	file.Write(t)
	return file
}

// Cleanup cleans up the test environment
func (te *TestEnvironment) Cleanup(t *testing.T) {
	for _, file := range te.Files {
		file.Remove(t)
	}
	err := os.RemoveAll(te.TempDir)
	require.NoError(t, err)
}

// GetPath returns the full path for a file in the test environment
func (te *TestEnvironment) GetPath(name string) string {
	return filepath.Join(te.TempDir, name)
}

// TestLogger provides test logging utilities
type TestLogger struct {
	Messages []string
}

// NewTestLogger creates a new test logger
func NewTestLogger() *TestLogger {
	return &TestLogger{
		Messages: make([]string, 0),
	}
}

// Log logs a message
func (tl *TestLogger) Log(msg string) {
	tl.Messages = append(tl.Messages, msg)
}

// GetMessages returns all logged messages
func (tl *TestLogger) GetMessages() []string {
	return tl.Messages
}

// Clear clears all logged messages
func (tl *TestLogger) Clear() {
	tl.Messages = make([]string, 0)
}

// AssertLogged asserts that a message was logged
func (tl *TestLogger) AssertLogged(t *testing.T, expected string) {
	for _, msg := range tl.Messages {
		if msg == expected {
			return
		}
	}
	t.Errorf("Expected message '%s' not found in logs. Got: %v", expected, tl.Messages)
}

// TestTimeout provides timeout utilities for tests
type TestTimeout struct {
	Duration time.Duration
}

// NewTestTimeout creates a new test timeout
func NewTestTimeout(duration time.Duration) *TestTimeout {
	return &TestTimeout{Duration: duration}
}

// WithTimeout runs a function with a timeout
func (tt *TestTimeout) WithTimeout(t *testing.T, fn func()) {
	done := make(chan bool, 1)
	go func() {
		fn()
		done <- true
	}()

	select {
	case <-done:
		return
	case <-time.After(tt.Duration):
		t.Fatalf("Test timed out after %v", tt.Duration)
	}
}

// TestDataProvider provides test data utilities
type TestDataProvider struct {
	Data map[string]interface{}
}

// NewTestDataProvider creates a new test data provider
func NewTestDataProvider() *TestDataProvider {
	return &TestDataProvider{
		Data: make(map[string]interface{}),
	}
}

// Set sets a test data value
func (tdp *TestDataProvider) Set(key string, value interface{}) {
	tdp.Data[key] = value
}

// Get gets a test data value
func (tdp *TestDataProvider) Get(key string) interface{} {
	return tdp.Data[key]
}

// GetString gets a test data value as string
func (tdp *TestDataProvider) GetString(key string) string {
	if value, ok := tdp.Data[key].(string); ok {
		return value
	}
	return ""
}

// GetInt gets a test data value as int
func (tdp *TestDataProvider) GetInt(key string) int {
	if value, ok := tdp.Data[key].(int); ok {
		return value
	}
	return 0
}

// GetBool gets a test data value as bool
func (tdp *TestDataProvider) GetBool(key string) bool {
	if value, ok := tdp.Data[key].(bool); ok {
		return value
	}
	return false
}

// Clear clears all test data
func (tdp *TestDataProvider) Clear() {
	tdp.Data = make(map[string]interface{})
}

// TestAssertion provides custom assertion utilities
type TestAssertion struct {
	t *testing.T
}

// NewTestAssertion creates a new test assertion
func NewTestAssertion(t *testing.T) *TestAssertion {
	return &TestAssertion{t: t}
}

// AssertNoError asserts that there is no error
func (ta *TestAssertion) AssertNoError(err error, msg ...string) {
	if err != nil {
		if len(msg) > 0 {
			ta.t.Fatalf("%s: %v", msg[0], err)
		} else {
			ta.t.Fatalf("Unexpected error: %v", err)
		}
	}
}

// AssertError asserts that there is an error
func (ta *TestAssertion) AssertError(err error, msg ...string) {
	if err == nil {
		if len(msg) > 0 {
			ta.t.Fatalf("Expected error but got none: %s", msg[0])
		} else {
			ta.t.Fatalf("Expected error but got none")
		}
	}
}

// AssertNotEmpty asserts that a value is not empty
func (ta *TestAssertion) AssertNotEmpty(value interface{}, msg ...string) {
	switch v := value.(type) {
	case string:
		if v == "" {
			ta.t.Fatalf("Expected non-empty string, got empty")
		}
	case []interface{}:
		if len(v) == 0 {
			ta.t.Fatalf("Expected non-empty slice, got empty")
		}
	case map[string]interface{}:
		if len(v) == 0 {
			ta.t.Fatalf("Expected non-empty map, got empty")
		}
	default:
		if value == nil {
			ta.t.Fatalf("Expected non-nil value, got nil")
		}
	}
}

// AssertEmpty asserts that a value is empty
func (ta *TestAssertion) AssertEmpty(value interface{}, msg ...string) {
	switch v := value.(type) {
	case string:
		if v != "" {
			ta.t.Fatalf("Expected empty string, got: %s", v)
		}
	case []interface{}:
		if len(v) != 0 {
			ta.t.Fatalf("Expected empty slice, got: %v", v)
		}
	case map[string]interface{}:
		if len(v) != 0 {
			ta.t.Fatalf("Expected empty map, got: %v", v)
		}
	default:
		if value != nil {
			ta.t.Fatalf("Expected nil value, got: %v", value)
		}
	}
}
