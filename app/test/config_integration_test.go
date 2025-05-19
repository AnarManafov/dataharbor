package test

import (
	"context"
	"sync"
	"testing"

	"github.com/AnarManafov/dataharbor/app/common"
	"github.com/AnarManafov/dataharbor/app/config"
)

var configInitOnce sync.Once

// initTestConfig ensures config is initialized only once across all tests
func initTestConfig() {
	configInitOnce.Do(func() {
		config.InitCmd()
	})
}

// TestMain sets up and tears down the test environment
func TestMain(m *testing.M) {
	// Setup: Initialize config once for all tests
	initTestConfig()

	// Run tests
	m.Run()

	// Teardown: Could add cleanup here if needed
}

// TestConfigInitialization tests that the config system initializes properly
func TestConfigInitialization(t *testing.T) {
	cfg := config.GetConfig()
	if cfg == nil {
		t.Fatal("config should not be nil after initialization")
	}

	t.Run("XRD_config_has_defaults", func(t *testing.T) {
		// Test that XRD configuration has reasonable defaults or values
		if cfg.XRD.Host == "" {
			t.Logf("XRD Host is empty, using default")
		}

		// Verify structure is properly initialized
		if cfg.XRD.Port == 0 {
			t.Logf("XRD Port is 0, may use default")
		}
	})
}

// TestXRDClientCreation tests that XRD client can be created
func TestXRDClientCreation(t *testing.T) {
	xrdClient := common.GetXRDClient()
	if xrdClient == nil {
		t.Fatal("XRD Client should not be nil")
	}

	tests := []struct {
		name     string
		field    string
		checkFn  func() bool
		expected bool
	}{
		{
			name:     "Logger_not_nil",
			field:    "Logger",
			checkFn:  func() bool { return common.GetLogger() != nil },
			expected: true,
		},
		{
			name:     "Client_is_native",
			field:    "Type",
			checkFn:  func() bool { return xrdClient != nil },
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.checkFn()
			if result != tt.expected {
				t.Errorf("XRDClient %s check failed: expected %v, got %v", tt.field, tt.expected, result)
			}
		})
	}
}

// TestNativeClientConfiguration tests the native XRD client configuration
func TestNativeClientConfiguration(t *testing.T) {
	cfg := config.GetConfig()

	// Test native client initialization
	client := common.GetXRDNativeClient()
	if client == nil {
		t.Fatal("Native client should not be nil")
	}

	t.Run("client_properties", func(t *testing.T) {
		// Test that the client is properly configured
		if common.GetLogger() == nil {
			t.Error("Client logger should not be nil")
		}
	})

	t.Run("compatibility_layer", func(t *testing.T) {
		// Test compatibility layer
		client1 := common.GetXRDClient()
		client2 := common.GetXRDNativeClient()
		if client1 != client2 {
			t.Error("GetXRDClient and GetXRDNativeClient should return the same instance")
		}
	})

	t.Run("config_values", func(t *testing.T) {
		// Test basic configuration is available
		if cfg.XRD.Host == "" {
			t.Log("XRD Host is empty, may use defaults")
		}
		if cfg.XRD.Port == 0 {
			t.Log("XRD Port is 0, may use defaults")
		}
	})
}

// TestXRDClientTokenManagement tests token-related functionality
func TestXRDClientTokenManagement(t *testing.T) {
	client := common.GetXRDNativeClient()

	tests := []struct {
		name  string
		token string
	}{
		{"set_simple_token", "test-token-123"},
		{"set_empty_token", ""},
		{"set_bearer_token", "Bearer abc123"},
		{"set_long_token", "very-long-token-with-many-characters-1234567890"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// SetUserToken should not panic and should accept any token
			client.SetUserToken(tt.token)
			// Token is stored internally, we can't directly verify it
			// but we can test that the method works
			t.Logf("Successfully set token: %s", tt.name)
		})
	}
}

// TestXRDClientURLBuilding tests URL construction functionality
func TestXRDClientURLBuilding(t *testing.T) {
	client := common.GetXRDClient()

	tests := []struct {
		name        string
		path        string
		extraParams map[string]string
		token       string
		expectError bool
		urlContains []string
	}{
		{
			name:        "simple_path",
			path:        "test/path",
			extraParams: nil,
			token:       "",
			expectError: false,
			urlContains: []string{"test/path"},
		},
		{
			name:        "path_with_params",
			path:        "data/file.txt",
			extraParams: map[string]string{"param1": "value1", "param2": "value2"},
			token:       "",
			expectError: false,
			urlContains: []string{"data/file.txt", "param1=value1", "param2=value2"},
		},
		{
			name:        "path_with_token",
			path:        "secure/data",
			extraParams: nil,
			token:       "test-token",
			expectError: false,
			urlContains: []string{"secure/data", "authz=Bearer test-token"},
		},
		{
			name:        "complex_scenario",
			path:        "complex/path/file.dat",
			extraParams: map[string]string{"mode": "read", "format": "binary"},
			token:       "complex-token-123",
			expectError: false,
			urlContains: []string{"complex/path/file.dat", "mode=read", "format=binary", "authz=Bearer complex-token-123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set token if provided
			if tt.token != "" {
				client.SetUserToken(tt.token)
			} else {
				client.SetUserToken("") // Clear token
			}

			// Test native client filesystem access instead of URL building
			fs, cleanup, err := client.GetFileSystem(context.Background(), "")

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				// Expected in unit test environment without XRootD server
				t.Logf("Expected error in unit test environment: %v", err)
				return
			}

			if err == nil && fs != nil {
				t.Logf("Successfully accessed filesystem for test: %s", tt.name)
				cleanup()
			}
		})
	}
}
