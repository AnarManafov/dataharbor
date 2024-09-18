package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRecoveryMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("should recover from panic and return 400 status code", func(t *testing.T) {
		router := gin.New()
		router.Use(RecoveryMiddleware())
		router.GET("/panic", func(c *gin.Context) {
			panic("test panic")
		})

		req, _ := http.NewRequest(http.MethodGet, "/panic", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "test panic")
	})

	t.Run("should pass through without panic", func(t *testing.T) {
		router := gin.New()
		router.Use(RecoveryMiddleware())
		router.GET("/no-panic", func(c *gin.Context) {
			c.String(http.StatusOK, "no panic")
		})

		req, _ := http.NewRequest(http.MethodGet, "/no-panic", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "no panic", w.Body.String())
	})
}
