package middleware

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/AnarManafov/dataharbor/app/common"
	"github.com/AnarManafov/dataharbor/app/config"
)

// Route patterns as routes.go registers them.
const (
	testDownloadRoute = "/api/v1/xrd/download"
	testBatchRoute    = "/api/v1/xrd/download/batch"
	testChunkRoute    = "/api/v1/xrd/upload/:uploadId/chunk"
)

func testStreamingSet() *RouteSet {
	s := NewRouteSet()
	s.Add(http.MethodGet, testDownloadRoute)
	s.Add(http.MethodPost, testBatchRoute)
	s.Add(http.MethodPut, testChunkRoute)
	return s
}

func TestDebugRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("should process request with body and capture response", func(t *testing.T) {
		router := gin.New()
		router.Use(DebugRequestBody(0, nil))
		router.POST("/api/test", func(c *gin.Context) {
			body, _ := io.ReadAll(c.Request.Body)
			c.JSON(http.StatusOK, gin.H{"received": string(body)})
		})

		reqBody := `{"test": "value"}`
		req, _ := http.NewRequest(http.MethodPost, "/api/test", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"received":"{\"test\": \"value\"}"`)
	})

	for _, path := range []string{"/health", "/static/script.js", "/assets/style.css", "/favicon.ico"} {
		t.Run("should skip "+path, func(t *testing.T) {
			logs := observeDebugLogs(t)
			router := gin.New()
			router.Use(DebugRequestBody(0, nil))
			router.GET(path, func(c *gin.Context) {
				c.String(http.StatusOK, "content")
			})

			req, _ := http.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "content", w.Body.String())
			assert.Empty(t, logs.All())
		})
	}

	t.Run("should handle request without body", func(t *testing.T) {
		logs := observeDebugLogs(t)
		router := gin.New()
		router.Use(DebugRequestBody(0, nil))
		router.GET("/api/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "hello"})
		})

		req, _ := http.NewRequest(http.MethodGet, "/api/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		msgs := loggedMessages(logs)
		assert.NotContains(t, msgs, "Request Body:")
		assert.Contains(t, msgs, `Response Body: {"message":"hello"}`)
	})

	t.Run("should handle empty request body", func(t *testing.T) {
		router := gin.New()
		router.Use(DebugRequestBody(0, nil))
		router.POST("/api/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "received"})
		})

		req, _ := http.NewRequest(http.MethodPost, "/api/test", strings.NewReader(""))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should describe, not log, non-JSON responses", func(t *testing.T) {
		logs := observeDebugLogs(t)
		router := gin.New()
		router.Use(DebugRequestBody(0, nil))
		router.GET("/api/binary", func(c *gin.Context) {
			_, buffered := c.Writer.(*bodyLogWriter)
			require.True(t, buffered)
			c.Data(http.StatusOK, "application/octet-stream", []byte{0x00, 0x01, 0x02})
		})
		router.GET("/api/text", func(c *gin.Context) {
			c.String(http.StatusOK, "plain text")
		})

		for _, p := range []string{"/api/binary", "/api/text"} {
			req, _ := http.NewRequest(http.MethodGet, p, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}
		msgs := loggedMessages(logs)
		assert.Contains(t, msgs, "Response Body: [3 bytes of application/octet-stream not logged]")
		assert.Contains(t, msgs, "Response Body: [10 bytes of text/plain not logged]")
		assert.NotContains(t, msgs, "plain text")
	})

	t.Run("non-positive limit falls back to the default", func(t *testing.T) {
		logs := observeDebugLogs(t)
		router := gin.New()
		router.Use(DebugRequestBody(-1, nil))
		router.GET("/api/json", func(c *gin.Context) {
			blw := c.Writer.(*bodyLogWriter)
			assert.Equal(t, config.DefaultDebugBodyLogLimit, blw.limit)
			c.JSON(http.StatusOK, gin.H{"pad": strings.Repeat("x", config.DefaultDebugBodyLogLimit-20)})
		})
		req, _ := http.NewRequest(http.MethodGet, "/api/json", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		msgs := loggedMessages(logs)
		assert.Contains(t, msgs, "Response Body: {")
		assert.NotContains(t, msgs, "(truncated)")
	})
}

func TestBodyLogWriter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newWriter := func(w *httptest.ResponseRecorder, contentType string, limit int) *bodyLogWriter {
		blw := &bodyLogWriter{ResponseWriter: createTestResponseWriter(w), limit: limit}
		blw.Header().Set("Content-Type", contentType)
		return blw
	}

	t.Run("should capture response body while writing", func(t *testing.T) {
		w := httptest.NewRecorder()
		blw := newWriter(w, "application/json", 64)

		testData := []byte(`{"a":"test response data"}`)
		n, err := blw.Write(testData)

		assert.NoError(t, err)
		assert.Equal(t, len(testData), n)
		assert.Equal(t, string(testData), string(blw.body))
		assert.Equal(t, string(testData), w.Body.String())
		assert.Equal(t, `{"a":"test response data"}`, blw.render())
	})

	t.Run("should capture multiple writes and WriteString", func(t *testing.T) {
		w := httptest.NewRecorder()
		blw := newWriter(w, "application/json", 64)

		_, _ = blw.Write([]byte(`{"a":`))
		_, _ = blw.WriteString(`"second"`)
		_, _ = blw.Write([]byte(`}`))

		assert.Equal(t, `{"a":"second"}`, string(blw.body))
		assert.Equal(t, `{"a":"second"}`, w.Body.String())
		assert.Equal(t, int64(14), blw.total)
	})

	t.Run("should keep at most limit bytes", func(t *testing.T) {
		w := httptest.NewRecorder()
		blw := newWriter(w, "application/json", 16)

		_, _ = blw.Write([]byte(`{"a":"`))
		for i := 0; i < 1000; i++ {
			_, _ = blw.WriteString("yyyyyyyyyy")
			_, _ = blw.Write([]byte("zzzzzzzzzz"))
		}
		_, _ = blw.Write([]byte(`"}`))

		assert.Len(t, blw.body, 16)
		assert.Equal(t, 16, cap(blw.body), "the capture buffer must never grow past the limit")
		assert.Equal(t, int64(6+20000+2), blw.total)
		assert.Equal(t, 6+20000+2, w.Body.Len())
		assert.Equal(t, `{`+common.TruncatedBodyMarker, blw.render())
	})

	t.Run("should not keep non-JSON bodies", func(t *testing.T) {
		w := httptest.NewRecorder()
		blw := newWriter(w, "text/html; charset=utf-8", 64)

		_, _ = blw.WriteString("<html>secret-ish</html>")

		assert.Empty(t, blw.body)
		assert.Equal(t, "<html>secret-ish</html>", w.Body.String())
		assert.Equal(t, "[23 bytes of text/html not logged]", blw.render())
	})

	t.Run("should handle empty writes", func(t *testing.T) {
		w := httptest.NewRecorder()
		blw := newWriter(w, "application/json", 64)

		n, err := blw.Write([]byte{})

		assert.NoError(t, err)
		assert.Equal(t, 0, n)
		assert.Empty(t, blw.body)
	})
}

func TestShouldSkipPath(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/health", true},
		{"/health/check", true},
		{"/healthcheck", true}, // prefix match
		{"/favicon.ico", true},
		{"/static/script.js", true},
		{"/assets/style.css", true},
		// Transfer routes are not skipped here: they are keyed by their
		// registered route (RouteSet) and keep their request lines.
		{"/download/file.txt", false},
		{"/api/v1/xrd/download", false},
		{"/api/users", false},
		{"/api/auth/login", false},
		{"/", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.expected, shouldSkipPath(tt.path))
		})
	}
}

func TestRouteSet(t *testing.T) {
	s := testStreamingSet()
	assert.True(t, s.Has(http.MethodGet, testDownloadRoute))
	assert.True(t, s.Has(http.MethodPut, testChunkRoute))
	assert.False(t, s.Has(http.MethodPost, testDownloadRoute), "method is part of the key")
	assert.False(t, s.Has(http.MethodPut, "/api/v1/xrd/upload/abc123/chunk"), "keys are route patterns, not request paths")
	assert.False(t, s.Has(http.MethodGet, ""), "an unmatched route is never streaming")
	assert.Equal(t, []string{
		"GET /api/v1/xrd/download",
		"POST /api/v1/xrd/download/batch",
		"PUT /api/v1/xrd/upload/:uploadId/chunk",
	}, s.List())

	var nilSet *RouteSet
	assert.False(t, nilSet.Has(http.MethodGet, testDownloadRoute))
	assert.Nil(t, nilSet.List())
}

// createTestResponseWriter creates a gin.ResponseWriter for testing
func createTestResponseWriter(w *httptest.ResponseRecorder) gin.ResponseWriter {
	c, _ := gin.CreateTestContext(w)
	return c.Writer
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

// loggedLine returns the first message that starts with prefix.
func loggedLine(logs *observer.ObservedLogs, prefix string) (string, bool) {
	for _, e := range logs.All() {
		if strings.HasPrefix(e.Message, prefix) {
			return e.Message, true
		}
	}
	return "", false
}

// countingReader records how many bytes were read from it. It is a
// ReadCloser so that http.NewRequest uses it as the body as is, and the
// handler can check it received this very reader.
type countingReader struct {
	r      io.Reader
	n      int
	closed bool
}

func (c *countingReader) Close() error {
	c.closed = true
	return nil
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += n
	return n, err
}

// patterned returns n deterministic bytes that are not all alike.
func patterned(n int, alphabet string) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[(i*7+i/13)%len(alphabet)]
	}
	return b
}

func TestDebugRequestBody_StreamingRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("large streamed download is written straight through", func(t *testing.T) {
		logs := observeDebugLogs(t)
		payload := patterned(8<<20, "abcdefghij") // 8 MiB
		router := gin.New()
		router.Use(DebugRequestBody(64, testStreamingSet()))
		router.GET(testDownloadRoute, func(c *gin.Context) {
			_, buffered := c.Writer.(*bodyLogWriter)
			assert.False(t, buffered, "download response writer must not be the capturing bodyLogWriter")
			c.Header("Content-Type", "application/json") // even a loggable type is not captured
			c.Status(http.StatusOK)
			for off := 0; off < len(payload); off += 64 << 10 {
				_, _ = c.Writer.Write(payload[off : off+64<<10])
			}
		})

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/xrd/download?path=/data/big.bin", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, sha256.Sum256(payload), sha256.Sum256(w.Body.Bytes()))
		msgs := loggedMessages(logs)
		assert.Contains(t, msgs, "Request: GET /api/v1/xrd/download")
		assert.Contains(t, msgs, "Response Status: 200")
		assert.NotContains(t, msgs, "Response Body:")
	})

	t.Run("batch download request and response are not touched", func(t *testing.T) {
		logs := observeDebugLogs(t)
		var origBody io.ReadCloser
		body := &countingReader{r: strings.NewReader(`{"files":["/a"]}`)}
		router := gin.New()
		router.Use(DebugRequestBody(64, testStreamingSet()))
		router.POST(testBatchRoute, func(c *gin.Context) {
			_, buffered := c.Writer.(*bodyLogWriter)
			assert.False(t, buffered)
			assert.Same(t, origBody, c.Request.Body, "the batch request body must not be wrapped")
			assert.Equal(t, 0, body.n)
			c.Status(http.StatusOK)
		})
		req, _ := http.NewRequest(http.MethodPost, testBatchRoute, body)
		req.Header.Set("Content-Type", "application/json")
		origBody = req.Body
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.NotContains(t, loggedMessages(logs), "Request Body:")
	})

	t.Run("large upload chunk reaches the handler unread and unwrapped", func(t *testing.T) {
		logs := observeDebugLogs(t)
		payload := patterned(16<<20, "0123456789abcdef") // 16 MiB
		body := &countingReader{r: bytes.NewReader(payload)}
		var origBody io.ReadCloser
		router := gin.New()
		router.Use(DebugRequestBody(64, testStreamingSet()))
		router.PUT(testChunkRoute, func(c *gin.Context) {
			assert.Equal(t, 0, body.n, "middleware read the chunk body before the handler")
			assert.Same(t, origBody, c.Request.Body, "the chunk body must not be wrapped")
			h := sha256.New()
			n, err := io.Copy(h, c.Request.Body)
			assert.NoError(t, err)
			assert.Equal(t, int64(len(payload)), n)
			assert.Equal(t, sha256.Sum256(payload), [32]byte(h.Sum(nil)))
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/xrd/upload/abc123/chunk?index=0", body)
		// Even a type the middleware would log must not be captured here.
		req.Header.Set("Content-Type", "application/json")
		origBody = req.Body
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		msgs := loggedMessages(logs)
		assert.Contains(t, msgs, "Request: PUT /api/v1/xrd/upload/abc123/chunk")
		assert.NotContains(t, msgs, "Request Body:")
		assert.NotContains(t, msgs, "Response Body:")
	})

	t.Run("the same path under another method is not streaming", func(t *testing.T) {
		logs := observeDebugLogs(t)
		router := gin.New()
		router.Use(DebugRequestBody(64, testStreamingSet()))
		router.POST(testChunkRoute, func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/xrd/upload/abc123/chunk", strings.NewReader(`{"a":1}`))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(httptest.NewRecorder(), req)
		assert.Contains(t, loggedMessages(logs), `Request Body: {"a":1}`)
	})

	t.Run("upload session create still logs its body", func(t *testing.T) {
		logs := observeDebugLogs(t)
		router := gin.New()
		router.Use(DebugRequestBody(0, testStreamingSet()))
		router.POST("/api/v1/xrd/upload/session", func(c *gin.Context) {
			// Does not read the body: it is logged anyway.
			c.JSON(http.StatusOK, gin.H{"uploadId": "u1"})
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/xrd/upload/session", strings.NewReader(`{"files":[]}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		msgs := loggedMessages(logs)
		assert.Contains(t, msgs, `Request Body: {"files":[]}`)
		assert.Contains(t, msgs, `Response Body: {"uploadId":"u1"}`)
	})
}

func TestDebugRequestBody_CappedCapture(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const limit = 256

	// bigJSON builds a JSON document of at least n bytes.
	bigJSON := func(n int) []byte {
		var b bytes.Buffer
		b.WriteString(`{"note":"visible-head","items":[`)
		for i := 0; b.Len() < n; i++ {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(`{"name":"file-`)
			b.WriteString(strings.Repeat("n", i%17))
			b.WriteString(`","size":123456}`)
		}
		b.WriteString(`]}`)
		return b.Bytes()
	}

	t.Run("large request and response bodies pass through, capped log", func(t *testing.T) {
		logs := observeDebugLogs(t)
		reqPayload := bigJSON(4 << 20)
		respPayload := bigJSON(4 << 20)
		body := &countingReader{r: bytes.NewReader(reqPayload)}
		var blw *bodyLogWriter

		router := gin.New()
		router.Use(DebugRequestBody(limit, testStreamingSet()))
		router.POST("/api/v1/xrd/ls/paged", func(c *gin.Context) {
			assert.LessOrEqual(t, body.n, limit+1, "middleware read past its limit before the handler")
			got, err := io.ReadAll(c.Request.Body)
			assert.NoError(t, err)
			assert.Equal(t, sha256.Sum256(reqPayload), sha256.Sum256(got), "handler did not receive the body unchanged")
			assert.NoError(t, c.Request.Body.Close())
			assert.True(t, body.closed, "Close did not reach the original body")

			blw = c.Writer.(*bodyLogWriter)
			c.Header("Content-Type", "application/json; charset=utf-8")
			c.Status(http.StatusOK)
			half := len(respPayload) / 2
			_, _ = c.Writer.Write(respPayload[:half])
			_, _ = c.Writer.WriteString(string(respPayload[half:]))
		})

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/xrd/ls/paged", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, sha256.Sum256(respPayload), sha256.Sum256(w.Body.Bytes()), "client did not receive the response unchanged")
		require.NotNil(t, blw)
		assert.Len(t, blw.body, limit)
		assert.Equal(t, limit, cap(blw.body), "response capture grew past the limit")
		assert.Equal(t, int64(len(respPayload)), blw.total)

		for _, prefix := range []string{"Request Body: ", "Response Body: "} {
			line, ok := loggedLine(logs, prefix)
			require.True(t, ok, "no %q line", prefix)
			assert.True(t, strings.HasSuffix(line, common.TruncatedBodyMarker), "not marked truncated: %s", line)
			assert.Contains(t, line, "visible-head")
			// Redaction changes the length, so assert a bound, not equality.
			assert.LessOrEqual(t, len(line), len(prefix)+limit+len(common.TruncatedBodyMarker)+64, "log line not capped: %d bytes", len(line))
		}
	})

	t.Run("body at the limit is logged whole", func(t *testing.T) {
		logs := observeDebugLogs(t)
		payload := `{"pad":"` + strings.Repeat("p", limit-10) + `"}`
		require.Len(t, payload, limit)
		router := gin.New()
		router.Use(DebugRequestBody(limit, nil))
		router.POST("/api/x", func(c *gin.Context) {
			got, _ := io.ReadAll(c.Request.Body)
			assert.Equal(t, payload, string(got))
			c.Status(http.StatusNoContent)
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/x", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(httptest.NewRecorder(), req)
		assert.Contains(t, loggedMessages(logs), "Request Body: "+payload+"\n")
	})

	t.Run("non-JSON request bodies are only sniffed", func(t *testing.T) {
		for _, ct := range []string{"application/octet-stream", "multipart/form-data; boundary=xyz", "text/plain", ""} {
			t.Run(ct, func(t *testing.T) {
				logs := observeDebugLogs(t)
				payload := patterned(1<<20, "xyz")
				body := &countingReader{r: bytes.NewReader(payload)}
				router := gin.New()
				router.Use(DebugRequestBody(limit, nil))
				router.POST("/api/v1/other", func(c *gin.Context) {
					assert.LessOrEqual(t, body.n, sniffLen, "middleware read more of a %q body than it needs to classify it", ct)
					got, _ := io.ReadAll(c.Request.Body)
					assert.Equal(t, sha256.Sum256(payload), sha256.Sum256(got), "handler did not receive the body unchanged")
					c.Status(http.StatusNoContent)
				})
				req, _ := http.NewRequest(http.MethodPost, "/api/v1/other", body)
				req.ContentLength = int64(len(payload))
				if ct != "" {
					req.Header.Set("Content-Type", ct)
				}
				router.ServeHTTP(httptest.NewRecorder(), req)

				line, ok := loggedLine(logs, "Request Body: ")
				require.True(t, ok)
				assert.Contains(t, line, "[1048576 bytes of ")
				assert.Contains(t, line, " not logged]")
				assert.NotContains(t, loggedMessages(logs), string(payload[:16]))
			})
		}
	})

	t.Run("unmatched routes are capped too", func(t *testing.T) {
		logs := observeDebugLogs(t)
		router := gin.New()
		router.Use(DebugRequestBody(limit, testStreamingSet()))
		router.NoRoute(func(c *gin.Context) {
			assert.Empty(t, c.FullPath())
			_, _ = io.Copy(io.Discard, c.Request.Body)
			c.JSON(http.StatusNotFound, gin.H{"message": "API endpoint not found"})
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/nowhere", bytes.NewReader(bigJSON(1<<20)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
		line, ok := loggedLine(logs, "Request Body: ")
		require.True(t, ok)
		assert.True(t, strings.HasSuffix(line, common.TruncatedBodyMarker))
	})

	t.Run("a read error reaches the handler", func(t *testing.T) {
		observeDebugLogs(t)
		boom := errors.New("client went away")
		router := gin.New()
		router.Use(DebugRequestBody(limit, nil))
		router.POST("/api/x", func(c *gin.Context) {
			got, err := io.ReadAll(c.Request.Body)
			assert.ErrorIs(t, err, boom)
			assert.Equal(t, `{"a":`, string(got))
			c.Status(http.StatusBadRequest)
		})
		body := io.MultiReader(strings.NewReader(`{"a":`), errReader{boom})
		req, _ := http.NewRequest(http.MethodPost, "/api/x", body)
		req.ContentLength = -1
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("a body cut by a read error is rendered as truncated", func(t *testing.T) {
		logs := observeDebugLogs(t)
		const secret = "Wq7Lm2Zx9Kd4Rt6Vb1"
		router := gin.New()
		router.Use(DebugRequestBody(limit, nil))
		router.POST("/api/x", func(c *gin.Context) {
			_, _ = io.Copy(io.Discard, c.Request.Body)
			c.Status(http.StatusBadRequest)
		})
		// The client went away inside a URL password: without its "@" the
		// value no longer looks like userinfo.
		body := io.MultiReader(strings.NewReader("a=1&next=https://alice:"+secret[:12]), errReader{errors.New("unexpected EOF")})
		req, _ := http.NewRequest(http.MethodPost, "/api/x", body)
		req.ContentLength = -1
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(httptest.NewRecorder(), req)

		line, ok := loggedLine(logs, "Request Body: ")
		require.True(t, ok)
		assert.Equal(t, "Request Body: a=1"+common.TruncatedBodyMarker, line)
		assert.NotContains(t, loggedMessages(logs), secret[:8])
	})
}

// Sentinels for JSON bodies whose Content-Type does not say JSON.
const (
	sniffSecret   = "Mv5Tq8Ld2Hx7Wc4Rk9Bz"
	invalidSecret = "Fn3Yp6Gs1Ju8Ka4Xe7Do"
)

// assertNoFragment fails if any 8-byte window of secret was logged.
func assertNoFragment(t *testing.T, logs *observer.ObservedLogs, secret string) {
	t.Helper()
	msgs := loggedMessages(logs)
	for i := 0; i+8 <= len(secret); i++ {
		if strings.Contains(msgs, secret[i:i+8]) {
			t.Fatalf("fragment %q of %q leaked:\n%s", secret[i:i+8], secret, msgs)
		}
	}
}

// echoRouter answers every POST /api/x with the request body, under the
// request's own Content-Type (none when it sent none).
func echoRouter(limit int) *gin.Engine {
	router := gin.New()
	router.Use(DebugRequestBody(limit, nil))
	router.POST("/api/x", func(c *gin.Context) {
		got, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		if ct := c.GetHeader("Content-Type"); ct != "" {
			c.Header("Content-Type", ct)
		}
		c.Status(http.StatusOK)
		_, _ = c.Writer.Write(got)
	})
	return router
}

// TestDebugRequestBody_JSONWithoutJSONType covers bodies that are JSON but do
// not say so: fetch() sends a string body as text/plain;charset=UTF-8, and a
// client may send no Content-Type or no Content-Length (chunked). They are
// logged, redacted like application/json, on both sides.
func TestDebugRequestBody_JSONWithoutJSONType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const limit = 256
	body := `{"note":"visible-sniffed","client_secret":"` + sniffSecret + `","n":1}`
	want := `{"client_secret":"[REDACTED]","n":1,"note":"visible-sniffed"}`

	for _, tc := range []struct {
		name, ct string
		chunked  bool
	}{
		{"text/plain from fetch", "text/plain;charset=UTF-8", false},
		{"no content type", "", false},
		{"text/plain, Content-Length -1", "text/plain", true},
		{"no content type, Content-Length -1", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logs := observeDebugLogs(t)
			// Leading whitespace longer than one sniff read: the body is
			// still classified by its first non-space byte.
			sent := strings.Repeat(" \n", sniffLen) + body
			var r io.Reader = strings.NewReader(sent)
			if tc.chunked {
				r = io.MultiReader(r) // hides the length from http.NewRequest
			}
			req, _ := http.NewRequest(http.MethodPost, "/api/x", r)
			if tc.chunked {
				req.ContentLength = -1
			}
			if tc.ct != "" {
				req.Header.Set("Content-Type", tc.ct)
			}
			w := httptest.NewRecorder()
			echoRouter(4096).ServeHTTP(w, req)
			require.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, sent, w.Body.String(), "the body did not pass through unchanged")

			msgs := loggedMessages(logs)
			assert.Contains(t, msgs, "Request Body: "+want+"\n")
			assert.Contains(t, msgs, "Response Body: "+want+"\n")
			assertNoFragment(t, logs, sniffSecret)
		})
	}

	t.Run("cut by the limit", func(t *testing.T) {
		for _, tc := range []struct {
			name, ct string
			chunked  bool
		}{
			{"text/plain", "text/plain;charset=UTF-8", false},
			{"no content type, Content-Length -1", "", true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				logs := observeDebugLogs(t)
				// The secret straddles the limit; another lies wholly past it.
				head := `{"note":"visible-sniffed","pad":"`
				pad := strings.Repeat("p", limit-len(head)-len(`","api_key":"`)-len(sniffSecret)/2)
				sent := head + pad + `","api_key":"` + sniffSecret + `","later":{"password":"` + invalidSecret + `"},"more":"` +
					strings.Repeat("q", 2*limit) + `"}`
				start := strings.Index(sent, sniffSecret)
				require.Less(t, start, limit)
				require.Greater(t, start+len(sniffSecret), limit)

				var r io.Reader = strings.NewReader(sent)
				if tc.chunked {
					r = io.MultiReader(r)
				}
				req, _ := http.NewRequest(http.MethodPost, "/api/x", r)
				if tc.chunked {
					req.ContentLength = -1
				}
				if tc.ct != "" {
					req.Header.Set("Content-Type", tc.ct)
				}
				w := httptest.NewRecorder()
				echoRouter(limit).ServeHTTP(w, req)
				require.Equal(t, sent, w.Body.String())

				for _, prefix := range []string{"Request Body: ", "Response Body: "} {
					line, ok := loggedLine(logs, prefix)
					require.True(t, ok, "no %q line", prefix)
					assert.Contains(t, line, "visible-sniffed")
					assert.True(t, strings.HasSuffix(line, common.TruncatedBodyMarker), "not marked truncated: %s", line)
				}
				assertNoFragment(t, logs, sniffSecret)
				assertNoFragment(t, logs, invalidSecret)
			})
		}
	})

	t.Run("text/plain that starts like JSON but is not JSON", func(t *testing.T) {
		for _, sent := range []string{
			`{password=` + invalidSecret + `}`,
			`{ note: "x", password: "` + invalidSecret + `" }`,
			`{"note":"x"} ` + invalidSecret,
		} {
			for _, lim := range []int{limit, 12} { // whole, and cut by the limit
				logs := observeDebugLogs(t)
				req, _ := http.NewRequest(http.MethodPost, "/api/x", strings.NewReader(sent))
				req.Header.Set("Content-Type", "text/plain")
				w := httptest.NewRecorder()
				echoRouter(lim).ServeHTTP(w, req)
				require.Equal(t, sent, w.Body.String())
				_, ok := loggedLine(logs, "Request Body: ")
				assert.True(t, ok, "no request body line for %q", sent)
				assertNoFragment(t, logs, invalidSecret)
			}
		}
	})

	t.Run("plain text is still not logged", func(t *testing.T) {
		logs := observeDebugLogs(t)
		sent := "hello " + invalidSecret
		req, _ := http.NewRequest(http.MethodPost, "/api/x", strings.NewReader(sent))
		req.Header.Set("Content-Type", "text/plain")
		echoRouter(limit).ServeHTTP(httptest.NewRecorder(), req)
		msgs := loggedMessages(logs)
		assert.Contains(t, msgs, "Request Body: [26 bytes of text/plain not logged]")
		assert.Contains(t, msgs, "Response Body: [26 bytes of text/plain not logged]")
		assertNoFragment(t, logs, invalidSecret)
	})
}

func TestBodyLogWriter_Sniffing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	newWriter := func(w *httptest.ResponseRecorder, contentType string, limit int) *bodyLogWriter {
		blw := &bodyLogWriter{ResponseWriter: createTestResponseWriter(w), limit: limit}
		if contentType != "" {
			blw.Header().Set("Content-Type", contentType)
		}
		return blw
	}

	t.Run("decided at the first non-space byte, across writes", func(t *testing.T) {
		w := httptest.NewRecorder()
		blw := newWriter(w, "text/plain", 64)
		_, _ = blw.Write(nil)
		_, _ = blw.WriteString("  \n")
		_, _ = blw.Write([]byte(`{"token":"` + sniffSecret + `"}`))
		assert.Equal(t, `{"token":"[REDACTED]"}`, blw.render())
	})

	t.Run("non-JSON first byte keeps nothing", func(t *testing.T) {
		w := httptest.NewRecorder()
		blw := newWriter(w, "", 64)
		_, _ = blw.WriteString("\n")
		_, _ = blw.WriteString("<p>{}</p>")
		assert.Nil(t, blw.body)
		assert.Equal(t, "[10 bytes of unknown content type not logged]", blw.render())
	})

	t.Run("whitespace up to the limit keeps nothing", func(t *testing.T) {
		w := httptest.NewRecorder()
		blw := newWriter(w, "text/plain", 8)
		_, _ = blw.WriteString(strings.Repeat(" ", 20) + "{}")
		assert.Nil(t, blw.body)
		assert.Equal(t, "[22 bytes of text/plain not logged]", blw.render())
	})
}

// TestDebugRequestBody_LimitIsClampedAndNotPreallocated: a huge configured
// limit is clamped to config.MaxDebugBodyLogLimit (MaxInt used to panic in
// make), and a tiny response does not reserve the limit.
func TestDebugRequestBody_LimitIsClampedAndNotPreallocated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, configured := range []int{config.MaxDebugBodyLogLimit, 100 << 20, int(^uint(0) >> 1)} {
		observeDebugLogs(t)
		var blw *bodyLogWriter
		router := gin.New()
		router.Use(DebugRequestBody(configured, nil))
		router.POST("/api/x", func(c *gin.Context) {
			got, _ := io.ReadAll(c.Request.Body)
			blw = c.Writer.(*bodyLogWriter)
			c.Data(http.StatusOK, "application/json", got)
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/x", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, `{}`, w.Body.String())
		require.NotNil(t, blw)
		assert.Equal(t, config.MaxDebugBodyLogLimit, blw.limit, "configured %d", configured)
		assert.LessOrEqual(t, cap(blw.body), minCaptureCap, "a 2-byte response reserved %d bytes", cap(blw.body))
	}
}

func TestAppendCapped(t *testing.T) {
	var b []byte
	b = appendCapped(b, []byte("ab"), 1000)
	assert.Equal(t, minCaptureCap, cap(b))
	b = appendCapped(b, bytes.Repeat([]byte("x"), 600), 1000)
	assert.Equal(t, 1000, cap(b), "growth is capped at the limit")
	b = appendCapped(b, bytes.Repeat([]byte("y"), 398), 1000)
	assert.Len(t, b, 1000)
	assert.Equal(t, 1000, cap(b))
	assert.Equal(t, "ab", string(b[:2]))
}
