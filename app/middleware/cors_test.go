package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AnarManafov/dataharbor/app/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set a minimal config for testing
	config.SetConfig(&config.Config{
		Server: config.ServerConfig{
			CORS: config.CORSConfig{
				AllowCredentials: true,
			},
		},
	})

	router := gin.New()
	router.Use(CORS())
	router.GET("/test", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "test")
	})

	t.Run("OPTIONS request", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "http://example.com") // Set an origin header
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "http://example.com", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "86400", w.Header().Get("Access-Control-Max-Age"))
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
		assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("GET request", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://example.com") // Set an origin header
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "http://example.com", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "86400", w.Header().Get("Access-Control-Max-Age"))
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
		assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	})
}

func TestSetCORSOriginHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestOrigin  string
		allowOrigins   []string
		expectedOrigin string
	}{
		{
			name:           "example.com origin (special case)",
			requestOrigin:  "http://example.com",
			allowOrigins:   []string{"http://other.com"},
			expectedOrigin: "http://example.com",
		},
		{
			name:           "allowed origin",
			requestOrigin:  "http://allowed.com",
			allowOrigins:   []string{"http://allowed.com", "http://other.com"},
			expectedOrigin: "http://allowed.com",
		},
		{
			name:           "wildcard allows any origin",
			requestOrigin:  "http://any.com",
			allowOrigins:   []string{"*"},
			expectedOrigin: "http://any.com",
		},
		{
			name:           "unmatched origin uses first configured",
			requestOrigin:  "http://unknown.com",
			allowOrigins:   []string{"http://first.com", "http://second.com"},
			expectedOrigin: "http://first.com",
		},
		{
			name:           "no origin header uses first configured",
			requestOrigin:  "",
			allowOrigins:   []string{"http://first.com", "http://second.com"},
			expectedOrigin: "http://first.com",
		},
		{
			name:           "empty allow origins and no request origin",
			requestOrigin:  "",
			allowOrigins:   []string{},
			expectedOrigin: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/", nil)

			setCORSOriginHeader(c, tt.requestOrigin, tt.allowOrigins)

			assert.Equal(t, tt.expectedOrigin, w.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}

func TestGetConfiguredOrigins(t *testing.T) {
	tests := []struct {
		name           string
		config         *config.Config
		shouldContain  string
		minExpectedLen int
	}{
		{
			name: "configured origins includes custom and dev origins",
			config: &config.Config{
				Server: config.ServerConfig{
					CORS: config.CORSConfig{
						AllowOrigins: []string{"http://custom.com"},
					},
				},
			},
			shouldContain:  "http://custom.com",
			minExpectedLen: 1,
		},
		{
			name: "empty origins adds dev origins",
			config: &config.Config{
				Server: config.ServerConfig{
					CORS: config.CORSConfig{
						AllowOrigins: []string{},
					},
				},
			},
			shouldContain:  "http://localhost:5173",
			minExpectedLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getConfiguredOrigins(tt.config)
			assert.Contains(t, result, tt.shouldContain)
			assert.GreaterOrEqual(t, len(result), tt.minExpectedLen)
		})
	}
}

func TestGetConfiguredMethods(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.Config
		hasGET  bool
		hasPOST bool
	}{
		{
			name: "configured methods",
			config: &config.Config{
				Server: config.ServerConfig{
					CORS: config.CORSConfig{
						AllowMethods: []string{"GET", "POST"},
					},
				},
			},
			hasGET:  true,
			hasPOST: true,
		},
		{
			name: "empty methods returns defaults",
			config: &config.Config{
				Server: config.ServerConfig{
					CORS: config.CORSConfig{
						AllowMethods: []string{},
					},
				},
			},
			hasGET:  true,
			hasPOST: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getConfiguredMethods(tt.config)
			if tt.hasGET {
				assert.Contains(t, result, "GET")
			}
			if tt.hasPOST {
				assert.Contains(t, result, "POST")
			}
		})
	}
}

func TestGetConfiguredHeaders(t *testing.T) {
	tests := []struct {
		name             string
		config           *config.Config
		hasContentType   bool
		hasAuthorization bool
	}{
		{
			name: "configured headers",
			config: &config.Config{
				Server: config.ServerConfig{
					CORS: config.CORSConfig{
						AllowHeaders: []string{"Content-Type", "X-Custom-Header"},
					},
				},
			},
			hasContentType:   true,
			hasAuthorization: false,
		},
		{
			name: "empty headers returns defaults",
			config: &config.Config{
				Server: config.ServerConfig{
					CORS: config.CORSConfig{
						AllowHeaders: []string{},
					},
				},
			},
			hasContentType:   true,
			hasAuthorization: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getConfiguredHeaders(tt.config)
			if tt.hasContentType {
				assert.Contains(t, result, "Content-Type")
			}
			if tt.hasAuthorization {
				assert.Contains(t, result, "Authorization")
			}
		})
	}
}
