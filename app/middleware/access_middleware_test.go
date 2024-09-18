package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAccessLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("should log access for non-multipart request", func(t *testing.T) {
		router := gin.New()
		router.Use(AccessLogger())
		router.POST("/test", func(ctx *gin.Context) {
			ctx.String(http.StatusOK, "OK")
		})

		body := `{"key":"value"}`
		req, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tid", "test-tid")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "OK", w.Body.String())
	})

	t.Run("should log access for multipart request", func(t *testing.T) {
		router := gin.New()
		router.Use(AccessLogger())
		router.POST("/test", func(ctx *gin.Context) {
			ctx.String(http.StatusOK, "OK")
		})

		body := `--boundary
Content-Disposition: form-data; name="file"; filename="test.txt"
Content-Type: text/plain

test content
--boundary--`
		req, _ := http.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
		req.Header.Set("X-Tid", "test-tid")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "OK", w.Body.String())
	})

	t.Run("should log access with no X-Tid header", func(t *testing.T) {
		router := gin.New()
		router.Use(AccessLogger())
		router.POST("/test", func(ctx *gin.Context) {
			ctx.String(http.StatusOK, "OK")
		})

		body := `{"key":"value"}`
		req, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "OK", w.Body.String())
	})

	t.Run("should log access with empty body", func(t *testing.T) {
		router := gin.New()
		router.Use(AccessLogger())
		router.POST("/test", func(ctx *gin.Context) {
			ctx.String(http.StatusOK, "OK")
		})

		req, _ := http.NewRequest(http.MethodPost, "/test", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tid", "test-tid")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "OK", w.Body.String())
	})
}
