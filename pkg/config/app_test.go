package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppConfig_NewAppConfig(t *testing.T) {
	config := NewAppConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "Mithril App", config.Name)
	assert.Equal(t, "1.0.0", config.Version)
	assert.Equal(t, "development", config.Environment)
	assert.Equal(t, false, config.Debug)
	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, "4000", config.Port)
	assert.Equal(t, "UTC", config.Timezone)
}

func TestAppConfig_GetAddress(t *testing.T) {
	config := &AppConfig{
		Host: "localhost",
		Port: "8080",
	}

	address := config.GetAddress()
	assert.Equal(t, "localhost:8080", address)
}

func TestAppConfig_IsDevelopment(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		expected    bool
	}{
		{"development", "development", true},
		{"production", "production", false},
		{"staging", "staging", false},
		{"testing", "testing", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &AppConfig{Environment: tt.environment}
			assert.Equal(t, tt.expected, config.IsDevelopment())
		})
	}
}

func TestAppConfig_IsProduction(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		expected    bool
	}{
		{"development", "development", false},
		{"production", "production", true},
		{"staging", "staging", false},
		{"testing", "testing", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &AppConfig{Environment: tt.environment}
			assert.Equal(t, tt.expected, config.IsProduction())
		})
	}
}

func TestAppConfig_IsStaging(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		expected    bool
	}{
		{"development", "development", false},
		{"production", "production", false},
		{"staging", "staging", true},
		{"testing", "testing", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &AppConfig{Environment: tt.environment}
			assert.Equal(t, tt.expected, config.IsStaging())
		})
	}
}

func TestAppConfig_IsTesting(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		expected    bool
	}{
		{"development", "development", false},
		{"production", "production", false},
		{"staging", "staging", false},
		{"testing", "testing", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &AppConfig{Environment: tt.environment}
			assert.Equal(t, tt.expected, config.IsTesting())
		})
	}
}

func TestAppConfig_LoadFromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("APP_NAME", "Test App")
	os.Setenv("APP_VERSION", "2.0.0")
	os.Setenv("APP_ENV", "testing")
	os.Setenv("APP_DEBUG", "true")
	os.Setenv("APP_HOST", "0.0.0.0")
	os.Setenv("APP_PORT", "8080")
	os.Setenv("APP_TIMEZONE", "America/New_York")
	os.Setenv("APP_SECRET_KEY", "test-secret-key-32-characters-long")
	os.Setenv("SESSION_SECRET", "test-session-secret-32-characters-long")
	os.Setenv("RATE_LIMIT_WINDOW", "2m")

	defer func() {
		os.Unsetenv("APP_NAME")
		os.Unsetenv("APP_VERSION")
		os.Unsetenv("APP_ENV")
		os.Unsetenv("APP_DEBUG")
		os.Unsetenv("APP_HOST")
		os.Unsetenv("APP_PORT")
		os.Unsetenv("APP_TIMEZONE")
		os.Unsetenv("APP_SECRET_KEY")
		os.Unsetenv("SESSION_SECRET")
		os.Unsetenv("RATE_LIMIT_WINDOW")
	}()

	manager, err := NewManagerFromEnv()
	require.NoError(t, err)

	appConfig := manager.GetAppConfig()

	// Verify values are loaded from environment
	assert.Equal(t, "Test App", appConfig.Name)
	assert.Equal(t, "2.0.0", appConfig.Version)
	assert.Equal(t, "testing", appConfig.Environment)
	assert.Equal(t, true, appConfig.Debug)
	assert.Equal(t, "0.0.0.0", appConfig.Host)
	assert.Equal(t, "8080", appConfig.Port)
	assert.Equal(t, "America/New_York", appConfig.Timezone)
	assert.Equal(t, "test-secret-key-32-characters-long", appConfig.SecretKey)
	assert.Equal(t, "test-session-secret-32-characters-long", appConfig.SessionSecret)
	assert.Equal(t, "2m", appConfig.RateLimitWindow.String())
}

func TestAppConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  *AppConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &AppConfig{
				Name:          "Test App",
				Version:       "1.0.0",
				Environment:   "development",
				Debug:         false,
				Host:          "localhost",
				Port:          "4000",
				Timezone:      "UTC",
				SecretKey:     "test-secret-key-32-characters-long",
				SessionSecret: "test-session-secret-32-characters-long",
			},
			wantErr: false,
		},
		{
			name: "missing required fields",
			config: &AppConfig{
				Name:          "",
				Version:       "1.0.0",
				Environment:   "",
				Debug:         false,
				Host:          "",
				Port:          "0",
				Timezone:      "UTC",
				SecretKey:     "",
				SessionSecret: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &ConfigManager{}
			err := manager.validateConfig(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAppConfig_DefaultValues(t *testing.T) {
	// Clear environment variables
	envVars := []string{
		"APP_NAME", "APP_VERSION", "APP_ENV", "APP_DEBUG", "APP_HOST", "APP_PORT",
		"APP_TIMEZONE", "APP_SECRET_KEY", "SESSION_SECRET", "RATE_LIMIT_WINDOW",
	}
	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}

	manager, err := NewManagerFromEnv()
	require.NoError(t, err)

	appConfig := manager.GetAppConfig()

	// Verify default values
	assert.Equal(t, "Mithril App", appConfig.Name)
	assert.Equal(t, "1.0.0", appConfig.Version)
	assert.Equal(t, "development", appConfig.Environment)
	assert.Equal(t, false, appConfig.Debug)
	assert.Equal(t, "localhost", appConfig.Host)
	assert.Equal(t, "4000", appConfig.Port)
	assert.Equal(t, "UTC", appConfig.Timezone)
}

func TestAppConfig_EnvironmentDetection(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		dev         bool
		prod        bool
		staging     bool
		testing     bool
	}{
		{
			name:        "development",
			environment: "development",
			dev:         true,
			prod:        false,
			staging:     false,
			testing:     false,
		},
		{
			name:        "production",
			environment: "production",
			dev:         false,
			prod:        true,
			staging:     false,
			testing:     false,
		},
		{
			name:        "staging",
			environment: "staging",
			dev:         false,
			prod:        false,
			staging:     true,
			testing:     false,
		},
		{
			name:        "testing",
			environment: "testing",
			dev:         false,
			prod:        false,
			staging:     false,
			testing:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &AppConfig{Environment: tt.environment}
			assert.Equal(t, tt.dev, config.IsDevelopment())
			assert.Equal(t, tt.prod, config.IsProduction())
			assert.Equal(t, tt.staging, config.IsStaging())
			assert.Equal(t, tt.testing, config.IsTesting())
		})
	}
}
