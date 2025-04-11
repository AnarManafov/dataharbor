package route

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRegisterRoutes(t *testing.T) {
	// Set up gin in test mode
	gin.SetMode(gin.TestMode)
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
	resp, err = http.Get(ts.URL + "/api/xrd/initialDir")
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	resp, err = http.Get(ts.URL + "/api/xrd/hostname")
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}
