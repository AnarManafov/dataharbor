package main

import (
	"testing"

	"github.com/AnarManafov/dataharbor/app/common"
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
	assert.NotNil(t, common.XrdConfig)
	assert.NotNil(t, common.ServerConfig)
}

func TestStartServer_DebugMode(t *testing.T) {
	// Mock the server configuration
	common.ServerConfig = common.ServerConfigType{
		Debug: true,
		Port:  20222,
	}

	// Mock gin router
	gin.SetMode(gin.TestMode)

	// Create a stop channel
	stop := make(chan struct{})

	// Call the startServer function
	go startServer(stop)

	// Add assertions to verify server start
	assert.Equal(t, gin.TestMode, gin.Mode())
	assert.Equal(t, 20222, common.ServerConfig.Port)

	// Send stop signal to stop the server
	close(stop)
}

func TestStartServer_ReleaseMode(t *testing.T) {
	// Mock the server configuration
	common.ServerConfig = common.ServerConfigType{
		Debug: false,
		Port:  20222,
	}

	// Mock gin router
	gin.SetMode(gin.ReleaseMode)

	// Create a stop channel
	stop := make(chan struct{})

	// Call the startServer function
	go startServer(stop)

	// Add assertions to verify server start
	assert.Equal(t, gin.ReleaseMode, gin.Mode())
	assert.Equal(t, 20222, common.ServerConfig.Port)

	// Send stop signal to stop the server
	close(stop)
}

// func TestStartServer_Error(t *testing.T) {
// 	// Mock the server configuration
// 	common.ServerConfig = common.ServerConfigType{
// 		Debug: true,
// 		Port:  20222,
// 	}

// 	// Mock gin router
// 	gin.SetMode(gin.TestMode)

// 	// Mock the gin.Engine to return an error
// 	mockEngine := new(MockEngine)
// 	mockEngine.On("Run", ":"+strconv.Itoa(common.ServerConfig.Port)).Return(errors.New("server error"))

// 	originalGinNew := gin.New
// 	defer func() { gin.New = originalGinNew }()
// 	gin.New = func() *gin.Engine {
// 		return mockEngine
// 	}

// 	// Call the startServer function
// 	go startServer()

// 	// Add assertions to verify server start
// 	mockEngine.AssertCalled(t, "Run", ":"+strconv.Itoa(common.ServerConfig.Port))
// }
