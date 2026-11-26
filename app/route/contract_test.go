package route

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/AnarManafov/dataharbor/app/common"
	"github.com/AnarManafov/dataharbor/app/config"
	"github.com/AnarManafov/dataharbor/app/controller"
	"github.com/AnarManafov/dataharbor/app/internal/apicontract"
)

// The tests in this file send requests through the real router and validate
// each response against api/openapi.yaml with the same validator the contract
// probe (make api-probe) uses. They cover every response that can be produced
// without an XRootD server or an identity provider; the rest is covered by the
// probe against a running stack.

// contractRouter builds the real router for the given auth mode and restores
// the previous configuration when the test ends.
func contractRouter(t *testing.T, authEnabled bool, upload config.UploadConfig) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	previous := config.GetConfig()
	t.Cleanup(func() { config.SetConfig(previous) })

	cfg := &config.Config{
		Env: "test",
		Server: config.ServerConfig{
			Address: "localhost:8080",
		},
		XRD: config.XRDConfig{
			Host:       "localhost",
			Port:       1094,
			InitialDir: "/tmp",
			Upload:     upload,
		},
		Auth: config.AuthConfig{
			Enabled: authEnabled,
		},
		Logging: config.LoggingConfig{
			Level: "info",
			Console: config.ConsoleConfig{
				Enabled: true,
				Format:  "text",
				Level:   "info",
			},
		},
	}
	config.SetConfig(cfg)
	common.InitLogger(&cfg.Logging)
	controller.InitAuth()

	r := gin.New()
	SetupRouter(r)
	return r
}

type contractCase struct {
	name   string
	method string
	path   string // concrete path, may carry a query string
	body   string
	ctype  string
	cookie string // raw Cookie header value
	status int    // expected HTTP status
}

func (c contractCase) run(t *testing.T, r *gin.Engine, spec *apicontract.Spec) {
	t.Helper()
	var body *bytes.Reader
	if c.body != "" {
		body = bytes.NewReader([]byte(c.body))
	} else {
		body = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(c.method, c.path, body)
	if c.ctype != "" {
		req.Header.Set("Content-Type", c.ctype)
	}
	if c.cookie != "" {
		req.Header.Set("Cookie", c.cookie)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, c.status, rec.Code, "unexpected status; body: %s", rec.Body.String())

	pathOnly, _, _ := strings.Cut(c.path, "?")
	err := spec.Validate(c.method, pathOnly, apicontract.Response{
		Status:      rec.Code,
		ContentType: rec.Header().Get("Content-Type"),
		Body:        rec.Body.Bytes(),
	})
	assert.NoError(t, err, "response does not match api/openapi.yaml; body: %s", rec.Body.String())
}

const (
	jsonType    = "application/json"
	bogusUpload = "/api/v1/xrd/upload/up_00000000000000000000000000000000"
)

// TestContract_AuthDisabled checks responses that need neither a session nor
// a storage server.
func TestContract_AuthDisabled(t *testing.T) {
	spec, err := apicontract.Load(openAPISpecPath)
	require.NoError(t, err)
	r := contractRouter(t, false, config.UploadConfig{})

	cases := []contractCase{
		{name: "health", method: "GET", path: "/health", status: 200},
		{name: "api health", method: "GET", path: "/api/health", status: 200},

		{name: "login, auth disabled", method: "GET", path: "/api/auth/login", status: 200},
		{name: "callback without code", method: "GET", path: "/api/auth/callback", status: 400},
		{name: "logout without session", method: "POST", path: "/api/auth/logout", status: 200},
		{name: "user without session", method: "GET", path: "/api/auth/user", status: 401},
		{name: "user with undecodable cookie", method: "GET", path: "/api/auth/user",
			cookie: "dataharbor-session=not-a-valid-session", status: 401},

		{name: "initialDir", method: "GET", path: "/api/v1/xrd/initialDir", status: 200},
		{name: "hostname", method: "GET", path: "/api/v1/xrd/hostname", status: 200},

		{name: "ls without dir", method: "GET", path: "/api/v1/xrd/ls", status: 400},
		{name: "paged ls, empty body", method: "POST", path: "/api/v1/xrd/ls/paged", status: 400},
		{name: "paged ls, not JSON", method: "POST", path: "/api/v1/xrd/ls/paged",
			body: "not json", ctype: jsonType, status: 400},
		{name: "paged ls, page 0", method: "POST", path: "/api/v1/xrd/ls/paged",
			body: `{"path":"/tmp","page":0}`, ctype: jsonType, status: 400},
		{name: "paged ls, empty path", method: "POST", path: "/api/v1/xrd/ls/paged",
			body: `{"path":"","page":1}`, ctype: jsonType, status: 400},

		{name: "download without path", method: "GET", path: "/api/v1/xrd/download", status: 400},
		{name: "download, relative path", method: "GET", path: "/api/v1/xrd/download?path=relative/file", status: 400},
		{name: "download, traversal", method: "GET", path: "/api/v1/xrd/download?path=/a/../b", status: 400},

		{name: "batch, empty body", method: "POST", path: "/api/v1/xrd/download/batch", status: 400},
		{name: "batch, no files", method: "POST", path: "/api/v1/xrd/download/batch",
			body: `{"basePath":"/tmp","files":[]}`, ctype: jsonType, status: 400},
		{name: "batch, bad base path", method: "POST", path: "/api/v1/xrd/download/batch",
			body: `{"basePath":"tmp","files":["a"]}`, ctype: jsonType, status: 400},
		{name: "batch, bad file name", method: "POST", path: "/api/v1/xrd/download/batch",
			body: `{"basePath":"/tmp","files":["a/b"]}`, ctype: jsonType, status: 400},
		{name: "batch, form without files", method: "POST", path: "/api/v1/xrd/download/batch",
			body: "basePath=%2Ftmp", ctype: "application/x-www-form-urlencoded", status: 400},

		{name: "upload limits", method: "GET", path: "/api/v1/xrd/upload/limits", status: 200},
		{name: "upload session, uploads disabled", method: "POST", path: "/api/v1/xrd/upload/session",
			body: `{"destDir":"/tmp","files":[{"relPath":"a","size":1}]}`, ctype: jsonType, status: 503},
		{name: "chunk, unknown id", method: "PUT", path: bogusUpload + "/chunk?offset=0",
			body: "x", ctype: "application/octet-stream", status: 404},
		{name: "complete, unknown id", method: "POST", path: bogusUpload + "/complete",
			body: `{"sha256":"00"}`, ctype: jsonType, status: 404},
		{name: "status, unknown id", method: "GET", path: bogusUpload + "/status", status: 404},
		{name: "abort, unknown id", method: "DELETE", path: bogusUpload, status: 404},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { c.run(t, r, spec) })
	}
}

// TestContract_UploadSessionValidation checks the 400 responses of session
// creation, which are only reachable with uploads enabled.
func TestContract_UploadSessionValidation(t *testing.T) {
	spec, err := apicontract.Load(openAPISpecPath)
	require.NoError(t, err)
	r := contractRouter(t, false, config.UploadConfig{
		Enabled:          true,
		MaxFileSize:      1 << 20,
		MaxBatchSize:     1 << 20,
		MaxFilesPerBatch: 2,
		ChunkSize:        1 << 10,
		TempSuffix:       ".dh-upload",
	})

	const sessionPath = "/api/v1/xrd/upload/session"
	cases := []contractCase{
		{name: "limits, uploads enabled", method: "GET", path: "/api/v1/xrd/upload/limits", status: 200},
		{name: "empty body", method: "POST", path: sessionPath, status: 400},
		{name: "relative destDir", method: "POST", path: sessionPath,
			body: `{"destDir":"tmp","files":[{"relPath":"a","size":1}]}`, ctype: jsonType, status: 400},
		{name: "empty files", method: "POST", path: sessionPath,
			body: `{"destDir":"/tmp","files":[]}`, ctype: jsonType, status: 400},
		{name: "too many files", method: "POST", path: sessionPath,
			body:  `{"destDir":"/tmp","files":[{"relPath":"a","size":1},{"relPath":"b","size":1},{"relPath":"c","size":1}]}`,
			ctype: jsonType, status: 400},
		{name: "bad relPath", method: "POST", path: sessionPath,
			body: `{"destDir":"/tmp","files":[{"relPath":"../a","size":1}]}`, ctype: jsonType, status: 400},
		{name: "file too large", method: "POST", path: sessionPath,
			body: `{"destDir":"/tmp","files":[{"relPath":"a","size":2097152}]}`, ctype: jsonType, status: 400},
		{name: "unknown onConflict", method: "POST", path: sessionPath,
			body: `{"destDir":"/tmp","files":[{"relPath":"a","size":1,"onConflict":"maybe"}]}`, ctype: jsonType, status: 400},
		{name: "overwrite not allowed", method: "POST", path: sessionPath,
			body: `{"destDir":"/tmp","files":[{"relPath":"a","size":1,"onConflict":"overwrite"}]}`, ctype: jsonType, status: 403},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { c.run(t, r, spec) })
	}
}

// TestContract_ProtectedRoutesRequireSession checks that, with auth enabled,
// every operation the specification declares as protected (by its `security`)
// answers 401 in the documented shape, and that the public ones do not.
func TestContract_ProtectedRoutesRequireSession(t *testing.T) {
	spec, err := apicontract.Load(openAPISpecPath)
	require.NoError(t, err)
	r := contractRouter(t, true, config.UploadConfig{})

	public := anonymousOperations(t)
	require.True(t, public["GET /health"], "GET /health must be declared public in the specification")

	protected := 0
	for _, op := range spec.Operations() {
		key := op.Method + " " + op.Path
		if public[key] {
			continue
		}
		protected++
		concrete := strings.ReplaceAll(op.Path, "{uploadId}", "up_00000000000000000000000000000000")
		for _, variant := range []struct{ name, cookie string }{
			{"no cookie", ""},
			{"undecodable cookie", "dataharbor-session=not-a-valid-session"},
		} {
			t.Run(key+", "+variant.name, func(t *testing.T) {
				contractCase{method: op.Method, path: concrete, cookie: variant.cookie, status: 401}.run(t, r, spec)
			})
		}
	}
	assert.Equal(t, 15, protected, "number of protected operations changed; update this test and the specification together")

	for _, c := range []contractCase{
		{name: "health stays public", method: "GET", path: "/health", status: 200},
		{name: "api health stays public", method: "GET", path: "/api/health", status: 200},
		{name: "callback stays public", method: "GET", path: "/api/auth/callback", status: 400},
		{name: "logout stays public", method: "POST", path: "/api/auth/logout", status: 200},
	} {
		t.Run(c.name, func(t *testing.T) { c.run(t, r, spec) })
	}
}

// anonymousOperations returns "METHOD /path" for every operation whose
// effective `security` in the specification allows a request without a
// session: an empty list, or a list with an empty requirement ({}).
func anonymousOperations(t *testing.T) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile(openAPISpecPath)
	require.NoError(t, err)
	out, err := parseAnonymousOperations(raw)
	require.NoError(t, err)
	return out
}

// isOpenAPIMethod reports whether a path-item key names an operation, as
// opposed to a path-level field such as `parameters`, `summary` or `servers`.
func isOpenAPIMethod(key string) bool {
	switch key {
	case "get", "put", "post", "delete", "options", "head", "patch", "trace":
		return true
	}
	return false
}

// parseAnonymousOperations is anonymousOperations on an in-memory document.
// Path items are decoded leniently and only their HTTP method keys are read,
// so path-level fields neither break decoding nor count as operations.
func parseAnonymousOperations(raw []byte) (map[string]bool, error) {
	type requirements = []map[string][]string
	var doc struct {
		Security requirements                    `yaml:"security"`
		Paths    map[string]map[string]yaml.Node `yaml:"paths"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}

	anonymous := func(reqs requirements) bool {
		if len(reqs) == 0 {
			return true
		}
		for _, r := range reqs {
			if len(r) == 0 {
				return true
			}
		}
		return false
	}
	out := map[string]bool{}
	for path, item := range doc.Paths {
		for method, node := range item {
			if !isOpenAPIMethod(method) {
				continue
			}
			var op struct {
				Security *requirements `yaml:"security"`
			}
			if err := node.Decode(&op); err != nil {
				return nil, fmt.Errorf("%s %s: %w", strings.ToUpper(method), path, err)
			}
			effective := doc.Security
			if op.Security != nil {
				effective = *op.Security
			}
			if anonymous(effective) {
				out[strings.ToUpper(method)+" "+path] = true
			}
		}
	}
	return out, nil
}

// TestParseAnonymousOperations covers path items that carry path-level fields
// next to their operations, which api/openapi.yaml does not use today.
func TestParseAnonymousOperations(t *testing.T) {
	const doc = `
openapi: 3.1.0
security:
  - cookieAuth: []
paths:
  /items/{id}:
    summary: One item
    description: Path-level fields are not operations.
    parameters:
      - name: id
        in: path
        required: true
        schema: {type: string}
    get:
      responses: {"200": {description: ok}}
    delete:
      security: []
      responses: {"204": {description: gone}}
  /mixed:
    post:
      security:
        - {}
        - cookieAuth: []
      responses: {"200": {description: ok}}
    put:
      security:
        - cookieAuth: []
      responses: {"200": {description: ok}}
`
	got, err := parseAnonymousOperations([]byte(doc))
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{
		"DELETE /items/{id}": true,
		"POST /mixed":        true,
	}, got)
}

// TestContract_UnknownAPIPath checks the 404 documented as
// components.responses.ApiNotFound, which belongs to no operation.
func TestContract_UnknownAPIPath(t *testing.T) {
	r := contractRouter(t, true, config.UploadConfig{})
	for _, c := range []struct{ method, path string }{
		{"GET", "/api/v1/does-not-exist"},
		{"GET", "/api/auth/logout"}, // wrong method on an existing path: 404, not 405
	} {
		req := httptest.NewRequest(c.method, c.path, nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code, "%s %s", c.method, c.path)
		assert.JSONEq(t, `{"message":"API endpoint not found"}`, rec.Body.String(), "%s %s", c.method, c.path)
	}
}

// TestContract_DetectsWrongSchema proves the checks above have teeth: when a
// schema in the specification is deliberately made wrong, a real response no
// longer validates.
func TestContract_DetectsWrongSchema(t *testing.T) {
	raw, err := os.ReadFile(openAPISpecPath)
	require.NoError(t, err)

	mutations := []struct {
		name, old, new string
		method, path   string
	}{
		{"health data constant", "data: {type: string, const: ok}", "data: {type: string, const: broken}", "GET", "/health"},
		{"initialDir data type", "data: {type: string}", "data: {type: integer}", "GET", "/api/v1/xrd/initialDir"},
		{"limits field renamed", "maxBatchFiles:", "maxBatchFilez:", "GET", "/api/v1/xrd/upload/limits"},
	}
	r := contractRouter(t, false, config.UploadConfig{})
	for _, m := range mutations {
		t.Run(m.name, func(t *testing.T) {
			require.Contains(t, string(raw), m.old, "the specification no longer contains the text this test mutates")
			broken, err := apicontract.Parse([]byte(strings.ReplaceAll(string(raw), m.old, m.new)))
			require.NoError(t, err)

			req := httptest.NewRequest(m.method, m.path, nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			require.Equal(t, http.StatusOK, rec.Code)

			err = broken.Validate(m.method, m.path, apicontract.Response{
				Status:      rec.Code,
				ContentType: rec.Header().Get("Content-Type"),
				Body:        rec.Body.Bytes(),
			})
			require.Error(t, err, "a deliberately wrong schema must be detected")
			var contractErr *apicontract.Error
			require.ErrorAs(t, err, &contractErr)
			assert.Equal(t, apicontract.KindSchema, contractErr.Kind)
		})
	}
}
