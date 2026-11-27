package route

import (
	"bytes"
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
	"github.com/AnarManafov/dataharbor/app/controller"
)

// transferHandlers are the handlers whose request or response body is a file.
// Every route that serves one must be registered with handleStreaming.
var transferHandlers = []string{
	"controller.DownloadFile",
	"controller.DownloadMultipleFiles",
	"controller.UploadChunk",
}

// TestStreamingRoutesMatchRegisteredRoutes guards against drift between the
// streaming set the debug middleware consults and the routes SetupRouter
// registers: every marked route must exist exactly as gin reports it (method
// and pattern, i.e. what c.FullPath() returns), and every route served by a
// transfer handler must be marked.
func TestStreamingRoutesMatchRegisteredRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetupRouter(r)

	registered := map[string]bool{}
	for _, route := range r.Routes() {
		registered[route.Method+" "+route.Path] = true
	}
	streaming := map[string]bool{}
	for _, key := range StreamingRoutes() {
		streaming[key] = true
		assert.True(t, registered[key], "streaming route %q is not registered", key)
	}

	for _, key := range []string{
		"GET /api/v1/xrd/download",
		"POST /api/v1/xrd/download/batch",
		"PUT /api/v1/xrd/upload/:uploadId/chunk",
	} {
		assert.True(t, streaming[key], "transfer route %q is not marked streaming", key)
	}

	served := 0
	for _, route := range r.Routes() {
		for _, h := range transferHandlers {
			if strings.HasSuffix(route.Handler, h) {
				served++
				assert.True(t, streaming[route.Method+" "+route.Path],
					"%s %s is served by %s but not registered with handleStreaming", route.Method, route.Path, h)
			}
		}
	}
	assert.Equal(t, len(transferHandlers), served, "a transfer handler is not registered (or its name changed)")
}

// debugRouter builds the real router with debug logging on, auth enabled (so
// protected routes answer 401 without a session) and a small body limit, and
// captures the debug log.
func debugRouter(t *testing.T, limit int) (*gin.Engine, *observer.ObservedLogs) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	previous := config.GetConfig()
	t.Cleanup(func() { config.SetConfig(previous) })
	cfg := &config.Config{
		Env:    "test",
		Server: config.ServerConfig{Address: "localhost:8080", Debug: true, DebugBodyLogLimit: limit},
		XRD:    config.XRDConfig{Host: "localhost", Port: 1094, InitialDir: "/tmp"},
		Auth:   config.AuthConfig{Enabled: true},
		Logging: config.LoggingConfig{
			Level:   "info",
			Console: config.ConsoleConfig{Enabled: true, Format: "text", Level: "info"},
		},
	}
	config.SetConfig(cfg)
	controller.InitAuth()

	core, logs := observer.New(zapcore.DebugLevel)
	prevLogger := common.Logger
	common.Logger = zap.New(core).Sugar()
	t.Cleanup(func() { common.Logger = prevLogger })

	r := gin.New()
	SetupRouter(r)
	return r, logs
}

// countingBody records how many bytes were read from it.
type countingBody struct {
	r io.Reader
	n int
}

func (b *countingBody) Read(p []byte) (int, error) {
	n, err := b.r.Read(p)
	b.n += n
	return n, err
}

func (b *countingBody) Close() error { return nil }

// requestLog returns the debug lines logged for the request whose request
// line is "Request: <method> <path>".
func requestLog(logs *observer.ObservedLogs, method, path string) string {
	var b strings.Builder
	in := false
	for _, e := range logs.All() {
		if strings.HasPrefix(e.Message, "Request: ") {
			in = e.Message == "Request: "+method+" "+path
		}
		if in {
			b.WriteString(e.Message)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// TestDebugBodiesOnRealRouter drives the real router: the debug middleware is
// global, so this proves it sees the registered route pattern (c.FullPath())
// and keys its decision on it.
func TestDebugBodiesOnRealRouter(t *testing.T) {
	const limit = 128
	r, logs := debugRouter(t, limit)
	big := []byte(`{"note":"visible-head","pad":"` + strings.Repeat("x", 1<<20) + `"}`)

	t.Run("upload chunk body is never read", func(t *testing.T) {
		body := &countingBody{r: bytes.NewReader(big)}
		req := httptest.NewRequest(http.MethodPut, "/api/v1/xrd/upload/up_probe/chunk", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusUnauthorized, w.Code)

		assert.Equal(t, 0, body.n, "the chunk body was read")
		lines := requestLog(logs, http.MethodPut, "/api/v1/xrd/upload/up_probe/chunk")
		assert.Contains(t, lines, "Response Status: 401")
		assert.NotContains(t, lines, "Request Body:")
		assert.NotContains(t, lines, "Response Body:")
	})

	t.Run("download response is not captured", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/xrd/download?path=/x", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusUnauthorized, w.Code)

		lines := requestLog(logs, http.MethodGet, "/api/v1/xrd/download")
		assert.Contains(t, lines, "Response Status: 401")
		assert.NotContains(t, lines, "Response Body:")
	})

	t.Run("other routes log a truncated body", func(t *testing.T) {
		body := &countingBody{r: bytes.NewReader(big)}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/xrd/upload/up_probe/complete", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusUnauthorized, w.Code)

		assert.LessOrEqual(t, body.n, limit+1, "the middleware read past its limit")
		lines := requestLog(logs, http.MethodPost, "/api/v1/xrd/upload/up_probe/complete")
		assert.Contains(t, lines, `Request Body: {"note":"visible-head"`+common.TruncatedBodyMarker)
		assert.Contains(t, lines, "Response Body: {")
	})
}
