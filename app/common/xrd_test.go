package common

import (
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
