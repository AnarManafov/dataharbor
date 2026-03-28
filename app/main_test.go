package main

import (
	"os"
	"sync"
	"testing"

	"github.com/AnarManafov/dataharbor/app/common"
	"github.com/AnarManafov/dataharbor/app/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock for gin.Engine
type MockEngine struct {
	mock.Mock
}

func (m *MockEngine) Run(addr ...string) error {
	args := m.Called(addr)
	return args.Error(0)
}

func TestInitialize(t *testing.T) {
	// Mock the initialization functions if necessary
	initialize()

	// Add assertions to verify initialization
	assert.NotNil(t, common.Logger)
}

func TestStartServer_DebugMode(t *testing.T) {
	// Set up a test configuration with debug enabled
	testConfig := &config.Config{
		Server: config.ServerConfig{
			Address: ":20222",
			Debug:   true,
		},
	}
	config.SetConfig(testConfig)

	// Mock gin router
	gin.SetMode(gin.TestMode)

	// Create a stop channel
	stop := make(chan struct{})

	// Call the startServer function
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		startServer(stop)
	}()

	// Add assertions to verify server start
	assert.Equal(t, gin.TestMode, gin.Mode())

	// Send stop signal and wait for the goroutine to finish
	close(stop)
	wg.Wait()
}

func TestStartServer_ReleaseMode(t *testing.T) {
	// Set up a test configuration with debug disabled
	testConfig := &config.Config{
		Server: config.ServerConfig{
			Address: ":20222",
			Debug:   false,
		},
	}
	config.SetConfig(testConfig)

	// Mock gin router
	gin.SetMode(gin.ReleaseMode)

	// Create a stop channel
	stop := make(chan struct{})

	// Call the startServer function
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		startServer(stop)
	}()

	// Add assertions to verify server start
	assert.Equal(t, gin.ReleaseMode, gin.Mode())

	// Send stop signal and wait for the goroutine to finish
	close(stop)
	wg.Wait()
}

func TestStartServer_SSLEnabled(t *testing.T) {
	// Set up a test configuration with SSL enabled
	testConfig := &config.Config{
		Server: config.ServerConfig{
			Address: ":20224",
			Debug:   true,
			SSL: config.SSLConfig{
				Enabled:  true,
				CertFile: "/tmp/nonexistent-cert.pem",
				KeyFile:  "/tmp/nonexistent-key.pem",
			},
		},
	}
	config.SetConfig(testConfig)

	gin.SetMode(gin.TestMode)

	// Create a stop channel
	stop := make(chan struct{})

	// This will fail to start due to missing cert files, but it exercises the SSL path
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			// Recover from any panic
			_ = recover()
		}()
		startServer(stop)
	}()

	// Signal stop and wait for the goroutine to finish
	close(stop)
	wg.Wait()
}

func TestStartServer_DefaultAddress(t *testing.T) {
	// Set up a test configuration with empty address (should default to :8080)
	testConfig := &config.Config{
		Server: config.ServerConfig{
			Address: "",
			Debug:   true,
		},
	}
	config.SetConfig(testConfig)

	gin.SetMode(gin.TestMode)

	// Create a stop channel
	stop := make(chan struct{})

	// Start server in goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		startServer(stop)
	}()

	// Signal stop and wait for the goroutine to finish
	close(stop)
	wg.Wait()
}

func TestInitialize_WithAuthEnabled(t *testing.T) {
	// Create a temp config file
	tempConfig := `
env: test
server:
  address: ":20228"
  debug: true
xrd:
  host: localhost
  port: 1094
  initial_dir: /tmp/
auth:
  enabled: true
  oidc:
    issuer: https://test.example.com
    client_id: test-client
    session_secret: test-secret-12345678901234567890
logging:
  level: info
  console:
    enabled: true
    format: text
`
	// Write to temp file
	tempFile := "/tmp/test_init_config.yaml"
	err := writeFile(tempFile, tempConfig)
	if err != nil {
		t.Skip("Could not create temp config file")
	}

	// Set config file
	originalConfigFile := config.ConfigFile
	config.ConfigFile = tempFile
	defer func() {
		config.ConfigFile = originalConfigFile
	}()

	// Initialize
	initialize()

	// Verify logger was initialized
	assert.NotNil(t, common.Logger)

	// Verify config was loaded
	cfg := config.GetConfig()
	assert.NotNil(t, cfg)
	assert.True(t, cfg.Auth.Enabled)
}

// writeFile is a helper function to write content to a file
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func TestInitialize_WithDebugEnabled(t *testing.T) {
	// Create a temp config file with debug enabled
	tempConfig := `
env: development
server:
  address: ":20229"
  debug: true
xrd:
  host: localhost
  port: 1094
  initial_dir: /tmp/
auth:
  enabled: false
logging:
  level: debug
  console:
    enabled: true
    format: text
`
	tempFile := "/tmp/test_init_debug_config.yaml"
	err := writeFile(tempFile, tempConfig)
	if err != nil {
		t.Skip("Could not create temp config file")
	}

	originalConfigFile := config.ConfigFile
	config.ConfigFile = tempFile
	defer func() {
		config.ConfigFile = originalConfigFile
	}()

	// Initialize
	initialize()

	// Verify logger was initialized
	assert.NotNil(t, common.Logger)

	// Verify config
	cfg := config.GetConfig()
	assert.NotNil(t, cfg)
	assert.True(t, cfg.Server.Debug)
	assert.Equal(t, "development", cfg.Env)
}

func TestInitialize_WithOIDCClientSecret(t *testing.T) {
	// Create a temp config file with full OIDC settings
	tempConfig := `
env: production
server:
  address: ":20230"
  debug: false
xrd:
  host: localhost
  port: 1094
  initial_dir: /tmp/
auth:
  enabled: true
  oidc:
    issuer: https://auth.example.com
    client_id: my-client-id
    client_secret: my-super-secret-key
    session_secret: session-secret-12345678901234567890
logging:
  level: info
  console:
    enabled: true
    format: json
`
	tempFile := "/tmp/test_init_oidc_config.yaml"
	err := writeFile(tempFile, tempConfig)
	if err != nil {
		t.Skip("Could not create temp config file")
	}

	originalConfigFile := config.ConfigFile
	config.ConfigFile = tempFile
	defer func() {
		config.ConfigFile = originalConfigFile
	}()

	// Initialize
	initialize()

	// Verify
	cfg := config.GetConfig()
	assert.NotNil(t, cfg)
	assert.True(t, cfg.Auth.Enabled)
	assert.Equal(t, "my-client-id", cfg.Auth.OIDC.ClientID)
	assert.NotEmpty(t, cfg.Auth.OIDC.ClientSecret)
}
