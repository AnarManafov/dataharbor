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
