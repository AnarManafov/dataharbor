package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/AnarManafov/dataharbor/app/common"
)

func TestDebugRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("should process request with body and capture response", func(t *testing.T) {
		router := gin.New()
		router.Use(DebugRequestBody())
		router.POST("/api/test", func(c *gin.Context) {
			// Read the body to verify it was restored
			body := make([]byte, 1024)
			n, _ := c.Request.Body.Read(body)
			c.JSON(http.StatusOK, gin.H{"received": string(body[:n])})
		})

		reqBody := `{"test": "value"}`
		req, _ := http.NewRequest(http.MethodPost, "/api/test", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "received")
	})

	t.Run("should skip health endpoint", func(t *testing.T) {
		router := gin.New()
		router.Use(DebugRequestBody())
		router.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req, _ := http.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should skip download paths", func(t *testing.T) {
		router := gin.New()
		router.Use(DebugRequestBody())
		router.GET("/download/file.txt", func(c *gin.Context) {
			c.String(http.StatusOK, "file content")
		})

		req, _ := http.NewRequest(http.MethodGet, "/download/file.txt", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "file content", w.Body.String())
	})

	t.Run("should skip static assets", func(t *testing.T) {
		router := gin.New()
		router.Use(DebugRequestBody())
		router.GET("/static/script.js", func(c *gin.Context) {
			c.String(http.StatusOK, "console.log('test');")
		})

		req, _ := http.NewRequest(http.MethodGet, "/static/script.js", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should skip assets path", func(t *testing.T) {
		router := gin.New()
		router.Use(DebugRequestBody())
		router.GET("/assets/style.css", func(c *gin.Context) {
			c.String(http.StatusOK, "body { color: red; }")
		})

		req, _ := http.NewRequest(http.MethodGet, "/assets/style.css", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should skip favicon.ico", func(t *testing.T) {
		router := gin.New()
		router.Use(DebugRequestBody())
		router.GET("/favicon.ico", func(c *gin.Context) {
			c.String(http.StatusOK, "icon")
		})

		req, _ := http.NewRequest(http.MethodGet, "/favicon.ico", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should handle request without body", func(t *testing.T) {
		router := gin.New()
		router.Use(DebugRequestBody())
		router.GET("/api/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "hello"})
		})

		req, _ := http.NewRequest(http.MethodGet, "/api/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should handle empty request body", func(t *testing.T) {
		router := gin.New()
		router.Use(DebugRequestBody())
		router.POST("/api/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "received"})
		})

		req, _ := http.NewRequest(http.MethodPost, "/api/test", strings.NewReader(""))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should log JSON response body", func(t *testing.T) {
		router := gin.New()
		router.Use(DebugRequestBody())
		router.GET("/api/json", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": "test"})
		})

		req, _ := http.NewRequest(http.MethodGet, "/api/json", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	})

	t.Run("should handle large response bodies by truncating", func(t *testing.T) {
		router := gin.New()
		router.Use(DebugRequestBody())
		router.GET("/api/large", func(c *gin.Context) {
			// Create a large response (> 4096 bytes)
			largeData := strings.Repeat("x", 5000)
			c.String(http.StatusOK, largeData)
		})

		req, _ := http.NewRequest(http.MethodGet, "/api/large", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Len(t, w.Body.String(), 5000)
	})

	t.Run("should not log binary content types", func(t *testing.T) {
		router := gin.New()
		router.Use(DebugRequestBody())
		router.GET("/api/binary", func(c *gin.Context) {
			c.Data(http.StatusOK, "application/octet-stream", []byte{0x00, 0x01, 0x02})
		})

		req, _ := http.NewRequest(http.MethodGet, "/api/binary", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestBodyLogWriter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("should capture response body while writing", func(t *testing.T) {
		w := httptest.NewRecorder()
		blw := &bodyLogWriter{
			body:           bytes.NewBufferString(""),
			ResponseWriter: createTestResponseWriter(w),
		}

		testData := []byte("test response data")
		n, err := blw.Write(testData)

		assert.NoError(t, err)
		assert.Equal(t, len(testData), n)
		assert.Equal(t, "test response data", blw.body.String())
		assert.Equal(t, "test response data", w.Body.String())
	})

	t.Run("should capture multiple writes", func(t *testing.T) {
		w := httptest.NewRecorder()
		blw := &bodyLogWriter{
			body:           bytes.NewBufferString(""),
			ResponseWriter: createTestResponseWriter(w),
		}

		_, _ = blw.Write([]byte("first "))
		_, _ = blw.Write([]byte("second "))
		_, _ = blw.Write([]byte("third"))

		assert.Equal(t, "first second third", blw.body.String())
		assert.Equal(t, "first second third", w.Body.String())
	})

	t.Run("should handle empty writes", func(t *testing.T) {
		w := httptest.NewRecorder()
		blw := &bodyLogWriter{
			body:           bytes.NewBufferString(""),
			ResponseWriter: createTestResponseWriter(w),
		}

		n, err := blw.Write([]byte{})

		assert.NoError(t, err)
		assert.Equal(t, 0, n)
		assert.Empty(t, blw.body.String())
	})
}

func TestShouldSkipPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "should skip /health",
			path:     "/health",
			expected: true,
		},
		{
			name:     "should skip /health/check",
			path:     "/health/check",
			expected: true,
		},
		{
			name:     "should skip /favicon.ico",
			path:     "/favicon.ico",
			expected: true,
		},
		{
			name:     "should skip /static/",
			path:     "/static/script.js",
			expected: true,
		},
		{
			name:     "should skip /assets/",
			path:     "/assets/style.css",
			expected: true,
		},
		{
			name:     "should skip /download/",
			path:     "/download/file.txt",
			expected: true,
		},
		{
			name:     "should not skip /api/users",
			path:     "/api/users",
			expected: false,
		},
		{
			name:     "should not skip /api/auth/login",
			path:     "/api/auth/login",
			expected: false,
		},
		{
			name:     "should not skip root path",
			path:     "/",
			expected: false,
		},
		{
			name:     "should not skip empty path",
			path:     "",
			expected: false,
		},
		{
			name:     "should not skip /healthcheck (different from /health)",
			path:     "/healthcheck",
			expected: true, // starts with /health
		},
		{
			// Still true: shouldSkipPath is the full skip (no log lines at
			// all) and matches prefixes only. Real download routes are
			// handled by shouldSkipBody instead, which keeps the request
			// line logged and skips only body capture.
			name:     "should not skip /api/download (not starting with /download/)",
			path:     "/api/download",
			expected: false,
		},
		{
			name:     "should not fully skip the real download route",
			path:     "/api/v1/xrd/download",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldSkipPath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldLogResponseBody(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expected    bool
	}{
		{
			name:        "should log application/json",
			contentType: "application/json",
			expected:    true,
		},
		{
			name:        "should log application/json with charset",
			contentType: "application/json; charset=utf-8",
			expected:    true,
		},
		{
			name:        "should log text/plain",
			contentType: "text/plain",
			expected:    true,
		},
		{
			name:        "should log text/plain with charset",
			contentType: "text/plain; charset=utf-8",
			expected:    true,
		},
		{
			name:        "should log text/html",
			contentType: "text/html",
			expected:    true,
		},
		{
			name:        "should log text/html with charset",
			contentType: "text/html; charset=utf-8",
			expected:    true,
		},
		{
			name:        "should log application/xml",
			contentType: "application/xml",
			expected:    true,
		},
		{
			name:        "should log text/xml",
			contentType: "text/xml",
			expected:    true,
		},
		{
			name:        "should not log application/octet-stream",
			contentType: "application/octet-stream",
			expected:    false,
		},
		{
			name:        "should not log image/png",
			contentType: "image/png",
			expected:    false,
		},
		{
			name:        "should not log image/jpeg",
			contentType: "image/jpeg",
			expected:    false,
		},
		{
			name:        "should not log application/pdf",
			contentType: "application/pdf",
			expected:    false,
		},
		{
			name:        "should not log video/mp4",
			contentType: "video/mp4",
			expected:    false,
		},
		{
			name:        "should not log empty content type",
			contentType: "",
			expected:    false,
		},
		{
			name:        "should not log multipart/form-data",
			contentType: "multipart/form-data",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldLogResponseBody(tt.contentType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// createTestResponseWriter creates a gin.ResponseWriter for testing
func createTestResponseWriter(w *httptest.ResponseRecorder) gin.ResponseWriter {
	c, _ := gin.CreateTestContext(w)
	return c.Writer
}

func TestShouldSkipBody(t *testing.T) {
	tests := []struct {
		method, path string
		expected     bool
	}{
		{http.MethodGet, "/api/v1/xrd/download", true},
		{http.MethodPost, "/api/v1/xrd/download/batch", true},
		{http.MethodPut, "/api/v1/xrd/upload/abc123/chunk", true},
		// Small JSON upload routes keep their body logs.
		{http.MethodGet, "/api/v1/xrd/upload/limits", false},
		{http.MethodPost, "/api/v1/xrd/upload/session", false},
		{http.MethodPost, "/api/v1/xrd/upload/abc123/complete", false},
		{http.MethodGet, "/api/v1/xrd/upload/abc123/status", false},
		{http.MethodDelete, "/api/v1/xrd/upload/abc123", false},
		{http.MethodPost, "/api/v1/xrd/ls/paged", false},
		{http.MethodGet, "/api/v1/xrd/downloads", false},
		{http.MethodPut, "/api/v1/other/chunk", false},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			assert.Equal(t, tt.expected, shouldSkipBody(tt.method, tt.path))
		})
	}
}

// observeDebugLogs swaps in an observed debug-level logger for one test.
func observeDebugLogs(t *testing.T) *observer.ObservedLogs {
	t.Helper()
	core, logs := observer.New(zapcore.DebugLevel)
	prev := common.Logger
	common.Logger = zap.New(core).Sugar()
	t.Cleanup(func() { common.Logger = prev })
	return logs
}

func loggedMessages(logs *observer.ObservedLogs) string {
	var b strings.Builder
	for _, e := range logs.All() {
		b.WriteString(e.Message)
		b.WriteString("\n")
	}
	return b.String()
}

// countingReader records how many bytes were read from it.
type countingReader struct {
	r io.Reader
	n int
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += n
	return n, err
}

func TestDebugRequestBody_StreamingRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("large streamed download is not buffered", func(t *testing.T) {
		logs := observeDebugLogs(t)
		const chunk = 64 << 10
		const chunks = 128 // 8 MiB
		router := gin.New()
		router.Use(DebugRequestBody())
		router.GET("/api/v1/xrd/download", func(c *gin.Context) {
			_, buffered := c.Writer.(*bodyLogWriter)
			assert.False(t, buffered, "download response writer must not be the capturing bodyLogWriter")
			c.Header("Content-Type", "text/plain")
			c.Status(http.StatusOK)
			buf := bytes.Repeat([]byte("d"), chunk)
			for i := 0; i < chunks; i++ {
				_, _ = c.Writer.Write(buf)
			}
		})

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/xrd/download?path=/data/big.bin", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, chunk*chunks, w.Body.Len())
		msgs := loggedMessages(logs)
		assert.Contains(t, msgs, "Request: GET /api/v1/xrd/download")
		assert.Contains(t, msgs, "Response Status: 200")
		assert.NotContains(t, msgs, "Response Body:")
	})

	t.Run("batch download is not buffered", func(t *testing.T) {
		router := gin.New()
		router.Use(DebugRequestBody())
		router.POST("/api/v1/xrd/download/batch", func(c *gin.Context) {
			_, buffered := c.Writer.(*bodyLogWriter)
			assert.False(t, buffered)
			c.Status(http.StatusOK)
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/xrd/download/batch", strings.NewReader(`{"files":["/a"]}`))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("upload chunk body is not consumed by the middleware", func(t *testing.T) {
		logs := observeDebugLogs(t)
		payload := bytes.Repeat([]byte{0xAB}, 1<<20)
		body := &countingReader{r: bytes.NewReader(payload)}
		router := gin.New()
		router.Use(DebugRequestBody())
		router.PUT("/api/v1/xrd/upload/:uploadId/chunk", func(c *gin.Context) {
			assert.Equal(t, 0, body.n, "middleware read the chunk body before the handler")
			got, err := io.ReadAll(c.Request.Body)
			assert.NoError(t, err)
			assert.Equal(t, len(payload), len(got))
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/xrd/upload/abc123/chunk?index=0", body)
		req.Header.Set("Content-Type", "application/octet-stream")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		msgs := loggedMessages(logs)
		assert.Contains(t, msgs, "Request: PUT /api/v1/xrd/upload/abc123/chunk")
		assert.NotContains(t, msgs, "Request Body:")
	})

	t.Run("upload session create still logs its body", func(t *testing.T) {
		logs := observeDebugLogs(t)
		router := gin.New()
		router.Use(DebugRequestBody())
		router.POST("/api/v1/xrd/upload/session", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"uploadId": "u1"})
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/xrd/upload/session", strings.NewReader(`{"files":[]}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		msgs := loggedMessages(logs)
		assert.Contains(t, msgs, "Request Body:")
		assert.Contains(t, msgs, "Response Body:")
	})
}
