package controller

import (
	"testing"

	"github.com/AnarManafov/dataharbor/app/common"
	"github.com/AnarManafov/dataharbor/app/config"
)

// TestMain sets up testing environment for native XRootD client
func TestMain(m *testing.M) {
	// Initialize the logger and the configuration
	common.InitLogger()
	config.InitCmd()

	// Create test config and set it as the global config
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
	config.SetConfig(testConfig)

	// Run the tests
	m.Run()
}
