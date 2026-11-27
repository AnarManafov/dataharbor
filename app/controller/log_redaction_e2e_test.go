package controller

// End-to-end guard for GH-82: a credential must never reach the log, at any
// level. Each test drives real requests through the production middleware
// order (Recovery → DebugRequestBody → TraceRequest → SessionAuthMiddleware)
// with the logger capturing at debug level, then searches everything that was
// logged for sentinel secrets.
//
// The tests deliberately know nothing about the redaction helper: they only
// look at what comes out of the logger, gin's writers and the standard
// library logger. That keeps them valid against any implementation.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/AnarManafov/dataharbor/app/common"
	"github.com/AnarManafov/dataharbor/app/config"
	"github.com/AnarManafov/dataharbor/app/middleware"
)

// Sentinels are plain alphanumerics so no encoding (URL, JSON, base64 of a
// header dump) can hide one, and each channel has its own so a failure names
// the channel that leaked.
const (
	sentBearer      = "SENTINELbearerHdr7Qx9Lm2Vt4Kp8Rz"
	sentCookie      = "SENTINELcookieVal3Hn6Bw1Ys5Dc9Fj"
	sentQuery       = "SENTINELqueryTok8Mk2Pv7Zq4Xt1Gh"
	sentBodyToken   = "SENTINELbodyAccess5Jr9Wd3Nc6Lb2"
	sentBodyNested  = "SENTINELbodyNested1Tf4Sg8Hv7Ke3"
	sentBodySecret  = "SENTINELbodySecret6Zp2Qm9Xc4Rn8"
	sentStoreAccess = "SENTINELstoreAccess9Bd5Vk1Wj7Ty3"
	sentStoreRefr   = "SENTINELstoreRefresh2Lg8Fh4Ms6Pq"
	sentStoreID     = "SENTINELstoreIdTok4Cn7Rx1Zd9Hw5"
	sentRespToken   = "SENTINELrespToken3Kv6Jt2Yb8Nf1"
	sentRespURL     = "SENTINELrespUrlState7Wq5Gd3Mx9"
	sentSetCookie   = "SENTINELsetCookie8Hs2Lp6Vc4Zk1"
	sentLocation    = "SENTINELlocationCode5Tr9Bn3Fy7"
	sentUserinfo    = "SENTINELuserinfoClaim1Xj4Qw8Dm6"
	sentIdPBody     = "SENTINELidpErrBody6Gc2Kz9Vh3Ts"
	sentAuthCode    = "SENTINELauthCode4Pm7Nd1Rb5Yx8"
	sentClientSec   = "SENTINELclientSecret9Fw3Hq6Lt2"

	// Sentinels in bodies larger than the debug body limit (GH-83): one
	// straddles the limit, the other lies wholly past it.
	sentStraddleStrong = "SENTINELstraddleStrong3Hq8Wd1Kv6"
	sentStraddleURL    = "SENTINELstraddleUrlPw7Tm2Xc5Lz9"
	sentStraddleForm   = "SENTINELstraddleForm4Rb9Jn2Gs7"
	sentPastLimit      = "SENTINELpastLimit8Vf3Qk6Mp1Yw"

	// Sentinels in JSON bodies whose Content-Type does not say JSON (fetch()
	// sends text/plain), and in a text/plain body that only starts like JSON.
	sentSniffedJSON  = "SENTINELsniffedJson5Kw2Pd8Xm4"
	sentSniffedStrad = "SENTINELsniffedStraddle7Lq3Vn6"
	sentNotJSON      = "SENTINELnotJson2Hc9Ry4Tb7Gz"

	// e2eBodyLimit is server.debug_body_log_limit in these tests.
	e2eBodyLimit = 512

	// Non-secret values that MUST survive, proving that logging is live and
	// that redaction is not simply dropping everything.
	visibleRequestMarker  = "e2e-visible-request-marker"
	visibleResponseMarker = "e2e-visible-response-marker"
)

// captureMode selects the logger the requests run against.
type captureMode string

const (
	// The bare logger: nothing but the call sites stands between a secret
	// and the log, so a statement that forgets to redact fails here.
	captureCallSites captureMode = "call sites only"
	// The logger exactly as main builds it, writing JSON to a file. This
	// proves the helpers and the logger's own redaction pass compose: no
	// secret gets through and the non-secret context survives both.
	captureProduction captureMode = "production logger"
)

// forEachCaptureMode runs fn once per capture mode.
func forEachCaptureMode(t *testing.T, fn func(t *testing.T, mode captureMode)) {
	t.Helper()
	for _, mode := range []captureMode{captureCallSites, captureProduction} {
		t.Run(string(mode), func(t *testing.T) { fn(t, mode) })
	}
}

// logCapture collects every sink a credential could reach: the zap logger,
// gin's default writers and the standard library logger.
type logCapture struct {
	zapLogs *observer.ObservedLogs // captureCallSites
	logFile string                 // captureProduction
	other   *syncBuffer
}

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// startLogCapture swaps in capturing sinks and restores the originals on
// cleanup. It must run before the router is built: SessionAuthMiddleware
// keeps the logger it sees at construction.
func startLogCapture(t *testing.T, mode captureMode) *logCapture {
	t.Helper()

	lc := &logCapture{other: &syncBuffer{}}
	prevLogger := common.Logger
	switch mode {
	case captureProduction:
		lc.logFile = filepath.Join(t.TempDir(), "backend.log")
		common.InitLogger(&config.LoggingConfig{
			Level: "debug",
			File: config.FileConfig{
				Enabled:  true,
				Level:    "debug",
				Format:   "json",
				Filename: lc.logFile,
				MaxSize:  10,
			},
		})
	default:
		core, logs := observer.New(zapcore.DebugLevel)
		lc.zapLogs = logs
		common.Logger = zap.New(core).Sugar()
	}

	prevGinOut, prevGinErr := gin.DefaultWriter, gin.DefaultErrorWriter
	gin.DefaultWriter, gin.DefaultErrorWriter = lc.other, lc.other
	prevStd := log.Writer()
	log.SetOutput(lc.other)

	t.Cleanup(func() {
		common.Logger = prevLogger
		gin.DefaultWriter, gin.DefaultErrorWriter = prevGinOut, prevGinErr
		log.SetOutput(prevStd)
	})
	return lc
}

// lines returns everything captured so far, one entry per element.
func (lc *logCapture) lines() []string {
	var out []string
	if lc.zapLogs != nil {
		for _, e := range lc.zapLogs.All() {
			line := e.Level.String() + " " + e.Message
			if len(e.Context) > 0 {
				line += " " + fmt.Sprint(e.ContextMap())
			}
			out = append(out, line)
		}
	}
	if lc.logFile != "" {
		// The file does not exist until the first entry is written.
		if data, err := os.ReadFile(lc.logFile); err == nil {
			out = append(out, strings.Split(string(data), "\n")...)
		}
	}
	if s := lc.other.String(); s != "" {
		out = append(out, strings.Split(s, "\n")...)
	}
	return out
}

func (lc *logCapture) text() string { return strings.Join(lc.lines(), "\n") }

// assertNoSecrets fails for every secret found anywhere in the capture and
// quotes the offending lines.
func (lc *logCapture) assertNoSecrets(t *testing.T, secrets map[string]string) {
	t.Helper()
	lines := lc.lines()
	for name, secret := range secrets {
		require.NotEmpty(t, secret, "secret %q is empty; the test would be vacuous", name)
		for _, line := range lines {
			if strings.Contains(line, secret) {
				t.Errorf("%s leaked into the log: %s", name, line)
			}
		}
	}
}

// assertLogged proves the capture is live: each marker must be present.
func (lc *logCapture) assertLogged(t *testing.T, markers ...string) {
	t.Helper()
	text := lc.text()
	for _, m := range markers {
		assert.Contains(t, text, m, "expected %q in the log; without it the leak assertions prove nothing", m)
	}
}

// redactionIdP is a mock identity provider. tokenStatus selects how the token
// endpoint answers; endSession overrides the advertised end-session endpoint.
type redactionIdP struct {
	server     *httptest.Server
	endSession string
}

func newRedactionIdP(t *testing.T) *redactionIdP {
	t.Helper()
	idp := &redactionIdP{}
	idp.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		base := "http://" + r.Host
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			endSession := idp.endSession
			if endSession == "" {
				endSession = base + "/logout"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"issuer":                 base,
				"authorization_endpoint": base + "/authorize",
				"token_endpoint":         base + "/token",
				"userinfo_endpoint":      base + "/userinfo",
				"end_session_endpoint":   endSession,
			})
		case "/userinfo":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sub":          "e2e-user",
				"email":        sentUserinfo + "@example.org",
				"access_token": sentIdPBody,
			})
		case "/token":
			// A failing token endpoint that echoes credentials back, as some
			// providers do in error details.
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error":             "invalid_grant",
				"error_description": "grant rejected",
				"refresh_token":     sentIdPBody,
				"detail":            "see https://idp.example/err?access_token=" + sentIdPBody,
			})
		case "/logout":
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error":         "server_error",
				"id_token_hint": sentIdPBody,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(idp.server.Close)
	return idp
}

// setupRedactionE2E configures auth against the mock IdP, starts the capture
// and builds a router with the production middleware order.
func setupRedactionE2E(t *testing.T, idp *redactionIdP, mode captureMode) (*gin.Engine, *logCapture) {
	t.Helper()

	prevCfg := config.GetConfig()
	config.SetConfig(&config.Config{
		Env:    "production",
		Server: config.ServerConfig{Address: ":8080", Debug: true, DebugBodyLogLimit: e2eBodyLimit},
		Auth: config.AuthConfig{
			Enabled:       true,
			SkipAuthPaths: []string{"/health"},
			OIDC: config.OIDCConfig{
				Issuer:                idp.server.URL,
				ClientID:              "e2e-client",
				ClientSecret:          sentClientSec,
				SessionSecret:         "e2e-session-secret-0123456789abcdef",
				TokenRefreshBufferSec: 60,
			},
		},
		Frontend: config.FrontendConfig{URL: "http://localhost:5173"},
	})
	t.Cleanup(func() { config.SetConfig(prevCfg) })

	lc := startLogCapture(t, mode)

	InitAuth()
	clearTokenStore()
	clearDiscoveryDocCache()
	clearUserInfoCache()
	t.Cleanup(func() {
		clearTokenStore()
		clearDiscoveryDocCache()
		clearUserInfoCache()
	})

	r := gin.New()
	r.Use(middleware.Recovery())
	r.Use(middleware.DebugRequestBody(config.GetConfig().Server.DebugBodyLogLimit, nil))
	r.Use(middleware.TraceRequest())

	auth := r.Group("/api/auth")
	auth.GET("/login", LoginInit)
	auth.GET("/callback", AuthCallback)
	auth.POST("/logout", Logout)
	auth.GET("/user", GetCurrentUser)

	api := r.Group("/api")
	api.Use(SessionAuthMiddleware())
	v1 := api.Group("/v1")
	v1.POST("/echo", func(c *gin.Context) {
		// A handler that answers with everything a response could carry.
		c.Header("Set-Cookie", "issued="+sentSetCookie+"; Path=/; HttpOnly")
		c.Header("Location", "https://app.example/cb?code="+sentLocation+"&state="+sentLocation)
		c.JSON(http.StatusOK, gin.H{
			"access_token": sentRespToken,
			"auth_url":     "https://idp.example/authorize?client_id=e2e&state=" + sentRespURL,
			"note":         visibleResponseMarker,
		})
	})
	v1.POST("/echo-raw", func(c *gin.Context) {
		// Echoes the request body, so a body that straddles the debug body
		// limit does so in both directions.
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		c.Data(http.StatusOK, c.ContentType(), body)
	})
	v1.GET("/panic", func(c *gin.Context) {
		panic("handler exploded")
	})
	return r, lc
}

// mintSessionCookie returns a signed session cookie that refers to tokenID.
func mintSessionCookie(t *testing.T, tokenID string) *http.Cookie {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	session, _ := SessionStore.Get(req, sessionName)
	session.Values["token_id"] = tokenID
	require.NoError(t, session.Save(req, w))
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionName {
			return c
		}
	}
	t.Fatal("session cookie was not issued")
	return nil
}

// credentialedRequest builds a request that carries a sentinel on every
// channel a client controls: bearer header, cookie, query string and body.
func credentialedRequest(method, path string, sessionCookie *http.Cookie) *http.Request {
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	var body *strings.Reader
	if method == http.MethodPost {
		payload, _ := json.Marshal(map[string]any{
			"access_token": sentBodyToken,
			"note":         "request-note",
			"nested": map[string]any{
				"refresh_token": sentBodyNested,
				"clients":       []any{map[string]any{"client_secret": sentBodySecret}},
			},
		})
		body = strings.NewReader(string(payload))
	} else {
		body = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path+sep+"access_token="+sentQuery, body)
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+sentBearer)
	req.Header.Set("X-E2e-Marker", visibleRequestMarker)
	req.AddCookie(&http.Cookie{Name: "other", Value: sentCookie})
	if sessionCookie != nil {
		req.AddCookie(sessionCookie)
	}
	return req
}

// clientSecrets are the sentinels a client sends with credentialedRequest.
func clientSecrets() map[string]string {
	return map[string]string{
		"bearer token (Authorization header)": sentBearer,
		"cookie value":                        sentCookie,
		"query-string token":                  sentQuery,
		"body access_token":                   sentBodyToken,
		"nested body refresh_token":           sentBodyNested,
		"body client_secret in array":         sentBodySecret,
		"configured OIDC client secret":       sentClientSec,
	}
}

func merge(maps ...map[string]string) map[string]string {
	out := map[string]string{}
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

func TestLogRedactionE2E_AcceptedRequest(t *testing.T) {
	forEachCaptureMode(t, runAcceptedRequest)
}

func runAcceptedRequest(t *testing.T, mode captureMode) {
	idp := newRedactionIdP(t)
	router, lc := setupRedactionE2E(t, idp, mode)

	tokenID := storeTokens(sentStoreAccess, sentStoreRefr, sentStoreID, time.Now().Add(time.Hour).Unix())
	sessionCookie := mintSessionCookie(t, tokenID)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, credentialedRequest(http.MethodPost, "/api/v1/echo", sessionCookie))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	lc.assertLogged(t, "Request Headers", "Request Body", "Response Headers", "Response Body",
		visibleRequestMarker, visibleResponseMarker, "Request completed")
	lc.assertNoSecrets(t, merge(clientSecrets(), map[string]string{
		"session cookie value":             sessionCookie.Value,
		"stored access token":              sentStoreAccess,
		"stored refresh token":             sentStoreRefr,
		"stored id token":                  sentStoreID,
		"response body access_token":       sentRespToken,
		"state inside a response URL":      sentRespURL,
		"Set-Cookie value":                 sentSetCookie,
		"code/state in Location header":    sentLocation,
		"userinfo claim (IdP response)":    sentUserinfo,
		"token field in userinfo response": sentIdPBody,
	}))
}

func TestLogRedactionE2E_RejectedRequests(t *testing.T) {
	forEachCaptureMode(t, runRejectedRequests)
}

func runRejectedRequests(t *testing.T, mode captureMode) {
	t.Run("no session", func(t *testing.T) {
		idp := newRedactionIdP(t)
		router, lc := setupRedactionE2E(t, idp, mode)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, credentialedRequest(http.MethodPost, "/api/v1/echo", nil))
		require.Equal(t, http.StatusUnauthorized, w.Code)

		lc.assertLogged(t, "Request Headers", visibleRequestMarker, "No token ID in session", "Request failed")
		lc.assertNoSecrets(t, clientSecrets())
	})

	t.Run("expired session and the IdP rejects the refresh", func(t *testing.T) {
		idp := newRedactionIdP(t)
		router, lc := setupRedactionE2E(t, idp, mode)

		tokenID := storeTokens(sentStoreAccess, sentStoreRefr, sentStoreID, time.Now().Add(-time.Hour).Unix())
		sessionCookie := mintSessionCookie(t, tokenID)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, credentialedRequest(http.MethodPost, "/api/v1/echo", sessionCookie))
		require.Equal(t, http.StatusUnauthorized, w.Code)

		lc.assertLogged(t, "Request Headers", "Failed to refresh token", "Request failed")
		lc.assertNoSecrets(t, merge(clientSecrets(), map[string]string{
			"session cookie value":    sessionCookie.Value,
			"stored access token":     sentStoreAccess,
			"stored refresh token":    sentStoreRefr,
			"stored id token":         sentStoreID,
			"IdP error response body": sentIdPBody,
		}))
	})
}

func TestLogRedactionE2E_LoginAndCallback(t *testing.T) {
	forEachCaptureMode(t, runLoginAndCallback)
}

func runLoginAndCallback(t *testing.T, mode captureMode) {
	idp := newRedactionIdP(t)
	router, lc := setupRedactionE2E(t, idp, mode)

	// Login: the server generates the OIDC state and a session cookie.
	w := httptest.NewRecorder()
	router.ServeHTTP(w, credentialedRequest(http.MethodGet, "/api/auth/login?redirect_path=/browse", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var loginResp struct {
		AuthURL string `json:"auth_url"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &loginResp))
	_, afterState, found := strings.Cut(loginResp.AuthURL, "&state=")
	require.True(t, found, "auth_url has no state: %s", loginResp.AuthURL)
	state, _, _ := strings.Cut(afterState, "&")
	require.NotEmpty(t, state)

	var loginCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionName {
			loginCookie = c
		}
	}
	require.NotNil(t, loginCookie, "login did not set the session cookie")

	// Callback: the code is exchanged at a token endpoint that fails and
	// echoes credentials in its error body.
	w = httptest.NewRecorder()
	router.ServeHTTP(w, credentialedRequest(http.MethodGet,
		"/api/auth/callback?code="+sentAuthCode+"&state="+state, loginCookie))
	require.NotEqual(t, http.StatusTemporaryRedirect, w.Code, "the mock token endpoint must fail")

	lc.assertLogged(t, "Request Headers", visibleRequestMarker, "Using authorization endpoint",
		"Using token endpoint", "invalid_grant")
	lc.assertNoSecrets(t, merge(clientSecrets(), map[string]string{
		"OIDC state":                  state,
		"session cookie set by login": loginCookie.Value,
		"authorization code":          sentAuthCode,
		"IdP error response body":     sentIdPBody,
	}))
}

func TestLogRedactionE2E_CurrentUserAndLogout(t *testing.T) {
	forEachCaptureMode(t, runCurrentUserAndLogout)
}

func runCurrentUserAndLogout(t *testing.T, mode captureMode) {
	idp := newRedactionIdP(t)
	// Nothing listens here, so the logout call fails with an error that
	// quotes the full logout URL, id_token_hint included.
	idp.endSession = "http://127.0.0.1:1/logout"
	router, lc := setupRedactionE2E(t, idp, mode)

	tokenID := storeTokens(sentStoreAccess, sentStoreRefr, sentStoreID, time.Now().Add(time.Hour).Unix())
	sessionCookie := mintSessionCookie(t, tokenID)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, credentialedRequest(http.MethodGet, "/api/auth/user", sessionCookie))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = httptest.NewRecorder()
	router.ServeHTTP(w, credentialedRequest(http.MethodPost, "/api/auth/logout", sessionCookie))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// The IdP logout runs in a goroutine; wait for its failure to be logged.
	require.Eventually(t, func() bool {
		return strings.Contains(lc.text(), "Failed to logout from OIDC provider")
	}, 10*time.Second, 20*time.Millisecond, "the asynchronous IdP logout never logged its failure")

	lc.assertLogged(t, "Request Headers", visibleRequestMarker, "User info fetched successfully")
	lc.assertNoSecrets(t, merge(clientSecrets(), map[string]string{
		"session cookie value":             sessionCookie.Value,
		"stored access token":              sentStoreAccess,
		"stored refresh token":             sentStoreRefr,
		"id token (id_token_hint in URL)":  sentStoreID,
		"token field in userinfo response": sentIdPBody,
	}))

	// GET /api/auth/user returns the user's claims to the client, so the debug
	// middleware's response-body line shows them (with token fields redacted,
	// checked above). Nothing else may: the identity-provider response must
	// not be dumped by the handlers at any level.
	for _, line := range lc.lines() {
		if strings.Contains(line, sentUserinfo) && !strings.Contains(line, "Response Body:") {
			t.Errorf("userinfo claim (IdP response) leaked into the log: %s", line)
		}
	}
}

func TestLogRedactionE2E_LogoutRejectedByIdP(t *testing.T) {
	forEachCaptureMode(t, runLogoutRejectedByIdP)
}

func runLogoutRejectedByIdP(t *testing.T, mode captureMode) {
	idp := newRedactionIdP(t)
	router, lc := setupRedactionE2E(t, idp, mode)

	tokenID := storeTokens(sentStoreAccess, sentStoreRefr, sentStoreID, time.Now().Add(time.Hour).Unix())
	sessionCookie := mintSessionCookie(t, tokenID)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, credentialedRequest(http.MethodPost, "/api/auth/logout", sessionCookie))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.Eventually(t, func() bool {
		return strings.Contains(lc.text(), "OIDC provider logout failed with status 500")
	}, 10*time.Second, 20*time.Millisecond, "the asynchronous IdP logout never logged the rejection")

	lc.assertNoSecrets(t, merge(clientSecrets(), map[string]string{
		"session cookie value":    sessionCookie.Value,
		"stored id token":         sentStoreID,
		"IdP error response body": sentIdPBody,
	}))
}

func TestLogRedactionE2E_PanicInHandler(t *testing.T) {
	forEachCaptureMode(t, runPanicInHandler)
}

func runPanicInHandler(t *testing.T, mode captureMode) {
	// gin dumps the request on a recovered panic in debug mode.
	prevMode := gin.Mode()
	gin.SetMode(gin.DebugMode)
	t.Cleanup(func() { gin.SetMode(prevMode) })

	idp := newRedactionIdP(t)
	router, lc := setupRedactionE2E(t, idp, mode)

	tokenID := storeTokens(sentStoreAccess, sentStoreRefr, sentStoreID, time.Now().Add(time.Hour).Unix())
	sessionCookie := mintSessionCookie(t, tokenID)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, credentialedRequest(http.MethodGet, "/api/v1/panic", sessionCookie))
	require.Equal(t, http.StatusInternalServerError, w.Code)

	lc.assertLogged(t, "handler exploded")
	lc.assertNoSecrets(t, merge(clientSecrets(), map[string]string{
		"session cookie value": sessionCookie.Value,
		"stored access token":  sentStoreAccess,
	}))
}

// straddlingBody returns head + pad + mid + secret + tail, with the pad sized
// so that secret starts half its length before limit: the limit cuts it in
// two.
func straddlingBody(t *testing.T, limit int, head, mid, secret, tail string) string {
	t.Helper()
	padLen := limit - len(secret)/2 - len(head) - len(mid)
	require.Positive(t, padLen)
	body := head + strings.Repeat("p", padLen) + mid + secret + tail
	start := strings.Index(body, secret)
	require.Less(t, start, limit)
	require.Greater(t, start+len(secret), limit)
	return body
}

// assertNoSecretFragments fails if any 8-byte window of a secret is logged,
// so neither a prefix nor a suffix of a cut secret can slip through.
func (lc *logCapture) assertNoSecretFragments(t *testing.T, secrets map[string]string) {
	t.Helper()
	text := lc.text()
	for name, secret := range secrets {
		for i := 0; i+8 <= len(secret); i++ {
			if strings.Contains(text, secret[i:i+8]) {
				t.Errorf("fragment %q of %s leaked into the log", secret[i:i+8], name)
			}
		}
	}
}

func TestLogRedactionE2E_BodiesLargerThanTheLimit(t *testing.T) {
	forEachCaptureMode(t, runBodiesLargerThanTheLimit)
}

func runBodiesLargerThanTheLimit(t *testing.T, mode captureMode) {
	jsonHead := `{"note":"` + visibleRequestMarker + `","pad":"`
	past := `","later":{"client_secret":"` + sentPastLimit + `","n":1},"more":"` + strings.Repeat("q", 2*e2eBodyLimit) + `"}`
	variants := []struct {
		name, ctype, body string
		chunked           bool // no Content-Length (-1)
		secrets           map[string]string
	}{
		{
			// fetch() with a JSON string body and no explicit header.
			name:  "JSON sent as text/plain straddles the limit",
			ctype: "text/plain;charset=UTF-8",
			body:  straddlingBody(t, e2eBodyLimit, jsonHead, `","client_secret":"`, sentSniffedStrad, past),
			secrets: map[string]string{
				"straddling client_secret": sentSniffedStrad,
				"client_secret past limit": sentPastLimit,
			},
		},
		{
			name:    "JSON with no Content-Type and no Content-Length straddles the limit",
			ctype:   "",
			chunked: true,
			body:    straddlingBody(t, e2eBodyLimit, jsonHead, `","refresh_token":"`, sentSniffedStrad, past),
			secrets: map[string]string{
				"straddling refresh_token": sentSniffedStrad,
				"client_secret past limit": sentPastLimit,
			},
		},
		{
			name:  "strong key straddles the limit",
			ctype: "application/json",
			body:  straddlingBody(t, e2eBodyLimit, jsonHead, `","refresh_token":"`, sentStraddleStrong, past),
			secrets: map[string]string{
				"straddling refresh_token": sentStraddleStrong,
				"client_secret past limit": sentPastLimit,
			},
		},
		{
			// A neutral key: only the shape (URL userinfo) marks the secret,
			// and a cut URL no longer has that shape.
			name:  "URL password straddles the limit",
			ctype: "application/json",
			body:  straddlingBody(t, e2eBodyLimit, jsonHead, `","link":"https://alice:`, sentStraddleURL, `@example.org/x`+past),
			secrets: map[string]string{
				"straddling URL password":  sentStraddleURL,
				"client_secret past limit": sentPastLimit,
			},
		},
		{
			name:  "form field straddles the limit",
			ctype: "application/x-www-form-urlencoded",
			body: straddlingBody(t, e2eBodyLimit, "note="+visibleRequestMarker+"&pad=", "&access_token=", sentStraddleForm,
				"&client_secret="+sentPastLimit+"&more="+strings.Repeat("q", 2*e2eBodyLimit)),
			secrets: map[string]string{
				"straddling form access_token": sentStraddleForm,
				"client_secret past limit":     sentPastLimit,
			},
		},
	}

	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			idp := newRedactionIdP(t)
			router, lc := setupRedactionE2E(t, idp, mode)
			tokenID := storeTokens(sentStoreAccess, sentStoreRefr, sentStoreID, time.Now().Add(time.Hour).Unix())
			sessionCookie := mintSessionCookie(t, tokenID)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/echo-raw", strings.NewReader(v.body))
			if v.chunked {
				req.ContentLength = -1
			}
			if v.ctype != "" {
				req.Header.Set("Content-Type", v.ctype)
			}
			req.AddCookie(sessionCookie)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, v.body, w.Body.String(), "the handler or client did not get the whole body")

			text := lc.text()
			assert.Equal(t, 2, strings.Count(text, visibleRequestMarker), "request and response bodies must both be logged:\n%s", text)
			assert.Equal(t, 2, strings.Count(text, "(truncated)"), "request and response bodies must both be marked truncated:\n%s", text)
			lc.assertNoSecretFragments(t, merge(v.secrets, map[string]string{
				"session cookie value": sessionCookie.Value,
				"stored access token":  sentStoreAccess,
			}))
		})
	}
}

func TestLogRedactionE2E_JSONBodiesWithoutJSONType(t *testing.T) {
	forEachCaptureMode(t, runJSONBodiesWithoutJSONType)
}

// runJSONBodiesWithoutJSONType sends whole (not truncated) bodies that are
// JSON without saying so, which debug mode logs redacted, and a text/plain
// body that only starts like JSON, which it must not log at all.
func runJSONBodiesWithoutJSONType(t *testing.T, mode captureMode) {
	jsonBody := `{"note":"` + visibleRequestMarker + `","client_secret":"` + sentSniffedJSON +
		`","nested":{"password":"` + sentSniffedJSON + `"}}`
	variants := []struct {
		name, ctype, body string
		chunked, logged   bool
	}{
		{name: "text/plain from fetch", ctype: "text/plain;charset=UTF-8", body: jsonBody, logged: true},
		{name: "no Content-Type", body: jsonBody, logged: true},
		{name: "no Content-Type, no Content-Length", body: jsonBody, chunked: true, logged: true},
		{name: "text/plain that is not JSON", ctype: "text/plain",
			body: `{note: "` + visibleRequestMarker + `", password: "` + sentNotJSON + `"}`},
	}
	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			idp := newRedactionIdP(t)
			router, lc := setupRedactionE2E(t, idp, mode)
			tokenID := storeTokens(sentStoreAccess, sentStoreRefr, sentStoreID, time.Now().Add(time.Hour).Unix())
			sessionCookie := mintSessionCookie(t, tokenID)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/echo-raw", strings.NewReader(v.body))
			if v.chunked {
				req.ContentLength = -1
			}
			if v.ctype != "" {
				req.Header.Set("Content-Type", v.ctype)
			}
			req.AddCookie(sessionCookie)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, v.body, w.Body.String())

			text := lc.text()
			if v.logged {
				assert.Equal(t, 2, strings.Count(text, visibleRequestMarker), "request and response bodies must both be logged:\n%s", text)
				// Two secrets in each of two bodies, plus the session cookie.
				assert.GreaterOrEqual(t, strings.Count(text, "[REDACTED]"), 2*2+1, "both secrets redacted in both bodies:\n%s", text)
			} else {
				assert.NotContains(t, text, visibleRequestMarker, "a body that is not JSON was logged")
			}
			lc.assertNoSecretFragments(t, map[string]string{
				"sniffed JSON secret":  sentSniffedJSON,
				"not-JSON secret":      sentNotJSON,
				"session cookie value": sessionCookie.Value,
				"stored access token":  sentStoreAccess,
			})
		})
	}
}
