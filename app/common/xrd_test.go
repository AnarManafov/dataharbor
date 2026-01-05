package common

import (
	"errors"
	"fmt"
	"testing"

	"github.com/AnarManafov/dataharbor/app/config"
	"github.com/stretchr/testify/assert"
)

func TestGetXRDNativeClient(t *testing.T) {
	// Set up test configuration
	testConfig := &config.Config{
		Env: "test",
		Server: config.ServerConfig{
			Address: ":8080",
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
			Console: config.ConsoleConfig{
				Enabled: true,
				Level:   "info",
				Format:  "text",
			},
			File: config.FileConfig{
				Enabled: false,
			},
		},
		XRD: config.XRDConfig{
			Host:       "localhost",
			Port:       1094,
			InitialDir: "/tmp/",
			User:       "testuser",
		},
		Auth: config.AuthConfig{
			Enabled: false,
			SkipAuthPaths: []string{
				"/health",
			},
		},
		Frontend: config.FrontendConfig{
			URL:        "http://localhost:5173",
			AssetPaths: []string{},
			DistDir:    "dist",
		},
	}

	// Set the test config
	originalConfig := config.GetConfig()
	config.SetConfig(testConfig)
	defer config.SetConfig(originalConfig) // Restore original config after test

	// Test that the native client can be created
	t.Run("client creation", func(t *testing.T) {
		client := GetXRDNativeClient()
		assert.NotNil(t, client, "Native client should not be nil")
		assert.Equal(t, "localhost:1094", client.address, "Client address should match config")
		assert.Equal(t, "testuser", client.username, "Client username should match config")
	})

	// Test that GetXRDClient returns the same instance
	t.Run("client compatibility", func(t *testing.T) {
		client1 := GetXRDClient()
		client2 := GetXRDNativeClient()
		assert.Equal(t, client1, client2, "GetXRDClient should return same instance as GetXRDNativeClient")
	})
}

func TestXRootDAuthError_Error(t *testing.T) {
	err := &XRootDAuthError{
		message: "authentication failed",
		cause:   nil,
	}

	assert.Equal(t, "authentication failed", err.Error())
}

func TestXRootDAuthError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &XRootDAuthError{
		message: "authentication failed",
		cause:   cause,
	}

	unwrapped := err.Unwrap()
	assert.Equal(t, cause, unwrapped)
}

func TestXRootDAuthError_UnwrapNil(t *testing.T) {
	err := &XRootDAuthError{
		message: "authentication failed",
		cause:   nil,
	}

	unwrapped := err.Unwrap()
	assert.Nil(t, unwrapped)
}

func TestIsAuthError_True(t *testing.T) {
	err := &XRootDAuthError{
		message: "auth error",
		cause:   nil,
	}

	assert.True(t, IsAuthError(err), "IsAuthError should return true for XRootDAuthError")
}

func TestIsAuthError_False(t *testing.T) {
	err := errors.New("regular error")

	assert.False(t, IsAuthError(err), "IsAuthError should return false for regular error")
}

func TestIsAuthError_WrappedError(t *testing.T) {
	authErr := &XRootDAuthError{
		message: "auth error",
		cause:   nil,
	}
	wrappedErr := fmt.Errorf("wrapped: %w", authErr)

	assert.True(t, IsAuthError(wrappedErr), "IsAuthError should return true for wrapped XRootDAuthError")
}

func TestIsAuthError_Nil(t *testing.T) {
	assert.False(t, IsAuthError(nil), "IsAuthError should return false for nil error")
}

func TestIsAuthorizationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "regular error",
			err:      errors.New("some random error"),
			expected: false,
		},
		{
			name:     "authorization error",
			err:      errors.New("authorization failed"),
			expected: true,
		},
		{
			name:     "unauthorized error",
			err:      errors.New("user is Unauthorized"),
			expected: true,
		},
		{
			name:     "authentication error",
			err:      errors.New("Authentication required"),
			expected: true,
		},
		{
			name:     "permission denied",
			err:      errors.New("Permission Denied for user"),
			expected: true,
		},
		{
			name:     "access denied",
			err:      errors.New("access denied"),
			expected: true,
		},
		{
			name:     "not authorized",
			err:      errors.New("User is not authorized"),
			expected: true,
		},
		{
			name:     "token error",
			err:      errors.New("Invalid token"),
			expected: true,
		},
		{
			name:     "credentials error",
			err:      errors.New("Invalid credentials"),
			expected: true,
		},
		{
			name:     "audience error",
			err:      errors.New("invalid aud claim"),
			expected: true,
		},
		{
			name:     "claim verification error",
			err:      errors.New("claim verification failed"),
			expected: true,
		},
		{
			name:     "scitokens error",
			err:      errors.New("scitokens validation failed"),
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isAuthorizationError(tc.err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestXRDClient_SetUserToken(t *testing.T) {
	// Set up test configuration
	testConfig := &config.Config{
		XRD: config.XRDConfig{
			Host:       "localhost",
			Port:       1094,
			InitialDir: "/tmp/",
			User:       "testuser",
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
			Console: config.ConsoleConfig{
				Enabled: true,
				Level:   "info",
				Format:  "text",
			},
		},
	}

	originalConfig := config.GetConfig()
	config.SetConfig(testConfig)
	defer config.SetConfig(originalConfig)

	// Create a client using the config set above
	client := NewXRDClient()

	// SetUserToken should not panic and should be a no-op
	assert.NotPanics(t, func() {
		client.SetUserToken("some-token")
	}, "SetUserToken should not panic")

	// Calling with empty token should also be fine
	assert.NotPanics(t, func() {
		client.SetUserToken("")
	}, "SetUserToken with empty token should not panic")
}

func TestNewXRDClient_DefaultUser(t *testing.T) {
	// Test that default user is "dataharbor" when not specified
	testConfig := &config.Config{
		XRD: config.XRDConfig{
			Host:       "localhost",
			Port:       1094,
			InitialDir: "/tmp/",
			User:       "", // Empty user to test default
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
			Console: config.ConsoleConfig{
				Enabled: true,
				Level:   "info",
				Format:  "text",
			},
		},
	}

	originalConfig := config.GetConfig()
	config.SetConfig(testConfig)
	defer config.SetConfig(originalConfig)

	client := NewXRDClient()
	assert.Equal(t, "dataharbor", client.username, "Default username should be 'dataharbor'")
}

func TestNewXRDClient_WithZTN(t *testing.T) {
	// Test client creation with ZTN enabled
	testConfig := &config.Config{
		XRD: config.XRDConfig{
			Host:       "localhost",
			Port:       1094,
			InitialDir: "/tmp/",
			User:       "testuser",
			EnableZTN:  true,
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
			Console: config.ConsoleConfig{
				Enabled: true,
				Level:   "info",
				Format:  "text",
			},
		},
	}

	originalConfig := config.GetConfig()
	config.SetConfig(testConfig)
	defer config.SetConfig(originalConfig)

	client := NewXRDClient()
	assert.True(t, client.enableZTN, "enableZTN should be true when configured")
	assert.Equal(t, "localhost:1094", client.address)
}
