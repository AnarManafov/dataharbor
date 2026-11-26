package apiprobe

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AnarManafov/dataharbor/app/internal/apicontract"
)

// testCookie is distinctive and contains characters that URL encoding
// changes, so both the raw and the encoded forms are exercised.
const testCookie = "PROBE-SECRET-3f9a/+=MTc2NDA5NjAwMHxEdi1CQkFFQ180SUFBUkFCRUFBQV9-redact-me"

const specPath = "testdata/spec.yaml"

// fakeBackend imitates the DataHarbor API closely enough for the probe.
type fakeBackend struct {
	t           *testing.T
	authEnabled bool
	noFiles     bool
	// badNames adds files smaller than small.txt whose names the server
	// would reject for download.
	badNames bool
	// busy429 answers this many download requests (single or batch) with
	// the 429 of the per-user download limit before serving them.
	busy429 int

	mu            sync.Mutex
	gatewayHealth bool     // GET /health answered like the nginx gateway
	brokenHealth  bool     // health endpoints answer JSON of the wrong shape
	requests      []string // "METHOD /path"
	cookieSeen    int      // requests that carried the right session cookie
	wrongCookie   int      // requests that carried a session cookie with another value
	uploads       map[string]*fakeUpload
	completedData map[string][]byte
}

type fakeUpload struct {
	name string
	size int
	data []byte
}

func newFakeBackend(t *testing.T) *fakeBackend {
	return &fakeBackend{t: t, authEnabled: true, uploads: map[string]*fakeUpload{}, completedData: map[string][]byte{}}
}

func (f *fakeBackend) requested(prefix string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, r := range f.requests {
		if strings.HasPrefix(r, prefix) {
			out = append(out, r)
		}
	}
	return out
}

// takeBusy consumes one queued 429 answer.
func (f *fakeBackend) takeBusy() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.busy429 <= 0 {
		return false
	}
	f.busy429--
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func ok(w http.ResponseWriter, data any) {
	writeJSON(w, 200, map[string]any{"code": 200, "data": data, "message": "success"})
}

func errJSON(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"code": status, "error": msg})
}

func (f *fakeBackend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.requests = append(f.requests, r.Method+" "+r.URL.Path)
	authed := false
	if c, err := r.Cookie(SessionCookieName); err == nil {
		if c.Value == testCookie {
			f.cookieSeen++
			authed = true
		} else {
			f.wrongCookie++
		}
	}
	f.mu.Unlock()

	// Echo the cookie back in a Set-Cookie header on every response, raw and
	// encoded, the way a refreshed session would; the probe must not print it.
	w.Header().Add("Set-Cookie", SessionCookieName+"="+testCookie+"; Path=/; HttpOnly")
	w.Header().Add("X-Echo", url.QueryEscape(testCookie))

	p := r.URL.Path
	switch {
	case r.Method == "GET" && p == "/health" && f.gatewayHealth:
		// What the nginx gateway of the docker deployments does.
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte("healthy\n"))
		return
	case r.Method == "GET" && (p == "/health" || p == "/api/health"):
		if f.brokenHealth {
			writeJSON(w, 200, map[string]any{"status": "up"})
			return
		}
		ok(w, "ok")
		return
	case r.Method == "GET" && p == "/api/auth/login":
		if !f.authEnabled {
			writeJSON(w, 200, map[string]any{"auth_url": "/auth/disabled", "message": "Authentication is disabled"})
			return
		}
		writeJSON(w, 200, map[string]any{"auth_url": "https://idp.example/auth?state=x"})
		return
	case r.Method == "POST" && p == "/api/auth/logout":
		f.t.Errorf("probe called logout")
		writeJSON(w, 200, map[string]any{"success": true})
		return
	case strings.HasPrefix(p, "/api/auth/callback"):
		f.t.Errorf("probe called the OIDC callback")
		errJSON(w, 400, "no")
		return
	case r.Method == "GET" && p == "/api/auth/user":
		if !authed || !f.authEnabled {
			errJSON(w, 401, "Not authenticated")
			return
		}
		// Echo the cookie (raw and URL-encoded) in the body.
		writeJSON(w, 200, map[string]any{"sub": "user-1", "echo": testCookie, "echoEnc": url.QueryEscape(testCookie)})
		return
	}

	if !strings.HasPrefix(p, "/api/v1/xrd/") {
		errJSON(w, 404, "API endpoint not found")
		return
	}
	if f.authEnabled && !authed {
		errJSON(w, 401, "Not authenticated")
		return
	}
	f.serveXRD(w, r, strings.TrimPrefix(p, "/api/v1/xrd"))
}

func (f *fakeBackend) serveXRD(w http.ResponseWriter, r *http.Request, p string) {
	switch {
	case r.Method == "GET" && p == "/initialDir":
		ok(w, "/data")
	case r.Method == "GET" && p == "/hostname":
		// Pad so the cookie straddles the 200-byte excerpt cap of a FAIL line.
		ok(w, strings.Repeat("h", 170)+testCookie)
	case r.Method == "GET" && p == "/ping":
		ok(w, map[string]any{"latencyMs": 1.5, "connectMs": 0.5, "status": "ok", "server": "xrd"})
	case r.Method == "GET" && p == "/vstat":
		ok(w, map[string]any{"freeSpaceMB": 10})
	case r.Method == "GET" && p == "/ls":
		entries := []map[string]any{{"name": "sub", "isDir": true, "size": 0, "mtime": 1}}
		if !f.noFiles {
			entries = append(entries,
				map[string]any{"name": "big.bin", "isDir": false, "size": 1 << 20, "mtime": 1},
				map[string]any{"name": "small.txt", "isDir": false, "size": 5, "mtime": 1})
		}
		if f.badNames {
			for _, n := range []string{"a..b", `back\slash`, "cr\rname", "lf\nname", "nul\x00name", "..", "."} {
				entries = append(entries, map[string]any{"name": n, "isDir": false, "size": 1, "mtime": 1})
			}
		}
		writeJSON(w, 200, map[string]any{"directory": r.URL.Query().Get("dir"), "entries": entries})
	case r.Method == "POST" && p == "/ls/paged":
		writeJSON(w, 200, map[string]any{"code": 200, "items": []any{}, "totalItems": 0})
	case (p == "/download" || p == "/download/batch") && f.takeBusy():
		errJSON(w, 429, "Too many concurrent downloads. Please wait for a current download to complete.")
	case r.Method == "GET" && p == "/download":
		if r.URL.Query().Get("path") != "/data/small.txt" {
			errJSON(w, 404, "File not found or inaccessible")
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(200)
		_, _ = w.Write(bytes.Repeat([]byte("A"), 64<<10)) // more than the probe reads
	case r.Method == "POST" && p == "/download/batch":
		var req struct {
			BasePath string   `json:"basePath"`
			Files    []string `json:"files"`
		}
		if json.NewDecoder(r.Body).Decode(&req) != nil || req.BasePath != "/data" || len(req.Files) != 1 || req.Files[0] != "small.txt" {
			errJSON(w, 400, "Invalid request body")
			return
		}
		w.Header().Set("Content-Type", "application/x-tar")
		w.WriteHeader(200)
		_, _ = w.Write(make([]byte, 10240))
	case r.Method == "GET" && p == "/upload/limits":
		ok(w, map[string]any{"upload": map[string]any{"enabled": true, "chunkSize": 16}, "download": map[string]any{}})
	case r.Method == "POST" && p == "/upload/session":
		var req struct {
			DestDir string `json:"destDir"`
			Files   []struct {
				RelPath string `json:"relPath"`
				Size    int    `json:"size"`
			} `json:"files"`
		}
		if json.NewDecoder(r.Body).Decode(&req) != nil || req.DestDir != "/scratch" || len(req.Files) != 1 || req.Files[0].Size <= 0 {
			errJSON(w, 400, "Invalid request")
			return
		}
		f.mu.Lock()
		id := fmt.Sprintf("up-%d", len(f.uploads)+1)
		f.uploads[id] = &fakeUpload{name: req.Files[0].RelPath, size: req.Files[0].Size}
		f.mu.Unlock()
		ok(w, map[string]any{"sessionId": "s", "chunkSize": 16, "files": []any{
			map[string]any{"relPath": req.Files[0].RelPath, "uploadId": id, "status": "accepted", "conflict": "none", "destPath": "/scratch/" + req.Files[0].RelPath},
		}})
	case strings.HasPrefix(p, "/upload/"):
		f.serveUpload(w, r, strings.TrimPrefix(p, "/upload/"))
	default:
		errJSON(w, 404, "API endpoint not found")
	}
}

func (f *fakeBackend) serveUpload(w http.ResponseWriter, r *http.Request, rest string) {
	id, action, _ := strings.Cut(rest, "/")
	f.mu.Lock()
	defer f.mu.Unlock()
	up, found := f.uploads[id]
	if !found {
		errJSON(w, 404, "Unknown uploadId")
		return
	}
	switch {
	case r.Method == "PUT" && action == "chunk":
		body, _ := io.ReadAll(r.Body)
		if r.URL.Query().Get("offset") != fmt.Sprint(len(up.data)) || r.ContentLength != int64(len(body)) || len(body) > 16 {
			errJSON(w, 409, "offset mismatch")
			return
		}
		up.data = append(up.data, body...)
		ok(w, map[string]any{"bytesReceived": len(up.data)})
	case r.Method == "GET" && action == "status":
		ok(w, map[string]any{"uploadId": id, "bytesReceived": len(up.data), "state": "uploading", "size": up.size})
	case r.Method == "POST" && action == "complete":
		var req struct {
			Sha256 string `json:"sha256"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		sum := sha256.Sum256(up.data)
		if len(up.data) != up.size || req.Sha256 != hex.EncodeToString(sum[:]) {
			errJSON(w, 400, "checksum mismatch")
			return
		}
		f.completedData[up.name] = up.data
		ok(w, map[string]any{"uploadId": id, "destPath": "/scratch/" + up.name, "sha256": req.Sha256, "size": up.size})
	case r.Method == "DELETE" && action == "":
		delete(f.uploads, id)
		ok(w, map[string]any{"uploadId": id, "state": "aborted"})
	default:
		errJSON(w, 404, "API endpoint not found")
	}
}

type runResult struct {
	code           int
	stdout, stderr string
}

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func runProbe(t *testing.T, baseURL string, extraEnv map[string]string, args ...string) runResult {
	t.Helper()
	e := map[string]string{EnvBaseURL: baseURL, EnvCookie: testCookie}
	for k, v := range extraEnv {
		e[k] = v
	}
	var out, errOut bytes.Buffer
	code := Main(args, env(e), &out, &errOut)
	return runResult{code: code, stdout: out.String(), stderr: errOut.String()}
}

// assertNoCookie checks that neither the cookie nor any recognisable part of
// it reached the output, in raw or URL-encoded form.
func assertNoCookie(t *testing.T, r runResult) {
	t.Helper()
	all := r.stdout + r.stderr
	for _, s := range []string{testCookie, url.QueryEscape(testCookie), url.PathEscape(testCookie), testCookie[:14], "redact-me", "MTc2NDA5NjAw"} {
		assert.NotContains(t, all, s, "output leaks the session cookie")
	}
}

var lineRE = regexp.MustCompile(`^(PASS|FAIL|SKIP)  (GET|POST|PUT|DELETE|PATCH) /\S*  (\d{3}|-)(  .+)?$`)

// assertFormat checks that every line but the summary is a check line.
func assertFormat(t *testing.T, stdout string) []string {
	t.Helper()
	lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
	require.NotEmpty(t, lines)
	for _, l := range lines[:len(lines)-1] {
		assert.Regexp(t, lineRE, l)
	}
	assert.Regexp(t, `^SUMMARY  pass=\d+ fail=\d+ skip=\d+$`, lines[len(lines)-1])
	return lines
}

func hasLine(lines []string, prefix string) bool {
	for _, l := range lines {
		if strings.HasPrefix(l, prefix) {
			return true
		}
	}
	return false
}

func TestProbe_AllPass_TLS(t *testing.T) {
	fb := newFakeBackend(t)
	srv := httptest.NewTLSServer(fb)
	defer srv.Close()

	r := runProbe(t, srv.URL, map[string]string{EnvInsecure: "1"}, "-spec", specPath)
	assert.Equal(t, ExitOK, r.code, r.stdout+r.stderr)
	assert.Empty(t, r.stderr)
	assert.NotContains(t, r.stdout, "FAIL")
	lines := assertFormat(t, r.stdout)
	assertNoCookie(t, r)

	for _, want := range []string{
		"PASS  GET /health  200",
		"PASS  GET /api/health  200",
		"PASS  GET /api/auth/login  200  without session cookie",
		"PASS  GET /api/auth/user  200",
		"PASS  GET /api/auth/user  401  without session cookie",
		"PASS  GET /api/v1/xrd/initialDir  401  without session cookie",
		"PASS  GET /api/v1/xrd/initialDir  200",
		"PASS  GET /api/v1/xrd/hostname  200",
		"PASS  GET /api/v1/xrd/ping  200",
		"PASS  GET /api/v1/xrd/vstat  200",
		"PASS  GET /api/v1/xrd/ls  200",
		"PASS  POST /api/v1/xrd/ls/paged  200",
		"PASS  GET /api/v1/xrd/upload/limits  200",
		"PASS  GET /api/v1/xrd/upload/{uploadId}/status  404  unknown uploadId",
		"PASS  GET /api/v1/xrd/download  200",
		"PASS  POST /api/v1/xrd/download/batch  200",
		"SKIP  POST /api/auth/logout  -  never called",
		"SKIP  GET /api/auth/callback  -  never called",
		"SKIP  POST /api/v1/xrd/upload/session  -  upload checks run only with -upload and -scratch-dir",
		"SKIP  PUT /api/v1/xrd/upload/{uploadId}/chunk  -  upload checks",
		"SKIP  POST /api/v1/xrd/upload/{uploadId}/complete  -  upload checks",
		"SKIP  DELETE /api/v1/xrd/upload/{uploadId}  -  upload checks",
	} {
		assert.True(t, hasLine(lines, want), "missing line %q in:\n%s", want, r.stdout)
	}
	assert.Equal(t, "SUMMARY  pass=16 fail=0 skip=6", lines[len(lines)-1])

	// The smallest file was chosen and the server saw the cookie only as a
	// request cookie with the right value.
	assert.Contains(t, r.stdout, "smallest file (5 bytes)")
	assert.Greater(t, fb.cookieSeen, 10)
	assert.Zero(t, fb.wrongCookie)
	assert.Empty(t, fb.requested("POST /api/auth/logout"))
	assert.Empty(t, fb.requested("GET /api/auth/callback"))
	assert.Empty(t, fb.requested("POST /api/v1/xrd/upload/session"))
	assert.Empty(t, fb.requested("PUT "))
	assert.Empty(t, fb.requested("DELETE "))
}

func TestProbe_AllPass_PlainHTTP_EverySpecOperationAccountedFor(t *testing.T) {
	fb := newFakeBackend(t)
	srv := httptest.NewServer(fb)
	defer srv.Close()

	r := runProbe(t, srv.URL+"/", nil, "-spec", specPath) // trailing slash tolerated
	require.Equal(t, ExitOK, r.code, r.stdout+r.stderr)
	lines := assertFormat(t, r.stdout)

	// Every operation of the spec appears on at least one line.
	spec, err := apicontract.Load(specPath)
	require.NoError(t, err)
	ops := spec.Operations()
	require.Len(t, ops, 20)
	for _, op := range ops {
		needle := " " + op.Method + " " + op.Path + "  "
		assert.True(t, strings.Contains(r.stdout, needle), "operation %s not accounted for", needle)
	}
	assert.Len(t, lines, 16+6+1) // PASS lines, SKIP lines, summary
}

func TestProbe_HealthAnsweredByGatewayIsSkipped(t *testing.T) {
	fb := newFakeBackend(t)
	fb.gatewayHealth = true
	srv := httptest.NewServer(fb)
	defer srv.Close()

	r := runProbe(t, srv.URL, nil, "-spec", specPath)
	assert.Equal(t, ExitOK, r.code, r.stdout+r.stderr)
	lines := assertFormat(t, r.stdout)
	assertNoCookie(t, r)
	assert.True(t, hasLine(lines, "SKIP  GET /health  -  answered by a gateway"), r.stdout)
	assert.True(t, hasLine(lines, "PASS  GET /api/health  200"), r.stdout)
	assert.Equal(t, "SUMMARY  pass=15 fail=0 skip=7", lines[len(lines)-1])
}

// A JSON answer of the wrong shape is a contract violation, not a gateway.
func TestProbe_WrongHealthJSONStillFails(t *testing.T) {
	fb := newFakeBackend(t)
	fb.brokenHealth = true
	srv := httptest.NewServer(fb)
	defer srv.Close()

	r := runProbe(t, srv.URL, nil, "-spec", specPath)
	assert.Equal(t, ExitMismatch, r.code, r.stdout+r.stderr)
	lines := assertFormat(t, r.stdout)
	assert.True(t, hasLine(lines, "FAIL  GET /health  200"), r.stdout)
	assert.True(t, hasLine(lines, "FAIL  GET /api/health  200"), r.stdout)
}

func TestProbe_MutatedSchemaFails(t *testing.T) {
	fb := newFakeBackend(t)
	srv := httptest.NewServer(fb)
	defer srv.Close()

	data, err := os.ReadFile(specPath)
	require.NoError(t, err)
	// Break the shared string envelope: data must now be an integer. This hits
	// /health, /api/health, initialDir and hostname (whose body echoes the cookie).
	mutated := strings.Replace(string(data), "              data: {type: string}\n              message: {type: string}",
		"              data: {type: integer}\n              message: {type: string}", 1)
	require.NotEqual(t, string(data), mutated)
	bad := filepath.Join(t.TempDir(), "openapi.yaml")
	require.NoError(t, os.WriteFile(bad, []byte(mutated), 0o600))

	r := runProbe(t, srv.URL, nil, "-spec", bad)
	assert.Equal(t, ExitMismatch, r.code)
	lines := assertFormat(t, r.stdout)
	assertNoCookie(t, r)

	var healthFail string
	for _, l := range lines {
		if strings.HasPrefix(l, "FAIL  GET /health  200  ") {
			healthFail = l
		}
	}
	require.NotEmpty(t, healthFail, r.stdout)
	assert.Contains(t, healthFail, "schema violation")
	assert.Contains(t, healthFail, "at /data (schema #/components/responses/StringOK/content/application~1json/schema/properties/data/type)")
	assert.Contains(t, healthFail, `[body: {"code":200,"data":"ok","message":"success"}\n]`)

	// The hostname body is long and carries the cookie across the excerpt cap.
	assert.True(t, hasLine(lines, "FAIL  GET /api/v1/xrd/hostname  200  "), r.stdout)
	assert.Contains(t, r.stdout, redacted)
	for _, l := range lines {
		if strings.HasPrefix(l, "FAIL  GET /api/v1/xrd/hostname") {
			_, body, found := strings.Cut(l, "[body: ")
			require.True(t, found)
			assert.LessOrEqual(t, len(body), excerptLen+len("...]"))
		}
	}
	// initialDir broke, so the directory-dependent checks are skipped with reason.
	assert.True(t, hasLine(lines, "SKIP  GET /api/v1/xrd/ls  -  initial directory unknown"), r.stdout)
}

func TestProbe_WrongStatusFails(t *testing.T) {
	fb := newFakeBackend(t)
	srv := httptest.NewServer(fb)
	defer srv.Close()

	// A cookie the server does not accept: user and protected routes answer a
	// documented 401, which is contract-valid but not the expected status.
	r := runProbe(t, srv.URL, map[string]string{EnvCookie: "other-cookie"}, "-spec", specPath)
	assert.Equal(t, ExitMismatch, r.code)
	assert.Contains(t, r.stdout, "FAIL  GET /api/auth/user  401  expected status 200")
	assert.NotContains(t, r.stdout, "other-cookie")
}

func TestProbe_UndocumentedOperationFails(t *testing.T) {
	fb := newFakeBackend(t)
	srv := httptest.NewServer(fb)
	defer srv.Close()

	data, err := os.ReadFile(specPath)
	require.NoError(t, err)
	mutated := strings.Replace(string(data), "  /api/v1/xrd/hostname:\n", "  /api/v1/xrd/hostname-renamed:\n", 1)
	bad := filepath.Join(t.TempDir(), "openapi.yaml")
	require.NoError(t, os.WriteFile(bad, []byte(mutated), 0o600))

	r := runProbe(t, srv.URL, nil, "-spec", bad)
	assert.Equal(t, ExitMismatch, r.code)
	assert.Contains(t, r.stdout, "FAIL  GET /api/v1/xrd/hostname  200  operation is not documented in the specification")
	assert.Contains(t, r.stdout, "SKIP  GET /api/v1/xrd/hostname-renamed  -  not exercised by the probe")
	assertNoCookie(t, r)
}

func TestProbe_NetworkErrors(t *testing.T) {
	t.Run("connection refused", func(t *testing.T) {
		srv := httptest.NewServer(newFakeBackend(t))
		base := srv.URL
		srv.Close()
		r := runProbe(t, base, nil, "-spec", specPath, "-timeout", "2s")
		assert.Equal(t, ExitMismatch, r.code)
		assert.Contains(t, r.stdout, "FAIL  GET /health  -  request failed: ")
		assertFormat(t, r.stdout)
		assertNoCookie(t, r)
	})
	t.Run("TLS verification without opt-in", func(t *testing.T) {
		fb := newFakeBackend(t)
		srv := httptest.NewUnstartedServer(fb)
		srv.Config.ErrorLog = log.New(io.Discard, "", 0) // expected handshake failures
		srv.StartTLS()
		defer srv.Close()
		r := runProbe(t, srv.URL, nil, "-spec", specPath)
		assert.Equal(t, ExitMismatch, r.code)
		assert.Contains(t, r.stdout, "certificate")
		assertNoCookie(t, r)
		assert.Zero(t, fb.cookieSeen, "no request may reach the server over an unverified connection")
	})
	t.Run("server drops connections", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			hj, ok := w.(http.Hijacker)
			require.True(t, ok)
			conn, _, err := hj.Hijack()
			require.NoError(t, err)
			_ = conn.Close()
		}))
		defer srv.Close()
		r := runProbe(t, srv.URL, nil, "-spec", specPath)
		assert.Equal(t, ExitMismatch, r.code)
		assert.Contains(t, r.stdout, "request failed: ")
		assertNoCookie(t, r)
	})
}

func TestProbe_AuthDisabled(t *testing.T) {
	fb := newFakeBackend(t)
	fb.authEnabled = false
	srv := httptest.NewServer(fb)
	defer srv.Close()

	r := runProbe(t, srv.URL, nil, "-spec", specPath)
	require.Equal(t, ExitOK, r.code, r.stdout)
	assert.Contains(t, r.stdout, "PASS  GET /api/auth/user  401  auth disabled on server")
	assert.Contains(t, r.stdout, "SKIP  GET /api/v1/xrd/initialDir  -  unauthenticated check: auth is disabled")
}

func TestProbe_NoFileToDownload(t *testing.T) {
	fb := newFakeBackend(t)
	fb.noFiles = true
	srv := httptest.NewServer(fb)
	defer srv.Close()

	r := runProbe(t, srv.URL, nil, "-spec", specPath)
	require.Equal(t, ExitOK, r.code, r.stdout)
	assert.Contains(t, r.stdout, "SKIP  GET /api/v1/xrd/download  -  no regular file found")
	assert.Contains(t, r.stdout, "SKIP  POST /api/v1/xrd/download/batch  -  no regular file found")
	assert.Empty(t, fb.requested("GET /api/v1/xrd/download"))
}

func TestProbe_Upload(t *testing.T) {
	fb := newFakeBackend(t)
	srv := httptest.NewTLSServer(fb)
	defer srv.Close()

	r := runProbe(t, srv.URL, map[string]string{EnvInsecure: "1"}, "-spec", specPath, "-upload", "-scratch-dir", "/scratch/")
	require.Equal(t, ExitOK, r.code, r.stdout)
	lines := assertFormat(t, r.stdout)
	assertNoCookie(t, r)

	for _, want := range []string{
		"PASS  POST /api/v1/xrd/upload/session  200",
		"PASS  PUT /api/v1/xrd/upload/{uploadId}/chunk  200  offset 0, 16 bytes",
		"PASS  PUT /api/v1/xrd/upload/{uploadId}/chunk  200  offset 16, 16 bytes",
		"PASS  GET /api/v1/xrd/upload/{uploadId}/status  200  live session",
		"PASS  POST /api/v1/xrd/upload/{uploadId}/complete  200  file remains in the scratch directory (no delete endpoint): /scratch/api-probe-",
		"PASS  POST /api/v1/xrd/upload/session  200  second session, to be aborted",
		"PASS  DELETE /api/v1/xrd/upload/{uploadId}  200  abort",
	} {
		assert.True(t, hasLine(lines, want), "missing line %q in:\n%s", want, r.stdout)
	}
	assert.NotContains(t, r.stdout, "SKIP  POST /api/v1/xrd/upload/session")
	// Exactly one file was published, with the content the probe hashed.
	require.Len(t, fb.completedData, 1)
	for name, data := range fb.completedData {
		assert.True(t, strings.HasPrefix(name, "api-probe-"))
		assert.Contains(t, string(data), "api-probe contract check")
	}
	assert.Empty(t, fb.uploads["up-2"], "the second session must be aborted")
	assert.Empty(t, fb.requested("POST /api/auth/logout"))
}

// Names the server's validateFilePath / validateBatchFileName reject must
// never be chosen as the download sample, even when they are the smallest.
func TestProbe_SkipsNamesTheServerRejects(t *testing.T) {
	fb := newFakeBackend(t)
	fb.badNames = true
	srv := httptest.NewServer(fb)
	defer srv.Close()

	r := runProbe(t, srv.URL, nil, "-spec", specPath)
	require.Equal(t, ExitOK, r.code, r.stdout)
	assert.Contains(t, r.stdout, "smallest file (5 bytes)")
	assert.Contains(t, r.stdout, "PASS  GET /api/v1/xrd/download  200")
	assert.Contains(t, r.stdout, "PASS  POST /api/v1/xrd/download/batch  200")
}

func TestUsableName(t *testing.T) {
	for _, n := range []string{"small.txt", "a.b", ".hidden", "a b", "é.txt"} {
		assert.True(t, usableName(n), "%q should be usable", n)
	}
	for _, n := range []string{"", ".", "..", "a..b", "...", "a/b", `a\b`, "a\x00b", "a\rb", "a\nb", "bad\xff"} {
		assert.False(t, usableName(n), "%q should be rejected", n)
	}
}

// shortBusyBackoff makes the 429 retry loop fast for a test.
func shortBusyBackoff(t *testing.T) {
	t.Helper()
	oldN, oldD := busyRetries, busyDelay
	busyRetries, busyDelay = 3, time.Millisecond
	t.Cleanup(func() { busyRetries, busyDelay = oldN, oldD })
}

// The download slot of the previous check may still be held for a moment:
// a transient 429 is retried and the check passes.
func TestProbe_DownloadBusyIsRetried(t *testing.T) {
	shortBusyBackoff(t)
	fb := newFakeBackend(t)
	fb.busy429 = 2
	srv := httptest.NewServer(fb)
	defer srv.Close()

	r := runProbe(t, srv.URL, nil, "-spec", specPath)
	require.Equal(t, ExitOK, r.code, r.stdout)
	assert.Contains(t, r.stdout, "PASS  GET /api/v1/xrd/download  200")
	assert.Contains(t, r.stdout, "PASS  POST /api/v1/xrd/download/batch  200")
	assert.Len(t, fb.requested("GET /api/v1/xrd/download"), 3)
	assert.Len(t, fb.requested("POST /api/v1/xrd/download/batch"), 1)
	lines := assertFormat(t, r.stdout)
	assert.Equal(t, "SUMMARY  pass=16 fail=0 skip=6", lines[len(lines)-1])
}

// A 429 that persists is the documented concurrency limit, not a contract
// violation: it ends in an explained SKIP after a bounded number of retries.
func TestProbe_DownloadBusyPersistsIsSkipped(t *testing.T) {
	shortBusyBackoff(t)
	fb := newFakeBackend(t)
	fb.busy429 = 1000
	srv := httptest.NewServer(fb)
	defer srv.Close()

	r := runProbe(t, srv.URL, nil, "-spec", specPath)
	require.Equal(t, ExitOK, r.code, r.stdout)
	lines := assertFormat(t, r.stdout)
	assert.True(t, hasLine(lines, "SKIP  GET /api/v1/xrd/download  -  first 4096 bytes"), r.stdout)
	assert.True(t, hasLine(lines, "SKIP  POST /api/v1/xrd/download/batch  -  archive of the same file, first bytes only: server still answers 429"), r.stdout)
	assert.Len(t, fb.requested("GET /api/v1/xrd/download"), 1+busyRetries)
	assert.Len(t, fb.requested("POST /api/v1/xrd/download/batch"), 1+busyRetries)
	assert.Equal(t, "SUMMARY  pass=14 fail=0 skip=8", lines[len(lines)-1])
}

// A 429 that does not match the spec is still a contract FAIL, not a SKIP.
func TestProbe_UndocumentedBusyStillFails(t *testing.T) {
	shortBusyBackoff(t)
	fb := newFakeBackend(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/xrd/download/batch" {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(429)
			_, _ = w.Write([]byte("slow down"))
			return
		}
		fb.ServeHTTP(w, r)
	}))
	defer srv.Close()

	r := runProbe(t, srv.URL, nil, "-spec", specPath)
	assert.Equal(t, ExitMismatch, r.code, r.stdout)
	assert.Contains(t, r.stdout, "FAIL  POST /api/v1/xrd/download/batch  429")
}

// A JSON body of exactly the limit is accepted; one byte more is rejected.
func TestSend_JSONBodyLimitBoundary(t *testing.T) {
	old := jsonBodyLimit
	jsonBodyLimit = 64
	t.Cleanup(func() { jsonBodyLimit = old })

	var size int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`"` + strings.Repeat("x", size-2) + `"`))
	}))
	defer srv.Close()
	base, err := url.Parse(srv.URL)
	require.NoError(t, err)
	p := &prober{opts: &Options{BaseURL: base, Timeout: time.Second}, client: srv.Client()}

	size = 64
	res, err := p.send(call{method: "GET", path: "/x"})
	require.NoError(t, err)
	assert.Len(t, res.body, 64)

	size = 65
	_, err = p.send(call{method: "GET", path: "/x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JSON body exceeds 64 bytes")
}

func TestProbe_UploadFailureCleansUp(t *testing.T) {
	fb := newFakeBackend(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			errJSON(w, 409, "offset mismatch")
			return
		}
		fb.ServeHTTP(w, r)
	}))
	defer srv.Close()

	r := runProbe(t, srv.URL, nil, "-spec", specPath, "-upload", "-scratch-dir", "/scratch")
	assert.Equal(t, ExitMismatch, r.code)
	assert.Contains(t, r.stdout, "FAIL  PUT /api/v1/xrd/upload/{uploadId}/chunk  409")
	assert.Contains(t, r.stdout, "SKIP  POST /api/v1/xrd/upload/{uploadId}/complete  -  chunk upload failed")
	assert.Contains(t, r.stdout, "PASS  DELETE /api/v1/xrd/upload/{uploadId}  200  cleanup of the failed session")
	assert.Empty(t, fb.completedData)
	assertNoCookie(t, r)
}

func TestMain_UsageErrors(t *testing.T) {
	fb := newFakeBackend(t)
	srv := httptest.NewServer(fb)
	defer srv.Close()

	cases := []struct {
		name string
		env  map[string]string
		args []string
		want string
	}{
		{"missing base URL", map[string]string{EnvBaseURL: ""}, nil, EnvBaseURL + " is not set"},
		{"relative base URL", map[string]string{EnvBaseURL: "localhost:22000"}, nil, "absolute http(s) URL"},
		{"base URL with query", map[string]string{EnvBaseURL: srv.URL + "?x=1"}, nil, "must not contain"},
		{"missing cookie", map[string]string{EnvCookie: ""}, nil, EnvCookie + " is not set"},
		{"cookie with separator", map[string]string{EnvCookie: testCookie + "; other=1"}, nil, "not allowed in a cookie value"},
		{"bad insecure value", map[string]string{EnvInsecure: "yes"}, nil, EnvInsecure},
		{"cookie as positional argument", nil, []string{"-spec", specPath, testCookie}, "positional arguments"},
		{"unknown cookie flag", nil, []string{"-cookie", testCookie}, "flag provided but not defined"},
		{"upload without scratch dir", nil, []string{"-spec", specPath, "-upload"}, "must be given together"},
		{"scratch dir without upload", nil, []string{"-spec", specPath, "-scratch-dir", "/s"}, "must be given together"},
		{"relative scratch dir", nil, []string{"-spec", specPath, "-upload", "-scratch-dir", "s"}, "absolute remote path"},
		{"root scratch dir", nil, []string{"-spec", specPath, "-upload", "-scratch-dir", "/"}, "must not be the root directory"},
		{"double-slash root scratch dir", nil, []string{"-spec", specPath, "-upload", "-scratch-dir", "//"}, "must not be the root directory"},
		{"scratch dir with traversal", nil, []string{"-spec", specPath, "-upload", "-scratch-dir", "/s/../t"}, "absolute remote path"},
		{"scratch dir with newline", nil, []string{"-spec", specPath, "-upload", "-scratch-dir", "/s\nt"}, "absolute remote path"},
		{"bad timeout", nil, []string{"-spec", specPath, "-timeout", "0s"}, "-timeout must be positive"},
		{"missing spec", nil, []string{"-spec", "testdata/missing.yaml"}, "read spec"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := runProbe(t, srv.URL, c.env, c.args...)
			assert.Equal(t, ExitUsage, r.code)
			assert.Contains(t, r.stderr, c.want)
			assert.Empty(t, r.stdout)
			assertNoCookie(t, r)
		})
	}
	assert.Empty(t, fb.requested(""), "usage errors must not send any request")

	t.Run("help", func(t *testing.T) {
		r := runProbe(t, srv.URL, nil, "-h")
		assert.Equal(t, ExitUsage, r.code)
		assert.Contains(t, r.stderr, EnvCookie)
		assertNoCookie(t, r)
	})
}

func TestMain_CookieWithNamePrefix(t *testing.T) {
	fb := newFakeBackend(t)
	srv := httptest.NewServer(fb)
	defer srv.Close()

	r := runProbe(t, srv.URL, map[string]string{EnvCookie: " " + SessionCookieName + "=" + testCookie + "\n"}, "-spec", specPath)
	assert.Equal(t, ExitOK, r.code, r.stdout)
	assert.Zero(t, fb.wrongCookie)
	assertNoCookie(t, r)
}

func TestRedactor(t *testing.T) {
	red := newRedactor(testCookie)
	in := fmt.Sprintf("a %s b %s c %s", testCookie, url.QueryEscape(testCookie), url.PathEscape(testCookie))
	assert.Equal(t, "a [REDACTED] b [REDACTED] c [REDACTED]", red.String(in))
	assert.Equal(t, "unchanged", newRedactor("").String("unchanged"))

	var buf bytes.Buffer
	w := redactingWriter{w: &buf, r: red}
	n, err := fmt.Fprintf(w, "Get \"https://h/?c=%s\": EOF\n", url.QueryEscape(testCookie))
	require.NoError(t, err)
	assert.Positive(t, n)
	assert.Equal(t, "Get \"https://h/?c=[REDACTED]\": EOF\n", buf.String())
}

func TestOneLine(t *testing.T) {
	assert.Equal(t, `a\nb\r\tc\x00\x7f`, oneLine("a\nb\r\tc\x00\x7f", 0))
	assert.Equal(t, `bad\xff`, oneLine("bad\xff", 0))
	assert.Equal(t, "abc...", oneLine("abcdef", 3))
	assert.Equal(t, "é...", oneLine("éé", 3)) // never splits a rune
	assert.Equal(t, "short", oneLine("short", 10))
}
