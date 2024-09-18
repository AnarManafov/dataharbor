package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORSMiddleware())
	router.GET("/test", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "test")
	})

	t.Run("OPTIONS request", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "86400", w.Header().Get("Access-Control-Max-Age"))
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("GET request", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "86400", w.Header().Get("Access-Control-Max-Age"))
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	})
}
