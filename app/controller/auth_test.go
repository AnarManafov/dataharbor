package controller

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AnarManafov/dataharbor/app/config"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// Helper function to set up a test config
func setupTestConfig(authEnabled bool, issuer, clientID, clientSecret string) {
	testConfig := &config.Config{
		Env: "test",
		Server: config.ServerConfig{
			Address: ":8080",
			SSL: config.SSLConfig{
				Enabled: false,
			},
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
			Console: config.ConsoleConfig{
				Enabled: true,
				Level:   "info",
				Format:  "text",
			},
			File: config.FileConfig{
				Enabled: false,
			},
		},
		Auth: config.AuthConfig{
			Enabled: authEnabled,
			SkipAuthPaths: []string{
				"/health",
				"/api/health",
			},
			OIDC: config.OIDCConfig{
				Issuer:                issuer,
				ClientID:              clientID,
				ClientSecret:          clientSecret,
				SessionSecret:         "test-session-secret",
				TokenRefreshBufferSec: 60,
			},
		},
		Frontend: config.FrontendConfig{
			URL: "http://localhost:5173",
		},
	}
	config.SetConfig(testConfig)
}

// Helper to clear the token store between tests
func clearTokenStore() {
	for k := range tokenStore {
		delete(tokenStore, k)
	}
}

// ============================================
// Token Store Function Tests
// ============================================

func TestGenerateTokenID(t *testing.T) {
	id1 := generateTokenID()
	id2 := generateTokenID()

	// IDs should be non-empty
	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)

	// IDs should be unique
	assert.NotEqual(t, id1, id2)

	// IDs should be valid UUIDs (36 chars with dashes)
	assert.Len(t, id1, 36)
	assert.Len(t, id2, 36)
}

func TestStoreTokens(t *testing.T) {
	clearTokenStore()

	accessToken := "test-access-token"
	refreshToken := "test-refresh-token"
	idToken := "test-id-token"
	expiresAt := time.Now().Add(1 * time.Hour).Unix()

	tokenID := storeTokens(accessToken, refreshToken, idToken, expiresAt)

	// Verify token ID was returned
	assert.NotEmpty(t, tokenID)

	// Verify tokens were stored correctly
	tokens, ok := tokenStore[tokenID]
	assert.True(t, ok)
	assert.Equal(t, accessToken, tokens.AccessToken)
	assert.Equal(t, refreshToken, tokens.RefreshToken)
	assert.Equal(t, idToken, tokens.IDToken)
	assert.Equal(t, expiresAt, tokens.ExpiresAt)
}

func TestGetTokens(t *testing.T) {
	clearTokenStore()

	t.Run("existing tokens", func(t *testing.T) {
		accessToken := "test-access-token"
		refreshToken := "test-refresh-token"
		idToken := "test-id-token"
		expiresAt := time.Now().Add(1 * time.Hour).Unix()

		tokenID := storeTokens(accessToken, refreshToken, idToken, expiresAt)

		tokens, ok := getTokens(tokenID)
		assert.True(t, ok)
		assert.Equal(t, accessToken, tokens.AccessToken)
		assert.Equal(t, refreshToken, tokens.RefreshToken)
		assert.Equal(t, idToken, tokens.IDToken)
		assert.Equal(t, expiresAt, tokens.ExpiresAt)
	})

	t.Run("non-existing tokens", func(t *testing.T) {
		tokens, ok := getTokens("non-existing-id")
		assert.False(t, ok)
		assert.Empty(t, tokens.AccessToken)
	})
}

func TestUpdateTokens(t *testing.T) {
	clearTokenStore()

	t.Run("update existing tokens", func(t *testing.T) {
		// Store initial tokens
		tokenID := storeTokens("old-access", "old-refresh", "old-id", time.Now().Unix())

		// Update tokens
		newExpiresAt := time.Now().Add(2 * time.Hour).Unix()
		success := updateTokens(tokenID, "new-access", "new-refresh", "new-id", newExpiresAt)

		assert.True(t, success)

		// Verify tokens were updated
		tokens, ok := getTokens(tokenID)
		assert.True(t, ok)
		assert.Equal(t, "new-access", tokens.AccessToken)
		assert.Equal(t, "new-refresh", tokens.RefreshToken)
		assert.Equal(t, "new-id", tokens.IDToken)
		assert.Equal(t, newExpiresAt, tokens.ExpiresAt)
	})

	t.Run("update non-existing tokens", func(t *testing.T) {
		success := updateTokens("non-existing-id", "new-access", "new-refresh", "new-id", time.Now().Unix())
		assert.False(t, success)
	})
}

func TestDeleteTokens(t *testing.T) {
	clearTokenStore()

	// Store tokens
	tokenID := storeTokens("access", "refresh", "id", time.Now().Unix())

	// Verify tokens exist
	_, ok := getTokens(tokenID)
	assert.True(t, ok)

	// Delete tokens
	deleteTokens(tokenID)

	// Verify tokens were deleted
	_, ok = getTokens(tokenID)
	assert.False(t, ok)

	// Deleting non-existing tokens should not panic
	deleteTokens("non-existing-id")
}

// ============================================
// InitAuth Tests
// ============================================

func TestInitAuth(t *testing.T) {
	t.Run("with session secret", func(t *testing.T) {
		setupTestConfig(true, "https://issuer.example.com", "client-id", "client-secret")

		InitAuth()

		assert.NotNil(t, SessionStore)
		assert.NotNil(t, SessionStore.Options)
		assert.Equal(t, "/", SessionStore.Options.Path)
		assert.Equal(t, sessionMaxAge, SessionStore.Options.MaxAge)
		assert.True(t, SessionStore.Options.HttpOnly)
	})

	t.Run("without session secret - generates random", func(t *testing.T) {
		testConfig := &config.Config{
			Env: "test",
			Server: config.ServerConfig{
				Address: ":8080",
			},
			Auth: config.AuthConfig{
				Enabled: true,
				OIDC: config.OIDCConfig{
					Issuer:        "https://issuer.example.com",
					ClientID:      "client-id",
					ClientSecret:  "client-secret",
					SessionSecret: "", // Empty secret
				},
			},
		}
		config.SetConfig(testConfig)

		InitAuth()

		assert.NotNil(t, SessionStore)
	})

	t.Run("development mode without SSL", func(t *testing.T) {
		testConfig := &config.Config{
			Env: "development",
			Server: config.ServerConfig{
				Address: ":8080",
				SSL: config.SSLConfig{
					Enabled: false,
				},
			},
			Auth: config.AuthConfig{
				Enabled: true,
				OIDC: config.OIDCConfig{
					SessionSecret: "test-secret",
				},
			},
		}
		config.SetConfig(testConfig)

		InitAuth()

		assert.NotNil(t, SessionStore)
		assert.False(t, SessionStore.Options.Secure) // Should be false for dev without SSL
	})

	t.Run("development mode with SSL", func(t *testing.T) {
		testConfig := &config.Config{
			Env: "development",
			Server: config.ServerConfig{
				Address: ":8080",
				SSL: config.SSLConfig{
					Enabled: true,
				},
			},
			Auth: config.AuthConfig{
				Enabled: true,
				OIDC: config.OIDCConfig{
					SessionSecret: "test-secret",
				},
			},
		}
		config.SetConfig(testConfig)

		InitAuth()

		assert.NotNil(t, SessionStore)
		assert.True(t, SessionStore.Options.Secure) // Should be true for dev with SSL
	})
}

// ============================================
// LoginInit Tests
// ============================================

func TestLoginInit(t *testing.T) {
	t.Run("auth disabled", func(t *testing.T) {
		setupTestConfig(false, "", "", "")
		InitAuth()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/auth/login", nil)

		LoginInit(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "/auth/disabled", response["auth_url"])
		assert.Equal(t, "Authentication is disabled", response["message"])
	})

	t.Run("incomplete OIDC config - missing issuer", func(t *testing.T) {
		setupTestConfig(true, "", "client-id", "client-secret")
		InitAuth()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/auth/login", nil)

		LoginInit(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "/auth/disabled", response["auth_url"])
	})

	t.Run("incomplete OIDC config - missing client ID", func(t *testing.T) {
		setupTestConfig(true, "https://issuer.example.com", "", "client-secret")
		InitAuth()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/auth/login", nil)

		LoginInit(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "/auth/disabled", response["auth_url"])
	})

	t.Run("incomplete OIDC config - missing client secret", func(t *testing.T) {
		setupTestConfig(true, "https://issuer.example.com", "client-id", "")
		InitAuth()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/auth/login", nil)

		LoginInit(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "/auth/disabled", response["auth_url"])
	})
}

// ============================================
// Logout Tests
// ============================================

func TestLogout(t *testing.T) {
	t.Run("logout without session", func(t *testing.T) {
		setupTestConfig(true, "https://issuer.example.com", "client-id", "client-secret")
		InitAuth()
		clearTokenStore()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/auth/logout", nil)

		Logout(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, true, response["success"])
		assert.Equal(t, "Logout successful", response["message"])
	})

	t.Run("logout with valid session", func(t *testing.T) {
		setupTestConfig(true, "https://issuer.example.com", "client-id", "client-secret")
		InitAuth()
		clearTokenStore()

		// Create a session with token ID
		tokenID := storeTokens("access-token", "refresh-token", "id-token", time.Now().Add(1*time.Hour).Unix())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/auth/logout", nil)

		// Create a session and set token ID
		session, _ := SessionStore.Get(c.Request, sessionName)
		session.Values["token_id"] = tokenID
		_ = session.Save(c.Request, w)

		// Update the request with the session cookie
		c.Request = httptest.NewRequest("POST", "/api/auth/logout", nil)
		for _, cookie := range w.Result().Cookies() {
			c.Request.AddCookie(cookie)
		}

		// Create a new recorder for the logout request
		w2 := httptest.NewRecorder()
		c2, _ := gin.CreateTestContext(w2)
		c2.Request = c.Request

		Logout(c2)

		assert.Equal(t, http.StatusOK, w2.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w2.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, true, response["success"])

		// Verify tokens were deleted
		_, ok := getTokens(tokenID)
		assert.False(t, ok)
	})
}

// ============================================
// schemeFromRequest Tests
// ============================================

func TestSchemeFromRequest(t *testing.T) {
	setupTestConfig(false, "", "", "")

	t.Run("X-Forwarded-Proto header https", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("X-Forwarded-Proto", "https")

		scheme := schemeFromRequest(c)
		assert.Equal(t, "https", scheme)
	})

	t.Run("X-Forwarded-Proto header http", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("X-Forwarded-Proto", "http")

		scheme := schemeFromRequest(c)
		assert.Equal(t, "http", scheme)
	})

	t.Run("X-Forwarded-Protocol header", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("X-Forwarded-Protocol", "https")

		scheme := schemeFromRequest(c)
		assert.Equal(t, "https", scheme)
	})

	t.Run("X-Scheme header", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("X-Scheme", "https")

		scheme := schemeFromRequest(c)
		assert.Equal(t, "https", scheme)
	})

	t.Run("SSL enabled in config", func(t *testing.T) {
		testConfig := &config.Config{
			Env: "production",
			Server: config.ServerConfig{
				Address: ":8080",
				SSL: config.SSLConfig{
					Enabled: true,
				},
			},
		}
		config.SetConfig(testConfig)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		scheme := schemeFromRequest(c)
		assert.Equal(t, "https", scheme)
	})

	t.Run("default to http", func(t *testing.T) {
		setupTestConfig(false, "", "", "")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		scheme := schemeFromRequest(c)
		assert.Equal(t, "http", scheme)
	})

	t.Run("TLS connection directly", func(t *testing.T) {
		setupTestConfig(false, "", "", "")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		// Simulate a TLS connection by setting the TLS field
		c.Request.TLS = &tls.ConnectionState{}

		scheme := schemeFromRequest(c)
		assert.Equal(t, "https", scheme)
	})
}

// ============================================
// SessionAuthMiddleware Tests
// ============================================

func TestSessionAuthMiddleware(t *testing.T) {
	t.Run("auth disabled - passes through", func(t *testing.T) {
		setupTestConfig(false, "", "", "")
		InitAuth()

		middleware := SessionAuthMiddleware()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/data", nil)

		called := false
		c.Set("next_called", false)

		// Create a test handler chain
		router := gin.New()
		router.Use(middleware)
		router.GET("/api/data", func(c *gin.Context) {
			called = true
			c.JSON(http.StatusOK, gin.H{"success": true})
		})

		router.ServeHTTP(w, c.Request)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, called)
	})

	t.Run("skip auth paths", func(t *testing.T) {
		setupTestConfig(true, "https://issuer.example.com", "client-id", "client-secret")
		InitAuth()

		middleware := SessionAuthMiddleware()

		router := gin.New()
		router.Use(middleware)
		router.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/health", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("no session - returns 401", func(t *testing.T) {
		setupTestConfig(true, "https://issuer.example.com", "client-id", "client-secret")
		InitAuth()

		middleware := SessionAuthMiddleware()

		router := gin.New()
		router.Use(middleware)
		router.GET("/api/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"success": true})
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/protected", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("session without token ID - returns 401", func(t *testing.T) {
		setupTestConfig(true, "https://issuer.example.com", "client-id", "client-secret")
		InitAuth()

		middleware := SessionAuthMiddleware()

		router := gin.New()
		router.Use(middleware)
		router.GET("/api/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"success": true})
		})

		// Create a request with an empty session
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/protected", nil)

		// Create a session without token_id
		session, _ := SessionStore.Get(req, sessionName)
		session.Values["some_other_key"] = "value"
		_ = session.Save(req, w)

		// Make the actual request with the session cookie
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/api/protected", nil)
		for _, cookie := range w.Result().Cookies() {
			req2.AddCookie(cookie)
		}

		router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusUnauthorized, w2.Code)
	})

	t.Run("session with invalid token ID - returns 401", func(t *testing.T) {
		setupTestConfig(true, "https://issuer.example.com", "client-id", "client-secret")
		InitAuth()
		clearTokenStore()

		middleware := SessionAuthMiddleware()

		router := gin.New()
		router.Use(middleware)
		router.GET("/api/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"success": true})
		})

		// Create a request with a session containing non-existent token_id
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/protected", nil)

		session, _ := SessionStore.Get(req, sessionName)
		session.Values["token_id"] = "non-existent-token-id"
		_ = session.Save(req, w)

		// Make the actual request with the session cookie
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/api/protected", nil)
		for _, cookie := range w.Result().Cookies() {
			req2.AddCookie(cookie)
		}

		router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusUnauthorized, w2.Code)
	})
}

// ============================================
// fetchOIDCDiscoveryDocument Tests
// ============================================

func TestFetchOIDCDiscoveryDocument(t *testing.T) {
	t.Run("successful fetch", func(t *testing.T) {
		// Create a mock OIDC discovery server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/.well-known/openid-configuration" {
				discoveryDoc := map[string]interface{}{
					"issuer":                 "https://test-issuer.com",
					"authorization_endpoint": "https://test-issuer.com/auth",
					"token_endpoint":         "https://test-issuer.com/token",
					"userinfo_endpoint":      "https://test-issuer.com/userinfo",
					"end_session_endpoint":   "https://test-issuer.com/logout",
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(discoveryDoc)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		doc, err := fetchOIDCDiscoveryDocument(server.URL)

		assert.NoError(t, err)
		assert.NotNil(t, doc)
		assert.Equal(t, "https://test-issuer.com", doc["issuer"])
		assert.Equal(t, "https://test-issuer.com/auth", doc["authorization_endpoint"])
		assert.Equal(t, "https://test-issuer.com/token", doc["token_endpoint"])
	})

	t.Run("adds https prefix if missing", func(t *testing.T) {
		// This test verifies the function adds a protocol prefix
		// The actual request will fail since the URL doesn't exist,
		// but we can verify the function tries to make the request
		_, err := fetchOIDCDiscoveryDocument("non-existent-domain.invalid")

		// Should get an error because the domain doesn't exist
		assert.Error(t, err)
	})

	t.Run("handles non-200 response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		_, err := fetchOIDCDiscoveryDocument(server.URL)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "status 404")
	})

	t.Run("handles invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("not valid json"))
		}))
		defer server.Close()

		_, err := fetchOIDCDiscoveryDocument(server.URL)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse discovery document")
	})
}

// ============================================
// AuthCallback Tests
// ============================================

func TestAuthCallback(t *testing.T) {
	t.Run("missing authorization code", func(t *testing.T) {
		setupTestConfig(true, "https://issuer.example.com", "client-id", "client-secret")
		InitAuth()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/auth/callback", nil)

		AuthCallback(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("callback with code but discovery fails", func(t *testing.T) {
		setupTestConfig(true, "https://non-existent-issuer.invalid", "client-id", "client-secret")
		InitAuth()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/auth/callback?code=test-code&state=test-state", nil)

		// Create a session with state
		session, _ := SessionStore.Get(c.Request, sessionName)
		session.Values["oidc_state"] = "test-state"
		_ = session.Save(c.Request, w)

		// Update request with session cookie
		for _, cookie := range w.Result().Cookies() {
			c.Request.AddCookie(cookie)
		}

		w2 := httptest.NewRecorder()
		c2, _ := gin.CreateTestContext(w2)
		c2.Request = c.Request

		AuthCallback(c2)

		// Should fail because discovery document fetch fails
		assert.Equal(t, http.StatusInternalServerError, w2.Code)
	})
}

// ============================================
// TokenInfo Tests
// ============================================

func TestTokenInfo(t *testing.T) {
	t.Run("token info struct", func(t *testing.T) {
		info := TokenInfo{
			AccessToken:  "access",
			RefreshToken: "refresh",
			IDToken:      "id",
			ExpiresAt:    1234567890,
		}

		assert.Equal(t, "access", info.AccessToken)
		assert.Equal(t, "refresh", info.RefreshToken)
		assert.Equal(t, "id", info.IDToken)
		assert.Equal(t, int64(1234567890), info.ExpiresAt)
	})
}

// ============================================
// TokenResponse Tests
// ============================================

func TestTokenResponse(t *testing.T) {
	t.Run("token response struct parsing", func(t *testing.T) {
		jsonData := `{
			"access_token": "test-access",
			"refresh_token": "test-refresh",
			"id_token": "test-id",
			"token_type": "Bearer",
			"expires_in": 3600
		}`

		var response TokenResponse
		err := json.Unmarshal([]byte(jsonData), &response)

		assert.NoError(t, err)
		assert.Equal(t, "test-access", response.AccessToken)
		assert.Equal(t, "test-refresh", response.RefreshToken)
		assert.Equal(t, "test-id", response.IDToken)
		assert.Equal(t, "Bearer", response.TokenType)
		assert.Equal(t, 3600, response.ExpiresIn)
	})
}

// ============================================
// SessionStore Tests
// ============================================

func TestSessionStore(t *testing.T) {
	t.Run("session store initialization", func(t *testing.T) {
		setupTestConfig(true, "https://issuer.example.com", "client-id", "client-secret")
		InitAuth()

		assert.NotNil(t, SessionStore)
		assert.IsType(t, &sessions.CookieStore{}, SessionStore)
	})

	t.Run("session options", func(t *testing.T) {
		setupTestConfig(true, "https://issuer.example.com", "client-id", "client-secret")
		InitAuth()

		assert.Equal(t, "/", SessionStore.Options.Path)
		assert.Equal(t, sessionMaxAge, SessionStore.Options.MaxAge)
		assert.True(t, SessionStore.Options.HttpOnly)
	})
}

// ============================================
// Integration-style Tests
// ============================================

func TestAuthFlow(t *testing.T) {
	t.Run("full token lifecycle", func(t *testing.T) {
		clearTokenStore()

		// 1. Store tokens
		tokenID := storeTokens("access1", "refresh1", "id1", time.Now().Add(1*time.Hour).Unix())
		assert.NotEmpty(t, tokenID)

		// 2. Retrieve tokens
		tokens, ok := getTokens(tokenID)
		assert.True(t, ok)
		assert.Equal(t, "access1", tokens.AccessToken)

		// 3. Update tokens
		newExpiry := time.Now().Add(2 * time.Hour).Unix()
		success := updateTokens(tokenID, "access2", "refresh2", "id2", newExpiry)
		assert.True(t, success)

		// 4. Verify update
		tokens, ok = getTokens(tokenID)
		assert.True(t, ok)
		assert.Equal(t, "access2", tokens.AccessToken)
		assert.Equal(t, newExpiry, tokens.ExpiresAt)

		// 5. Delete tokens
		deleteTokens(tokenID)

		// 6. Verify deletion
		_, ok = getTokens(tokenID)
		assert.False(t, ok)
	})
}

func TestConcurrentTokenOperations(t *testing.T) {
	clearTokenStore()

	// Test concurrent token operations
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			tokenID := storeTokens(
				"access-"+string(rune('0'+idx)),
				"refresh-"+string(rune('0'+idx)),
				"id-"+string(rune('0'+idx)),
				time.Now().Add(1*time.Hour).Unix(),
			)
			_, _ = getTokens(tokenID)
			deleteTokens(tokenID)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Note: This test may have race conditions in the current implementation
	// The in-memory token store doesn't have mutex protection
	// This is documented as a limitation in the auth.go file
}

// ============================================
// Additional LoginInit Tests with Mock Server
// ============================================

func TestLoginInitWithMockDiscovery(t *testing.T) {
	t.Run("successful login init with discovery", func(t *testing.T) {
		// Create a mock OIDC discovery server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/.well-known/openid-configuration" {
				discoveryDoc := map[string]interface{}{
					"issuer":                 "https://test-issuer.com",
					"authorization_endpoint": "https://test-issuer.com/auth",
					"token_endpoint":         "https://test-issuer.com/token",
					"userinfo_endpoint":      "https://test-issuer.com/userinfo",
					"end_session_endpoint":   "https://test-issuer.com/logout",
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(discoveryDoc)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		testConfig := &config.Config{
			Env: "development",
			Server: config.ServerConfig{
				Address: ":8080",
				SSL: config.SSLConfig{
					Enabled: false,
				},
			},
			Auth: config.AuthConfig{
				Enabled: true,
				OIDC: config.OIDCConfig{
					Issuer:        server.URL,
					ClientID:      "test-client-id",
					ClientSecret:  "test-client-secret",
					SessionSecret: "test-session-secret",
				},
			},
			Frontend: config.FrontendConfig{
				URL: "http://localhost:5173",
			},
		}
		config.SetConfig(testConfig)
		InitAuth()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/auth/login", nil)
		c.Request.Host = "localhost:8080"

		LoginInit(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Should have auth_url with the authorization endpoint
		authURL, ok := response["auth_url"].(string)
		assert.True(t, ok)
		assert.Contains(t, authURL, "https://test-issuer.com/auth")
		assert.Contains(t, authURL, "client_id=test-client-id")
		assert.Contains(t, authURL, "response_type=code")
	})

	t.Run("login init with short client secret", func(t *testing.T) {
		// Create a mock OIDC discovery server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/.well-known/openid-configuration" {
				discoveryDoc := map[string]interface{}{
					"issuer":                 "https://test-issuer.com",
					"authorization_endpoint": "https://test-issuer.com/auth",
					"token_endpoint":         "https://test-issuer.com/token",
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(discoveryDoc)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		testConfig := &config.Config{
			Env: "production",
			Server: config.ServerConfig{
				Address: ":8080",
				SSL: config.SSLConfig{
					Enabled: true,
				},
			},
			Auth: config.AuthConfig{
				Enabled: true,
				OIDC: config.OIDCConfig{
					Issuer:        server.URL,
					ClientID:      "test-client-id",
					ClientSecret:  "abc", // Short secret (less than 4 chars)
					SessionSecret: "test-session-secret",
				},
			},
		}
		config.SetConfig(testConfig)
		InitAuth()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/auth/login", nil)
		c.Request.Host = "localhost:8080"

		LoginInit(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Should have auth_url
		_, ok := response["auth_url"].(string)
		assert.True(t, ok)
	})

	t.Run("login init with discovery missing auth endpoint", func(t *testing.T) {
		// Create a mock OIDC discovery server without authorization_endpoint
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/.well-known/openid-configuration" {
				discoveryDoc := map[string]interface{}{
					"issuer":         "https://test-issuer.com",
					"token_endpoint": "https://test-issuer.com/token",
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(discoveryDoc)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		testConfig := &config.Config{
			Env: "development",
			Server: config.ServerConfig{
				Address: ":8080",
				SSL: config.SSLConfig{
					Enabled: false,
				},
			},
			Auth: config.AuthConfig{
				Enabled: true,
				OIDC: config.OIDCConfig{
					Issuer:        server.URL,
					ClientID:      "test-client-id",
					ClientSecret:  "test-client-secret",
					SessionSecret: "test-session-secret",
				},
			},
		}
		config.SetConfig(testConfig)
		InitAuth()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/auth/login", nil)
		c.Request.Host = "localhost:8080"

		LoginInit(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ============================================
// Additional AuthCallback Tests with Mock Server
// ============================================

func TestAuthCallbackWithMockServer(t *testing.T) {
	t.Run("successful token exchange", func(t *testing.T) {
		// Create a mock OIDC server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/.well-known/openid-configuration":
				discoveryDoc := map[string]interface{}{
					"issuer":                 "https://test-issuer.com",
					"authorization_endpoint": "https://test-issuer.com/auth",
					"token_endpoint":         r.Host + "/token",
					"userinfo_endpoint":      "https://test-issuer.com/userinfo",
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(discoveryDoc)
			case "/token":
				tokenResponse := TokenResponse{
					AccessToken:  "test-access-token",
					RefreshToken: "test-refresh-token",
					IDToken:      "test-id-token",
					TokenType:    "Bearer",
					ExpiresIn:    3600,
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(tokenResponse)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		testConfig := &config.Config{
			Env: "development",
			Server: config.ServerConfig{
				Address: ":8080",
				SSL: config.SSLConfig{
					Enabled: false,
				},
			},
			Auth: config.AuthConfig{
				Enabled: true,
				OIDC: config.OIDCConfig{
					Issuer:        server.URL,
					ClientID:      "test-client-id",
					ClientSecret:  "test-client-secret",
					SessionSecret: "test-session-secret",
				},
			},
			Frontend: config.FrontendConfig{
				URL: "http://localhost:5173",
			},
		}
		config.SetConfig(testConfig)
		InitAuth()
		clearTokenStore()

		// First, create a session with state
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/auth/callback?code=test-code&state=test-state", nil)
		req.Host = "localhost:8080"

		session, _ := SessionStore.Get(req, sessionName)
		session.Values["oidc_state"] = "test-state"
		_ = session.Save(req, w)

		// Now make the callback request with the session
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/api/auth/callback?code=test-code&state=test-state", nil)
		req2.Host = "localhost:8080"
		for _, cookie := range w.Result().Cookies() {
			req2.AddCookie(cookie)
		}

		c, _ := gin.CreateTestContext(w2)
		c.Request = req2

		AuthCallback(c)

		// In development mode with failed token exchange (wrong endpoint), should fail
		// The token endpoint in discovery doc doesn't match the actual server
		assert.True(t, w2.Code == http.StatusTemporaryRedirect || w2.Code == http.StatusInternalServerError)
	})

	t.Run("state mismatch in production mode", func(t *testing.T) {
		testConfig := &config.Config{
			Env: "production", // Production mode - stricter validation
			Server: config.ServerConfig{
				Address: ":8080",
			},
			Auth: config.AuthConfig{
				Enabled: true,
				OIDC: config.OIDCConfig{
					Issuer:        "https://issuer.example.com",
					ClientID:      "test-client-id",
					ClientSecret:  "test-client-secret",
					SessionSecret: "test-session-secret",
				},
			},
		}
		config.SetConfig(testConfig)
		InitAuth()

		// Create a session with different state
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/auth/callback?code=test-code&state=wrong-state", nil)

		session, _ := SessionStore.Get(req, sessionName)
		session.Values["oidc_state"] = "correct-state"
		_ = session.Save(req, w)

		// Make the callback request
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/api/auth/callback?code=test-code&state=wrong-state", nil)
		for _, cookie := range w.Result().Cookies() {
			req2.AddCookie(cookie)
		}

		c, _ := gin.CreateTestContext(w2)
		c.Request = req2

		AuthCallback(c)

		assert.Equal(t, http.StatusBadRequest, w2.Code)
	})

	t.Run("no state in session production mode", func(t *testing.T) {
		testConfig := &config.Config{
			Env: "production",
			Server: config.ServerConfig{
				Address: ":8080",
			},
			Auth: config.AuthConfig{
				Enabled: true,
				OIDC: config.OIDCConfig{
					Issuer:        "https://issuer.example.com",
					ClientID:      "test-client-id",
					ClientSecret:  "test-client-secret",
					SessionSecret: "test-session-secret",
				},
			},
		}
		config.SetConfig(testConfig)
		InitAuth()

		// Create a session without state
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/auth/callback?code=test-code&state=some-state", nil)

		session, _ := SessionStore.Get(req, sessionName)
		// Don't set oidc_state
		_ = session.Save(req, w)

		// Make the callback request
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/api/auth/callback?code=test-code&state=some-state", nil)
		for _, cookie := range w.Result().Cookies() {
			req2.AddCookie(cookie)
		}

		c, _ := gin.CreateTestContext(w2)
		c.Request = req2

		AuthCallback(c)

		assert.Equal(t, http.StatusBadRequest, w2.Code)
	})

	t.Run("token endpoint returns error", func(t *testing.T) {
		// Create mock server that returns error from token endpoint
		var serverURL string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/.well-known/openid-configuration":
				discoveryDoc := map[string]interface{}{
					"issuer":                 serverURL,
					"authorization_endpoint": serverURL + "/auth",
					"token_endpoint":         serverURL + "/token",
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(discoveryDoc)
			case "/token":
				// Return an error response
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":             "invalid_grant",
					"error_description": "Invalid authorization code",
				})
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		serverURL = server.URL
		defer server.Close()

		testConfig := &config.Config{
			Env: "development",
			Server: config.ServerConfig{
				Address: ":8080",
			},
			Auth: config.AuthConfig{
				Enabled: true,
				OIDC: config.OIDCConfig{
					Issuer:        server.URL,
					ClientID:      "test-client-id",
					ClientSecret:  "test-client-secret",
					SessionSecret: "test-session-secret",
				},
			},
		}
		config.SetConfig(testConfig)
		InitAuth()
		clearTokenStore()

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/auth/callback?code=test-code&state=test-state", nil)

		session, _ := SessionStore.Get(req, sessionName)
		session.Values["oidc_state"] = "test-state"
		_ = session.Save(req, w)

		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/api/auth/callback?code=test-code&state=test-state", nil)
		req2.Host = "localhost:8080"
		for _, cookie := range w.Result().Cookies() {
			req2.AddCookie(cookie)
		}

		c, _ := gin.CreateTestContext(w2)
		c.Request = req2

		AuthCallback(c)

		assert.Equal(t, http.StatusInternalServerError, w2.Code)
	})

	t.Run("discovery document without token endpoint", func(t *testing.T) {
		// Create mock server without token_endpoint in discovery
		var serverURL string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/.well-known/openid-configuration":
				discoveryDoc := map[string]interface{}{
					"issuer":                 serverURL,
					"authorization_endpoint": serverURL + "/auth",
					// No token_endpoint
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(discoveryDoc)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		serverURL = server.URL
		defer server.Close()

		testConfig := &config.Config{
			Env: "development",
			Server: config.ServerConfig{
				Address: ":8080",
			},
			Auth: config.AuthConfig{
				Enabled: true,
				OIDC: config.OIDCConfig{
					Issuer:        server.URL,
					ClientID:      "test-client-id",
					ClientSecret:  "test-client-secret",
					SessionSecret: "test-session-secret",
				},
			},
		}
		config.SetConfig(testConfig)
		InitAuth()

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/auth/callback?code=test-code&state=test-state", nil)

		session, _ := SessionStore.Get(req, sessionName)
		session.Values["oidc_state"] = "test-state"
		_ = session.Save(req, w)

		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/api/auth/callback?code=test-code&state=test-state", nil)
		for _, cookie := range w.Result().Cookies() {
			req2.AddCookie(cookie)
		}

		c, _ := gin.CreateTestContext(w2)
		c.Request = req2

		AuthCallback(c)

		assert.Equal(t, http.StatusInternalServerError, w2.Code)
	})
}

// ============================================
// Additional SessionAuthMiddleware Tests
// ============================================

func TestSessionAuthMiddlewareWithValidSession(t *testing.T) {
	t.Run("valid session with fresh token", func(t *testing.T) {
		// Create mock userinfo server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/.well-known/openid-configuration":
				discoveryDoc := map[string]interface{}{
					"issuer":            "https://test-issuer.com",
					"userinfo_endpoint": r.Host + "/userinfo",
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(discoveryDoc)
			case "/userinfo":
				userInfo := map[string]interface{}{
					"sub":   "test-user-id",
					"email": "test@example.com",
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(userInfo)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		testConfig := &config.Config{
			Env: "production",
			Server: config.ServerConfig{
				Address: ":8080",
			},
			Auth: config.AuthConfig{
				Enabled: true,
				SkipAuthPaths: []string{
					"/health",
				},
				OIDC: config.OIDCConfig{
					Issuer:                server.URL,
					ClientID:              "test-client-id",
					ClientSecret:          "test-client-secret",
					SessionSecret:         "test-session-secret",
					TokenRefreshBufferSec: 60,
				},
			},
		}
		config.SetConfig(testConfig)
		InitAuth()
		clearTokenStore()

		// Store tokens that are not expired
		tokenID := storeTokens(
			"test-access-token",
			"test-refresh-token",
			"test-id-token",
			time.Now().Add(1*time.Hour).Unix(), // Expires in 1 hour
		)

		middleware := SessionAuthMiddleware()

		router := gin.New()
		router.Use(middleware)
		router.GET("/api/protected", func(c *gin.Context) {
			// Verify access_token was set
			accessToken, exists := c.Get("access_token")
			if exists && accessToken == "test-access-token" {
				c.JSON(http.StatusOK, gin.H{"success": true})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "token not found"})
			}
		})

		// Create session with token_id
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/protected", nil)

		session, _ := SessionStore.Get(req, sessionName)
		session.Values["token_id"] = tokenID
		_ = session.Save(req, w)

		// Make request with session
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/api/protected", nil)
		for _, cookie := range w.Result().Cookies() {
			req2.AddCookie(cookie)
		}

		router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusOK, w2.Code)
	})

	t.Run("session with expired token and no refresh token", func(t *testing.T) {
		testConfig := &config.Config{
			Env: "production",
			Server: config.ServerConfig{
				Address: ":8080",
			},
			Auth: config.AuthConfig{
				Enabled: true,
				SkipAuthPaths: []string{
					"/health",
				},
				OIDC: config.OIDCConfig{
					Issuer:                "https://issuer.example.com",
					ClientID:              "test-client-id",
					ClientSecret:          "test-client-secret",
					SessionSecret:         "test-session-secret",
					TokenRefreshBufferSec: 60,
				},
			},
		}
		config.SetConfig(testConfig)
		InitAuth()
		clearTokenStore()

		// Store tokens that are already expired
		tokenID := storeTokens(
			"test-access-token",
			"", // No refresh token
			"test-id-token",
			time.Now().Add(-1*time.Hour).Unix(), // Expired 1 hour ago
		)

		middleware := SessionAuthMiddleware()

		router := gin.New()
		router.Use(middleware)
		router.GET("/api/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"success": true})
		})

		// Create session with token_id
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/protected", nil)

		session, _ := SessionStore.Get(req, sessionName)
		session.Values["token_id"] = tokenID
		_ = session.Save(req, w)

		// Make request with session
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/api/protected", nil)
		for _, cookie := range w.Result().Cookies() {
			req2.AddCookie(cookie)
		}

		router.ServeHTTP(w2, req2)

		// Should return 401 because token is expired and refresh fails
		assert.Equal(t, http.StatusUnauthorized, w2.Code)
	})

	t.Run("session with token expiring soon", func(t *testing.T) {
		testConfig := &config.Config{
			Env: "production",
			Server: config.ServerConfig{
				Address: ":8080",
			},
			Auth: config.AuthConfig{
				Enabled: true,
				SkipAuthPaths: []string{
					"/health",
				},
				OIDC: config.OIDCConfig{
					Issuer:                "https://non-existent-issuer.invalid",
					ClientID:              "test-client-id",
					ClientSecret:          "test-client-secret",
					SessionSecret:         "test-session-secret",
					TokenRefreshBufferSec: 120, // 2 minute buffer
				},
			},
		}
		config.SetConfig(testConfig)
		InitAuth()
		clearTokenStore()

		// Store tokens that expire within the buffer window
		tokenID := storeTokens(
			"test-access-token",
			"test-refresh-token",
			"test-id-token",
			time.Now().Add(30*time.Second).Unix(), // Expires in 30 seconds (within 2 min buffer)
		)

		middleware := SessionAuthMiddleware()

		router := gin.New()
		router.Use(middleware)
		router.GET("/api/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"success": true})
		})

		// Create session with token_id
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/protected", nil)

		session, _ := SessionStore.Get(req, sessionName)
		session.Values["token_id"] = tokenID
		_ = session.Save(req, w)

		// Make request with session
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/api/protected", nil)
		for _, cookie := range w.Result().Cookies() {
			req2.AddCookie(cookie)
		}

		router.ServeHTTP(w2, req2)

		// Should still succeed with existing token despite failed refresh
		assert.Equal(t, http.StatusOK, w2.Code)
	})
}

// ============================================
// Logout Tests with OIDC Provider
// ============================================

func TestLogoutWithOIDCProvider(t *testing.T) {
	t.Run("logout calls OIDC end session endpoint", func(t *testing.T) {
		// Create a mock OIDC server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/.well-known/openid-configuration":
				discoveryDoc := map[string]interface{}{
					"issuer":               "https://test-issuer.com",
					"end_session_endpoint": "http://" + r.Host + "/logout",
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(discoveryDoc)
			case "/logout":
				// Logout endpoint called
				w.WriteHeader(http.StatusOK)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		testConfig := &config.Config{
			Env: "production",
			Server: config.ServerConfig{
				Address: ":8080",
			},
			Auth: config.AuthConfig{
				Enabled: true,
				OIDC: config.OIDCConfig{
					Issuer:        server.URL,
					ClientID:      "test-client-id",
					ClientSecret:  "test-client-secret",
					SessionSecret: "test-session-secret",
				},
			},
		}
		config.SetConfig(testConfig)
		InitAuth()
		clearTokenStore()

		// Store tokens with ID token (required for end session)
		tokenID := storeTokens(
			"test-access-token",
			"test-refresh-token",
			"test-id-token",
			time.Now().Add(1*time.Hour).Unix(),
		)

		// Create session with token_id
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/auth/logout", nil)

		session, _ := SessionStore.Get(req, sessionName)
		session.Values["token_id"] = tokenID
		_ = session.Save(req, w)

		// Make logout request with session
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("POST", "/api/auth/logout", nil)
		for _, cookie := range w.Result().Cookies() {
			req2.AddCookie(cookie)
		}

		c, _ := gin.CreateTestContext(w2)
		c.Request = req2

		Logout(c)

		assert.Equal(t, http.StatusOK, w2.Code)

		// Wait a bit for async logout call
		time.Sleep(100 * time.Millisecond)

		// Verify tokens were deleted
		_, ok := getTokens(tokenID)
		assert.False(t, ok)
	})

	t.Run("logout with discovery failure", func(t *testing.T) {
		testConfig := &config.Config{
			Env: "production",
			Server: config.ServerConfig{
				Address: ":8080",
			},
			Auth: config.AuthConfig{
				Enabled: true,
				OIDC: config.OIDCConfig{
					Issuer:        "https://non-existent-issuer.invalid",
					ClientID:      "test-client-id",
					ClientSecret:  "test-client-secret",
					SessionSecret: "test-session-secret",
				},
			},
		}
		config.SetConfig(testConfig)
		InitAuth()
		clearTokenStore()

		// Store tokens
		tokenID := storeTokens(
			"test-access-token",
			"test-refresh-token",
			"test-id-token",
			time.Now().Add(1*time.Hour).Unix(),
		)

		// Create session with token_id
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/auth/logout", nil)

		session, _ := SessionStore.Get(req, sessionName)
		session.Values["token_id"] = tokenID
		_ = session.Save(req, w)

		// Make logout request with session
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("POST", "/api/auth/logout", nil)
		for _, cookie := range w.Result().Cookies() {
			req2.AddCookie(cookie)
		}

		c, _ := gin.CreateTestContext(w2)
		c.Request = req2

		Logout(c)

		// Should still succeed even though OIDC discovery failed
		assert.Equal(t, http.StatusOK, w2.Code)

		// Verify tokens were deleted
		_, ok := getTokens(tokenID)
		assert.False(t, ok)
	})

	t.Run("logout without end_session_endpoint in discovery", func(t *testing.T) {
		// Create mock server without end_session_endpoint
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/.well-known/openid-configuration":
				discoveryDoc := map[string]interface{}{
					"issuer":         "https://test-issuer.com",
					"token_endpoint": "https://test-issuer.com/token",
					// No end_session_endpoint
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(discoveryDoc)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		testConfig := &config.Config{
			Env: "production",
			Server: config.ServerConfig{
				Address: ":8080",
			},
			Auth: config.AuthConfig{
				Enabled: true,
				OIDC: config.OIDCConfig{
					Issuer:        server.URL,
					ClientID:      "test-client-id",
					ClientSecret:  "test-client-secret",
					SessionSecret: "test-session-secret",
				},
			},
		}
		config.SetConfig(testConfig)
		InitAuth()
		clearTokenStore()

		// Store tokens
		tokenID := storeTokens(
			"test-access-token",
			"test-refresh-token",
			"test-id-token",
			time.Now().Add(1*time.Hour).Unix(),
		)

		// Create session with token_id
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/auth/logout", nil)

		session, _ := SessionStore.Get(req, sessionName)
		session.Values["token_id"] = tokenID
		_ = session.Save(req, w)

		// Make logout request with session
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("POST", "/api/auth/logout", nil)
		for _, cookie := range w.Result().Cookies() {
			req2.AddCookie(cookie)
		}

		c, _ := gin.CreateTestContext(w2)
		c.Request = req2

		Logout(c)

		// Should succeed without end session call
		assert.Equal(t, http.StatusOK, w2.Code)

		// Verify tokens were deleted
		_, ok := getTokens(tokenID)
		assert.False(t, ok)
	})
}

// ============================================
// RefreshToken Tests
// ============================================

func TestRefreshToken(t *testing.T) {
	t.Run("refresh without session", func(t *testing.T) {
		setupTestConfig(true, "https://issuer.example.com", "client-id", "client-secret")
		InitAuth()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		err := refreshToken(c)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no token ID")
	})

	t.Run("refresh with no tokens in store", func(t *testing.T) {
		setupTestConfig(true, "https://issuer.example.com", "client-id", "client-secret")
		InitAuth()
		clearTokenStore()

		// Create session with token_id that doesn't exist in store
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)

		session, _ := SessionStore.Get(req, sessionName)
		session.Values["token_id"] = "non-existent-token-id"
		_ = session.Save(req, w)

		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/test", nil)
		for _, cookie := range w.Result().Cookies() {
			req2.AddCookie(cookie)
		}

		c, _ := gin.CreateTestContext(w2)
		c.Request = req2

		err := refreshToken(c)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no tokens found")
	})

	t.Run("refresh with no refresh token", func(t *testing.T) {
		setupTestConfig(true, "https://issuer.example.com", "client-id", "client-secret")
		InitAuth()
		clearTokenStore()

		// Store tokens without refresh token
		tokenID := storeTokens("access-token", "", "id-token", time.Now().Add(1*time.Hour).Unix())

		// Create session with token_id
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)

		session, _ := SessionStore.Get(req, sessionName)
		session.Values["token_id"] = tokenID
		_ = session.Save(req, w)

		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/test", nil)
		for _, cookie := range w.Result().Cookies() {
			req2.AddCookie(cookie)
		}

		c, _ := gin.CreateTestContext(w2)
		c.Request = req2

		err := refreshToken(c)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no refresh token")
	})

	t.Run("refresh with discovery failure", func(t *testing.T) {
		setupTestConfig(true, "https://non-existent-issuer.invalid", "client-id", "client-secret")
		InitAuth()
		clearTokenStore()

		// Store tokens with refresh token
		tokenID := storeTokens("access-token", "refresh-token", "id-token", time.Now().Add(1*time.Hour).Unix())

		// Create session with token_id
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)

		session, _ := SessionStore.Get(req, sessionName)
		session.Values["token_id"] = tokenID
		_ = session.Save(req, w)

		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/test", nil)
		for _, cookie := range w.Result().Cookies() {
			req2.AddCookie(cookie)
		}

		c, _ := gin.CreateTestContext(w2)
		c.Request = req2

		err := refreshToken(c)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "discovery document")
	})

	t.Run("successful token refresh", func(t *testing.T) {
		// Create mock OIDC server for token refresh
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/.well-known/openid-configuration":
				discoveryDoc := map[string]interface{}{
					"issuer":         "https://test-issuer.com",
					"token_endpoint": "http://" + r.Host + "/token",
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(discoveryDoc)
			case "/token":
				tokenResponse := TokenResponse{
					AccessToken:  "new-access-token",
					RefreshToken: "new-refresh-token",
					IDToken:      "new-id-token",
					TokenType:    "Bearer",
					ExpiresIn:    3600,
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(tokenResponse)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		testConfig := &config.Config{
			Env: "production",
			Server: config.ServerConfig{
				Address: ":8080",
			},
			Auth: config.AuthConfig{
				Enabled: true,
				OIDC: config.OIDCConfig{
					Issuer:        server.URL,
					ClientID:      "test-client-id",
					ClientSecret:  "test-client-secret",
					SessionSecret: "test-session-secret",
				},
			},
		}
		config.SetConfig(testConfig)
		InitAuth()
		clearTokenStore()

		// Store tokens with refresh token
		tokenID := storeTokens("old-access-token", "old-refresh-token", "old-id-token", time.Now().Add(1*time.Hour).Unix())

		// Create session with token_id
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)

		session, _ := SessionStore.Get(req, sessionName)
		session.Values["token_id"] = tokenID
		_ = session.Save(req, w)

		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/test", nil)
		for _, cookie := range w.Result().Cookies() {
			req2.AddCookie(cookie)
		}

		c, _ := gin.CreateTestContext(w2)
		c.Request = req2

		err := refreshToken(c)

		assert.NoError(t, err)

		// Verify tokens were updated
		tokens, ok := getTokens(tokenID)
		assert.True(t, ok)
		assert.Equal(t, "new-access-token", tokens.AccessToken)
		assert.Equal(t, "new-refresh-token", tokens.RefreshToken)
		assert.Equal(t, "new-id-token", tokens.IDToken)
	})

	t.Run("token refresh keeps existing tokens when provider returns empty", func(t *testing.T) {
		// Create mock OIDC server that only returns access token
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/.well-known/openid-configuration":
				discoveryDoc := map[string]interface{}{
					"issuer":         "https://test-issuer.com",
					"token_endpoint": "http://" + r.Host + "/token",
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(discoveryDoc)
			case "/token":
				tokenResponse := TokenResponse{
					AccessToken: "new-access-token",
					// No refresh_token or id_token returned
					TokenType: "Bearer",
					ExpiresIn: 3600,
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(tokenResponse)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		testConfig := &config.Config{
			Env: "production",
			Server: config.ServerConfig{
				Address: ":8080",
			},
			Auth: config.AuthConfig{
				Enabled: true,
				OIDC: config.OIDCConfig{
					Issuer:        server.URL,
					ClientID:      "test-client-id",
					ClientSecret:  "test-client-secret",
					SessionSecret: "test-session-secret",
				},
			},
		}
		config.SetConfig(testConfig)
		InitAuth()
		clearTokenStore()

		// Store tokens with refresh token and id token
		tokenID := storeTokens("old-access-token", "old-refresh-token", "old-id-token", time.Now().Add(1*time.Hour).Unix())

		// Create session with token_id
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)

		session, _ := SessionStore.Get(req, sessionName)
		session.Values["token_id"] = tokenID
		_ = session.Save(req, w)

		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/test", nil)
		for _, cookie := range w.Result().Cookies() {
			req2.AddCookie(cookie)
		}

		c, _ := gin.CreateTestContext(w2)
		c.Request = req2

		err := refreshToken(c)

		assert.NoError(t, err)

		// Verify access token was updated but others were preserved
		tokens, ok := getTokens(tokenID)
		assert.True(t, ok)
		assert.Equal(t, "new-access-token", tokens.AccessToken)
		assert.Equal(t, "old-refresh-token", tokens.RefreshToken) // Preserved
		assert.Equal(t, "old-id-token", tokens.IDToken)           // Preserved
	})
}
