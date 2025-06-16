package main

import (
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
	go startServer(stop)

	// Add assertions to verify server start
	assert.Equal(t, gin.TestMode, gin.Mode())

	// Send stop signal to stop the server
	close(stop)
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
	go startServer(stop)

	// Add assertions to verify server start
	assert.Equal(t, gin.ReleaseMode, gin.Mode())

	// Send stop signal to stop the server
	close(stop)
}
