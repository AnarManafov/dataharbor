package controller

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/stretchr/testify/assert"

	"github.com/AnarManafov/dataharbor/app/config"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestGetCurrentUser_NoSession(t *testing.T) {
	// Set up config
	testConfig := &config.Config{
		Env: "test",
		Auth: config.AuthConfig{
			Enabled: true,
			OIDC: config.OIDCConfig{
				SessionSecret: "test-secret-key-1234567890123456",
			},
		},
	}
	config.SetConfig(testConfig)

	// Initialize auth to set up session store
	InitAuth()

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/user", nil)

	// Call the function
	GetCurrentUser(c)

	// Should return unauthorized since no session
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Not authenticated")
}

func TestGetCurrentUser_SessionWithoutTokenID(t *testing.T) {
	// Set up config
	testConfig := &config.Config{
		Env: "test",
		Auth: config.AuthConfig{
			Enabled: true,
			OIDC: config.OIDCConfig{
				SessionSecret: "test-secret-key-1234567890123456",
			},
		},
	}
	config.SetConfig(testConfig)

	// Use a simple in-memory cookie store for testing
	SessionStore = sessions.NewCookieStore([]byte("test-secret-key-1234567890123456"))

	// Create test context with session but no token_id
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/user", nil)

	// Get session and save it (empty session)
	session, _ := SessionStore.Get(c.Request, sessionName)
	err := session.Save(c.Request, w)
	assert.NoError(t, err)

	// Call the function
	GetCurrentUser(c)

	// Should return unauthorized since no token_id in session
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetCurrentUser_TokenIDButNoTokens(t *testing.T) {
	// Set up config
	testConfig := &config.Config{
		Env: "test",
		Auth: config.AuthConfig{
			Enabled: true,
			OIDC: config.OIDCConfig{
				SessionSecret: "test-secret-key-1234567890123456",
			},
		},
	}
	config.SetConfig(testConfig)

	// Use a simple in-memory cookie store for testing
	SessionStore = sessions.NewCookieStore([]byte("test-secret-key-1234567890123456"))

	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/user", nil)

	// Get session and set token_id, but don't store actual tokens
	session, _ := SessionStore.Get(c.Request, sessionName)
	session.Values["token_id"] = "nonexistent-token-id"
	err := session.Save(c.Request, w)
	assert.NoError(t, err)

	// Add session cookie to the request
	for _, cookie := range w.Result().Cookies() {
		c.Request.AddCookie(cookie)
	}

	// Create new recorder for the actual call
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = c.Request

	// Call the function
	GetCurrentUser(c2)

	// Should return unauthorized since tokens not found
	assert.Equal(t, http.StatusUnauthorized, w2.Code)
}

func TestFetchUserInfo_NoDiscoveryEndpoint(t *testing.T) {
	// Set up config with invalid issuer
	testConfig := &config.Config{
		Env: "test",
		Auth: config.AuthConfig{
			Enabled: true,
			OIDC: config.OIDCConfig{
				Issuer: "http://invalid-issuer.example.com",
			},
		},
	}

	// Call fetchUserInfo with invalid config - should fail to get discovery document
	_, err := fetchUserInfo("test-token", testConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch OIDC discovery document")
}

func TestFetchUserInfo_MockServer(t *testing.T) {
	// Create mock OIDC discovery server
	discoveryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid-configuration" {
			response := map[string]any{
				"issuer":            "http://test-issuer",
				"userinfo_endpoint": "http://test-issuer/userinfo",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}
	}))
	defer discoveryServer.Close()

	testConfig := &config.Config{
		Env: "test",
		Auth: config.AuthConfig{
			Enabled: true,
			OIDC: config.OIDCConfig{
				Issuer: discoveryServer.URL,
			},
		},
	}

	// This will fail at the userinfo fetch step (since userinfo endpoint doesn't exist)
	_, err := fetchUserInfo("test-token", testConfig)
	assert.Error(t, err)
}

func TestFetchUserInfo_NoUserInfoEndpoint(t *testing.T) {
	// Create mock OIDC discovery server without userinfo_endpoint
	discoveryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid-configuration" {
			response := map[string]any{
				"issuer": "http://test-issuer",
				// No userinfo_endpoint
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}
	}))
	defer discoveryServer.Close()

	testConfig := &config.Config{
		Env: "test",
		Auth: config.AuthConfig{
			Enabled: true,
			OIDC: config.OIDCConfig{
				Issuer: discoveryServer.URL,
			},
		},
	}

	_, err := fetchUserInfo("test-token", testConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no userinfo endpoint in discovery document")
}

func TestFetchUserInfo_FullMockServer(t *testing.T) {
	// Create fully functional mock OIDC server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			// Return discovery document with userinfo endpoint pointing to same server
			serverURL := "http://" + r.Host
			response := map[string]any{
				"issuer":            serverURL,
				"userinfo_endpoint": serverURL + "/userinfo",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		case "/userinfo":
			// Return user info
			response := map[string]any{
				"sub":            "test-user-123",
				"name":           "Test User",
				"email":          "test@example.com",
				"email_verified": true,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	testConfig := &config.Config{
		Env: "test",
		Auth: config.AuthConfig{
			Enabled: true,
			OIDC: config.OIDCConfig{
				Issuer: mockServer.URL,
			},
		},
	}

	userInfo, err := fetchUserInfo("test-token", testConfig)
	assert.NoError(t, err)
	assert.NotNil(t, userInfo)
	assert.Equal(t, "test-user-123", userInfo["sub"])
	assert.Equal(t, "Test User", userInfo["name"])
	assert.Equal(t, true, userInfo["authenticated"])
}

func TestFetchUserInfo_BadStatusCode(t *testing.T) {
	// Create mock server that returns an error status code from userinfo
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			serverURL := "http://" + r.Host
			response := map[string]any{
				"issuer":            serverURL,
				"userinfo_endpoint": serverURL + "/userinfo",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		case "/userinfo":
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error": "invalid_token"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	testConfig := &config.Config{
		Env: "test",
		Auth: config.AuthConfig{
			Enabled: true,
			OIDC: config.OIDCConfig{
				Issuer: mockServer.URL,
			},
		},
	}

	_, err := fetchUserInfo("test-token", testConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "userinfo endpoint returned status 401")
}

func TestGetCurrentUser_WithTokensButExpired(t *testing.T) {
	// Create a fully functional mock OIDC server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			serverURL := "http://" + r.Host
			response := map[string]any{
				"issuer":            serverURL,
				"userinfo_endpoint": serverURL + "/userinfo",
				"token_endpoint":    serverURL + "/token",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		case "/token":
			// Token refresh endpoint - return error
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error": "invalid_grant"}`))
		case "/userinfo":
			// User info endpoint
			response := map[string]any{
				"sub":  "test-user",
				"name": "Test User",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// Set up config
	testConfig := &config.Config{
		Env: "test",
		Auth: config.AuthConfig{
			Enabled: true,
			OIDC: config.OIDCConfig{
				Issuer:                mockServer.URL,
				ClientID:              "test-client",
				ClientSecret:          "test-secret",
				SessionSecret:         "test-secret-key-1234567890123456",
				TokenRefreshBufferSec: 60,
			},
		},
	}
	config.SetConfig(testConfig)

	// Set up session store
	SessionStore = sessions.NewCookieStore([]byte("test-secret-key-1234567890123456"))

	// Create and store tokens that are expired
	// storeTokens(accessToken, refreshToken, idToken string, expiresAt int64) returns tokenID
	tokenID := storeTokens("expired-token", "test-refresh-token", "", 1) // Expired long ago (Unix timestamp 1 = 1970-01-01)

	// Create test context with session
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/user", nil)

	// Get session and set token_id
	session, _ := SessionStore.Get(c.Request, sessionName)
	session.Values["token_id"] = tokenID
	err := session.Save(c.Request, w)
	assert.NoError(t, err)

	// Add session cookie to request
	for _, cookie := range w.Result().Cookies() {
		c.Request.AddCookie(cookie)
	}

	// Create new recorder for actual call
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = c.Request

	// Call GetCurrentUser
	GetCurrentUser(c2)

	// Should return unauthorized since token is expired and refresh failed
	assert.Equal(t, http.StatusUnauthorized, w2.Code)
}

func TestGetCurrentUser_WithValidTokens(t *testing.T) {
	// Create a fully functional mock OIDC server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			serverURL := "http://" + r.Host
			response := map[string]any{
				"issuer":            serverURL,
				"userinfo_endpoint": serverURL + "/userinfo",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		case "/userinfo":
			// Verify authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer valid-test-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			response := map[string]any{
				"sub":   "test-user-123",
				"name":  "Test User",
				"email": "test@example.com",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	// Set up config
	testConfig := &config.Config{
		Env: "test",
		Auth: config.AuthConfig{
			Enabled: true,
			OIDC: config.OIDCConfig{
				Issuer:                mockServer.URL,
				SessionSecret:         "test-secret-key-1234567890123456",
				TokenRefreshBufferSec: 60,
			},
		},
	}
	config.SetConfig(testConfig)

	// Set up session store
	SessionStore = sessions.NewCookieStore([]byte("test-secret-key-1234567890123456"))

	// Create and store valid tokens
	futureTime := time.Now().Add(1 * time.Hour).Unix()
	tokenID := storeTokens("valid-test-token", "", "", futureTime)

	// Create test context with session
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/user", nil)

	// Get session and set token_id
	session, _ := SessionStore.Get(c.Request, sessionName)
	session.Values["token_id"] = tokenID
	err := session.Save(c.Request, w)
	assert.NoError(t, err)

	// Add session cookie to request
	for _, cookie := range w.Result().Cookies() {
		c.Request.AddCookie(cookie)
	}

	// Create new recorder for actual call
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = c.Request

	// Call GetCurrentUser
	GetCurrentUser(c2)

	// Should return 200 OK with user info
	assert.Equal(t, http.StatusOK, w2.Code)

	var userInfo map[string]any
	err = json.Unmarshal(w2.Body.Bytes(), &userInfo)
	assert.NoError(t, err)
	assert.Equal(t, "test-user-123", userInfo["sub"])
	assert.Equal(t, true, userInfo["authenticated"])
}

// ============================================
// UserInfo Cache Tests
// ============================================

// clearUserInfoCache clears the userinfo cache between tests.
func clearUserInfoCache() {
	userInfoCacheMu.Lock()
	for k := range userInfoCache {
		delete(userInfoCache, k)
	}
	userInfoCacheMu.Unlock()
}

// newMockUserInfoServer creates a test HTTP server that serves both the OIDC
// discovery document and a userinfo endpoint. It returns the server and a
// pointer to an int that counts how many requests hit the userinfo endpoint.
func newMockUserInfoServer(userInfo map[string]any) (*httptest.Server, *int) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			// Return discovery doc pointing userinfo to ourselves
			scheme := "http"
			doc := map[string]any{
				"issuer":            scheme + "://" + r.Host,
				"userinfo_endpoint": scheme + "://" + r.Host + "/userinfo",
			}
			_ = json.NewEncoder(w).Encode(doc)
		case "/userinfo":
			callCount++
			_ = json.NewEncoder(w).Encode(userInfo)
		default:
			http.NotFound(w, r)
		}
	}))
	return server, &callCount
}

func TestFetchUserInfoCache_HitAndMiss(t *testing.T) {
	clearUserInfoCache()
	clearDiscoveryDocCache()

	server, callCount := newMockUserInfoServer(map[string]any{
		"sub":  "user-1",
		"name": "Test User",
	})
	defer server.Close()

	cfg := &config.Config{
		Auth: config.AuthConfig{
			OIDC: config.OIDCConfig{
				Issuer:               server.URL,
				UserInfoCacheTTL:     3600,
				DiscoveryDocCacheTTL: 3600,
			},
		},
	}
	config.SetConfig(cfg)

	// First call — cache miss, should reach the server
	info1, err := fetchUserInfo("token-abc", cfg)
	assert.NoError(t, err)
	assert.Equal(t, "user-1", info1["sub"])
	assert.Equal(t, true, info1["authenticated"])
	assert.Equal(t, 1, *callCount)

	// Second call — cache hit
	info2, err := fetchUserInfo("token-abc", cfg)
	assert.NoError(t, err)
	assert.Equal(t, "user-1", info2["sub"])
	assert.Equal(t, 1, *callCount) // still 1
}

func TestFetchUserInfoCache_TTLExpiry(t *testing.T) {
	clearUserInfoCache()
	clearDiscoveryDocCache()

	server, callCount := newMockUserInfoServer(map[string]any{
		"sub": "user-2",
	})
	defer server.Close()

	cfg := &config.Config{
		Auth: config.AuthConfig{
			OIDC: config.OIDCConfig{
				Issuer:               server.URL,
				UserInfoCacheTTL:     1, // 1 second TTL
				DiscoveryDocCacheTTL: 3600,
			},
		},
	}
	config.SetConfig(cfg)

	// First call
	_, err := fetchUserInfo("token-xyz", cfg)
	assert.NoError(t, err)
	assert.Equal(t, 1, *callCount)

	// Cached
	_, err = fetchUserInfo("token-xyz", cfg)
	assert.NoError(t, err)
	assert.Equal(t, 1, *callCount)

	// Wait for TTL to expire
	time.Sleep(1100 * time.Millisecond)

	// Should fetch again
	_, err = fetchUserInfo("token-xyz", cfg)
	assert.NoError(t, err)
	assert.Equal(t, 2, *callCount)
}

func TestInvalidateUserInfoCache(t *testing.T) {
	clearUserInfoCache()
	clearDiscoveryDocCache()

	server, callCount := newMockUserInfoServer(map[string]any{
		"sub": "user-3",
	})
	defer server.Close()

	cfg := &config.Config{
		Auth: config.AuthConfig{
			OIDC: config.OIDCConfig{
				Issuer:               server.URL,
				UserInfoCacheTTL:     3600,
				DiscoveryDocCacheTTL: 3600,
			},
		},
	}
	config.SetConfig(cfg)

	// Populate cache
	_, err := fetchUserInfo("token-inv", cfg)
	assert.NoError(t, err)
	assert.Equal(t, 1, *callCount)

	// Invalidate
	invalidateUserInfoCache("token-inv", server.URL)

	// Next call should be a cache miss
	_, err = fetchUserInfo("token-inv", cfg)
	assert.NoError(t, err)
	assert.Equal(t, 2, *callCount)
}

func TestStartUserInfoCacheCleanup(t *testing.T) {
	clearUserInfoCache()

	// Manually insert an already-expired entry
	h := make([]byte, 32)
	cacheKey := hex.EncodeToString(h)
	userInfoCacheMu.Lock()
	userInfoCache[cacheKey] = userInfoCacheEntry{
		data:      map[string]any{"sub": "stale"},
		expiresAt: time.Now().Add(-1 * time.Minute), // already expired
	}
	userInfoCacheMu.Unlock()

	// Verify it's there
	userInfoCacheMu.RLock()
	assert.Equal(t, 1, len(userInfoCache))
	userInfoCacheMu.RUnlock()

	// Start cleanup with a very short interval
	startUserInfoCacheCleanup(200 * time.Millisecond)

	// Wait for at least one sweep
	time.Sleep(500 * time.Millisecond)

	// The expired entry should have been removed
	userInfoCacheMu.RLock()
	assert.Equal(t, 0, len(userInfoCache))
	userInfoCacheMu.RUnlock()
}
