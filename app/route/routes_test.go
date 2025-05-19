package route

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/AnarManafov/dataharbor/app/common"
	"github.com/AnarManafov/dataharbor/app/config"
)

func TestRegisterRoutes(t *testing.T) {
	// Set up gin in test mode
	gin.SetMode(gin.TestMode)

	// Initialize config for testing
	testConfig := &config.Config{
		Server: config.ServerConfig{
			Address: "localhost:8080",
			Debug:   false,
		},
		XRD: config.XRDConfig{
			Host:       "localhost",
			Port:       1094,
			InitialDir: "/tmp",
		},
		Logging: config.LoggingConfig{
			Level: "info",
			Console: config.ConsoleConfig{
				Enabled: true,
				Format:  "text",
				Level:   "info",
			},
			File: config.FileConfig{
				Enabled: false,
			},
		},
	}

	// Set the config
	config.SetConfig(testConfig)

	// Initialize logger
	common.InitLogger(&testConfig.Logging)

	r := gin.New()

	// Register routes
	RegisterRoutes(r)

	// Create test server
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Test health endpoint
	resp, err := http.Get(ts.URL + "/health")
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Test API endpoints with correct paths
	resp, err = http.Get(ts.URL + "/api/v1/xrd/initialDir")
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	resp, err = http.Get(ts.URL + "/api/v1/xrd/hostname")
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}
