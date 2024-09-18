package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestTraceMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("should set tid from header if present", func(t *testing.T) {
		router := gin.New()
		router.Use(TraceMiddleware())
		router.GET("/test", func(ctx *gin.Context) {
			tid, exists := ctx.Get("tid")
			assert.True(t, exists)
			assert.Equal(t, "test-tid", tid)
			ctx.Status(http.StatusOK)
		})

		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Tid", "test-tid")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should generate new tid if header is not present", func(t *testing.T) {
		router := gin.New()
		router.Use(TraceMiddleware())
		router.GET("/test", func(ctx *gin.Context) {
			tid, exists := ctx.Get("tid")
			assert.True(t, exists)
			assert.NotEmpty(t, tid)
			ctx.Status(http.StatusOK)
		})

		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
