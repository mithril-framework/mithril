package config

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigManager_NewManager(t *testing.T) {
	tests := []struct {
		name     string
		envFile  string
		wantErr  bool
		setupEnv func()
	}{
		{
			name:    "valid env file",
			envFile: ".env.test",
			wantErr: false,
			setupEnv: func() {
				_ = os.WriteFile(".env.test", []byte("APP_NAME=Test App\nAPP_VERSION=1.0.0\nAPP_ENV=testing\nAPP_DEBUG=true\nAPP_HOST=localhost\nAPP_PORT=4000\nAPP_TIMEZONE=UTC\nAPP_SECRET_KEY=test-secret-key-32-characters-long\nSESSION_SECRET=test-session-secret-32-characters-long\nRATE_LIMIT_WINDOW=1m\nDB_DRIVER=sqlite\nDB_HOST=localhost\nDB_PORT=5432\nDB_NAME=test\nDB_USER=test\nDB_PASSWORD=test\nJWT_SECRET=test-jwt-secret-32-characters-long\nMAIL_DRIVER=smtp\nMAIL_HOST=localhost\nMAIL_PORT=587\nMAIL_FROM_ADDRESS=test@example.com\nQUEUE_DRIVER=memory\nSTORAGE_DRIVER=local\nCORS_ENABLED=true\nRATE_LIMIT_ENABLED=true\nSESSION_ENABLED=true"), 0644)
			},
		},
		{
			name:    "non-existent env file",
			envFile: ".env.nonexistent",
			wantErr: false, // Should not error if file doesn't exist
			setupEnv: func() {
				// Set required env vars directly
				os.Setenv("APP_NAME", "Test App")
				os.Setenv("APP_VERSION", "1.0.0")
				os.Setenv("APP_ENV", "testing")
				os.Setenv("APP_DEBUG", "true")
				os.Setenv("APP_HOST", "localhost")
				os.Setenv("APP_PORT", "4000")
				os.Setenv("APP_TIMEZONE", "UTC")
				os.Setenv("APP_SECRET_KEY", "test-secret-key-32-characters-long")
				os.Setenv("SESSION_SECRET", "test-session-secret-32-characters-long")
				os.Setenv("RATE_LIMIT_WINDOW", "1m")
				os.Setenv("DB_DRIVER", "sqlite")
				os.Setenv("DB_HOST", "localhost")
				os.Setenv("DB_PORT", "5432")
				os.Setenv("DB_NAME", "test")
				os.Setenv("DB_USER", "test")
				os.Setenv("DB_PASSWORD", "test")
				os.Setenv("JWT_SECRET", "test-jwt-secret-32-characters-long")
				os.Setenv("MAIL_DRIVER", "smtp")
				os.Setenv("MAIL_HOST", "localhost")
				os.Setenv("MAIL_PORT", "587")
				os.Setenv("MAIL_FROM_ADDRESS", "test@example.com")
				os.Setenv("QUEUE_DRIVER", "memory")
				os.Setenv("STORAGE_DRIVER", "local")
				os.Setenv("CORS_ENABLED", "true")
				os.Setenv("RATE_LIMIT_ENABLED", "true")
				os.Setenv("SESSION_ENABLED", "true")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment
			cleanupEnv(t)
			tt.setupEnv()

			manager, err := NewManager(tt.envFile)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, manager)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, manager)
			}

			// Clean up
			os.Remove(".env.test")
		})
	}
}

func TestConfigManager_RegisterConfig(t *testing.T) {
	cleanupEnv(t)
	setupTestEnv(t)

	manager, err := NewManager(".env.test")
	require.NoError(t, err)

	// Test that the manager was created successfully
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.GetAppConfig())
	assert.NotNil(t, manager.GetDatabaseConfig())
}

func TestConfigManager_GetConfig(t *testing.T) {
	cleanupEnv(t)
	setupTestEnv(t)

	manager, err := NewManager(".env.test")
	require.NoError(t, err)

	tests := []struct {
		name       string
		configName string
		wantFound  bool
	}{
		{
			name:       "existing config",
			configName: "app",
			wantFound:  true,
		},
		{
			name:       "non-existing config",
			configName: "nonexistent",
			wantFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, found := manager.GetConfig(tt.configName)

			if tt.wantFound {
				assert.True(t, found)
				assert.NotNil(t, config)
			} else {
				assert.False(t, found)
				assert.Nil(t, config)
			}
		})
	}
}

func TestConfigManager_parseDuration(t *testing.T) {
	manager := &ConfigManager{}

	tests := []struct {
		name    string
		value   string
		want    time.Duration
		wantErr bool
	}{
		{
			name:    "standard duration - minutes",
			value:   "15m",
			want:    15 * time.Minute,
			wantErr: false,
		},
		{
			name:    "standard duration - hours",
			value:   "2h",
			want:    2 * time.Hour,
			wantErr: false,
		},
		{
			name:    "standard duration - seconds",
			value:   "30s",
			want:    30 * time.Second,
			wantErr: false,
		},
		{
			name:    "days duration",
			value:   "7d",
			want:    7 * 24 * time.Hour,
			wantErr: false,
		},
		{
			name:    "plain number as seconds",
			value:   "3600",
			want:    3600 * time.Second,
			wantErr: false,
		},
		{
			name:    "invalid duration",
			value:   "invalid",
			want:    0,
			wantErr: true,
		},
		{
			name:    "empty duration",
			value:   "",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := manager.parseDuration(tt.value)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestConfigManager_setFieldValue(t *testing.T) {
	manager := &ConfigManager{}

	tests := []struct {
		name    string
		field   interface{}
		value   string
		wantErr bool
	}{
		{
			name:    "string field",
			field:   "",
			value:   "test",
			wantErr: false,
		},
		{
			name:    "int field",
			field:   0,
			value:   "42",
			wantErr: false,
		},
		{
			name:    "bool field",
			field:   false,
			value:   "true",
			wantErr: false,
		},
		{
			name:    "duration field",
			field:   time.Duration(0),
			value:   "15m",
			wantErr: false,
		},
		{
			name:    "invalid int",
			field:   0,
			value:   "not-a-number",
			wantErr: true,
		},
		{
			name:    "invalid bool",
			field:   false,
			value:   "not-a-bool",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldValue := reflect.ValueOf(&tt.field).Elem()
			err := manager.setFieldValue(fieldValue, tt.value)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigManager_validateConfig(t *testing.T) {
	manager := &ConfigManager{}

	tests := []struct {
		name    string
		config  interface{}
		wantErr bool
	}{
		{
			name: "valid config - no required fields",
			config: struct {
				Name string `env:"NAME"`
			}{},
			wantErr: false,
		},
		{
			name: "valid config - required field set",
			config: struct {
				Name string `env:"NAME" required:"true"`
			}{Name: "test"},
			wantErr: false,
		},
		{
			name: "invalid config - required field not set",
			config: struct {
				Name string `env:"NAME" required:"true"`
			}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.validateConfig(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigManager_Load(t *testing.T) {
	cleanupEnv(t)

	// Create test .env file
	envContent := `APP_NAME=Test App
APP_VERSION=1.0.0
APP_ENV=testing
APP_DEBUG=true
APP_HOST=localhost
APP_PORT=4000
APP_TIMEZONE=UTC
APP_SECRET_KEY=test-secret-key-32-characters-long
SESSION_SECRET=test-session-secret-32-characters-long
RATE_LIMIT_WINDOW=1m
DB_DRIVER=sqlite
DB_HOST=localhost
DB_PORT=5432
DB_NAME=test
DB_USER=test
DB_PASSWORD=test
JWT_SECRET=test-jwt-secret-32-characters-long
MAIL_DRIVER=smtp
MAIL_HOST=localhost
MAIL_PORT=587
MAIL_FROM_ADDRESS=test@example.com
QUEUE_DRIVER=memory
STORAGE_DRIVER=local
CORS_ENABLED=true
RATE_LIMIT_ENABLED=true
SESSION_ENABLED=true`

	err := os.WriteFile(".env.test", []byte(envContent), 0644)
	require.NoError(t, err)
	defer os.Remove(".env.test")

	manager, err := NewManager(".env.test")
	require.NoError(t, err)
	require.NotNil(t, manager)

	// Test that environment variables are loaded
	assert.Equal(t, "Test App", os.Getenv("APP_NAME"))
	assert.Equal(t, "testing", os.Getenv("APP_ENV"))
	assert.Equal(t, "true", os.Getenv("APP_DEBUG"))
}

func TestConfigManager_Load_NonExistentFile(t *testing.T) {
	cleanupEnv(t)

	manager, err := NewManager(".env.nonexistent")
	require.NoError(t, err)
	assert.NotNil(t, manager)
}

func TestConfigManager_Load_InvalidFile(t *testing.T) {
	cleanupEnv(t)

	// Create invalid .env file
	err := os.WriteFile(".env.invalid", []byte("invalid content with spaces around ="), 0644)
	require.NoError(t, err)
	defer os.Remove(".env.invalid")

	manager, err := NewManager(".env.invalid")
	require.NoError(t, err)
	assert.NotNil(t, manager)
}

// Helper functions

func cleanupEnv(t *testing.T) {
	envVars := []string{
		"APP_NAME", "APP_VERSION", "APP_ENV", "APP_DEBUG", "APP_HOST", "APP_PORT", "APP_TIMEZONE",
		"APP_SECRET_KEY", "SESSION_SECRET", "RATE_LIMIT_WINDOW",
		"DB_DRIVER", "DB_HOST", "DB_PORT", "DB_NAME", "DB_USER", "DB_PASSWORD",
		"JWT_SECRET", "MAIL_DRIVER", "MAIL_HOST", "MAIL_PORT", "MAIL_FROM_ADDRESS",
		"QUEUE_DRIVER", "STORAGE_DRIVER", "CORS_ENABLED", "RATE_LIMIT_ENABLED", "SESSION_ENABLED",
	}

	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}
}

func setupTestEnv(t *testing.T) {
	envContent := `APP_NAME=Test App
APP_VERSION=1.0.0
APP_ENV=testing
APP_DEBUG=true
APP_HOST=localhost
APP_PORT=4000
APP_TIMEZONE=UTC
APP_SECRET_KEY=test-secret-key-32-characters-long
SESSION_SECRET=test-session-secret-32-characters-long
RATE_LIMIT_WINDOW=1m
DB_DRIVER=sqlite
DB_HOST=localhost
DB_PORT=5432
DB_NAME=test
DB_USER=test
DB_PASSWORD=test
JWT_SECRET=test-jwt-secret-32-characters-long
MAIL_DRIVER=smtp
MAIL_HOST=localhost
MAIL_PORT=587
MAIL_FROM_ADDRESS=test@example.com
QUEUE_DRIVER=memory
STORAGE_DRIVER=local
CORS_ENABLED=true
RATE_LIMIT_ENABLED=true
SESSION_ENABLED=true`

	err := os.WriteFile(".env.test", []byte(envContent), 0644)
	require.NoError(t, err)
}
