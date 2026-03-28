package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/AnarManafov/dataharbor/app/common"
	"github.com/AnarManafov/dataharbor/app/config"
	"github.com/AnarManafov/dataharbor/app/response"
)

// userInfoCacheEntry stores cached user info with an expiration time.
type userInfoCacheEntry struct {
	data      map[string]any
	expiresAt time.Time
}

var (
	userInfoCache   = make(map[string]userInfoCacheEntry)
	userInfoCacheMu sync.RWMutex
)

// invalidateUserInfoCache removes the cached userinfo for a specific access token.
// Called on logout so stale data is not served if the same token somehow reappears.
func invalidateUserInfoCache(accessToken, issuer string) {
	h := sha256.Sum256([]byte(accessToken + "|" + issuer))
	cacheKey := hex.EncodeToString(h[:])
	userInfoCacheMu.Lock()
	delete(userInfoCache, cacheKey)
	userInfoCacheMu.Unlock()
}

// startUserInfoCacheCleanup launches a background goroutine that periodically removes
// expired entries from the userinfo cache. Without this, entries from refreshed/expired
// tokens accumulate indefinitely while the service runs 24/7.
func startUserInfoCacheCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			userInfoCacheMu.Lock()
			for key, entry := range userInfoCache {
				if now.After(entry.expiresAt) {
					delete(userInfoCache, key)
				}
			}
			userInfoCacheMu.Unlock()
		}
	}()
}

// GetCurrentUser returns information about the currently authenticated user.
// Used by the frontend to determine if a user is logged in and get their profile.
func GetCurrentUser(c *gin.Context) {
	logger := common.GetLogger()
	cfg := config.GetConfig()

	// Debug information to track authentication issues
	logger.Infof("GetCurrentUser called - Headers: %v", c.Request.Header)
	logger.Infof("GetCurrentUser called - Cookies: %v", c.Request.Cookies())

	// Get session
	session, err := SessionStore.Get(c.Request, sessionName)
	if err != nil {
		logger.Error("Failed to get session", "error", err)

		// CORS headers needed for cross-domain authentication flows
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Origin", c.Request.Header.Get("Origin"))

		response.Error(c, http.StatusUnauthorized, "Invalid session")
		return
	}

	// Get token ID from session
	tokenID, ok := session.Values["token_id"].(string)
	if !ok || tokenID == "" {
		logger.Info("No token ID in session")

		// CORS headers needed for cross-domain authentication flows
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Origin", c.Request.Header.Get("Origin"))

		response.Error(c, http.StatusUnauthorized, "Not authenticated")
		return
	}

	// Get tokens from token store
	tokens, ok := getTokens(tokenID)
	if !ok {
		logger.Info("No tokens found for token ID")

		// CORS headers needed for cross-domain authentication flows
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Origin", c.Request.Header.Get("Origin"))

		response.Error(c, http.StatusUnauthorized, "Not authenticated")
		return
	}

	// Log whether we have a refresh token
	logger.Infof("Refresh token available: %v", tokens.RefreshToken != "")

	// Check token expiration and refresh if needed
	currentTime := time.Now().Unix()
	if tokens.ExpiresAt > 0 {
		timeRemaining := tokens.ExpiresAt - currentTime

		// Log token expiration details
		logger.Infof("Token expires at: %s (Unix: %d), Current time: %s (Unix: %d), Time remaining: %d seconds",
			time.Unix(tokens.ExpiresAt, 0).Format(time.RFC3339),
			tokens.ExpiresAt,
			time.Unix(currentTime, 0).Format(time.RFC3339),
			currentTime,
			timeRemaining)

		// If token is expired or will expire soon (configurable buffer)
		// This prevents tokens from expiring during ongoing operations
		// For short-lived tokens (e.g., 5 minutes), a 1-minute buffer is appropriate
		cfg := config.GetConfig()
		refreshBuffer := cfg.Auth.OIDC.TokenRefreshBufferSec
		if refreshBuffer <= 0 {
			refreshBuffer = 60 // Fallback to 1 minute if not configured
		}

		if currentTime > tokens.ExpiresAt || timeRemaining < refreshBuffer {
			logger.Info("Token expired or expiring soon, attempting refresh")
			if err := refreshToken(c); err != nil {
				logger.Error("Failed to refresh token", "error", err)

				// If token is actually expired, return unauthorized
				if currentTime > tokens.ExpiresAt {
					c.Header("Access-Control-Allow-Credentials", "true")
					c.Header("Access-Control-Allow-Origin", c.Request.Header.Get("Origin"))
					response.Error(c, http.StatusUnauthorized, "Session expired")
					return
				}
				// Otherwise continue with existing token
				logger.Warn("Using existing token despite failed refresh attempt")
			} else {
				// Get the updated tokens after refresh
				tokens, _ = getTokens(tokenID)
				logger.Info("Successfully refreshed token before user info fetch")
			}
		}
	} else {
		logger.Warn("No token expiration information available")
	}

	userInfo, err := fetchUserInfo(tokens.AccessToken, cfg)
	if err != nil {
		logger.Error("Failed to fetch user info", "error", err)

		// CORS headers needed for cross-domain authentication flows
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Origin", c.Request.Header.Get("Origin"))

		response.Error(c, http.StatusInternalServerError, "Failed to fetch user information")
		return
	}

	// CORS headers needed for cross-domain authentication flows
	c.Header("Access-Control-Allow-Credentials", "true")
	c.Header("Access-Control-Allow-Origin", c.Request.Header.Get("Origin"))

	c.JSON(http.StatusOK, userInfo)
}

// fetchUserInfo gets the user profile from Keycloak's userinfo endpoint.
// Results are cached per access token with a configurable TTL (default 60s)
// to avoid redundant HTTP calls on every authenticated request.
func fetchUserInfo(accessToken string, cfg *config.Config) (map[string]any, error) {
	logger := common.GetLogger()

	// Cache key: SHA-256 hash of the access token + issuer to differentiate per-IdP
	h := sha256.Sum256([]byte(accessToken + "|" + cfg.Auth.OIDC.Issuer))
	cacheKey := hex.EncodeToString(h[:])

	// Check cache
	userInfoCacheMu.RLock()
	if entry, ok := userInfoCache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		userInfoCacheMu.RUnlock()
		return entry.data, nil
	}
	userInfoCacheMu.RUnlock()

	// Get discovery document using centralized helper function
	discoveryDoc, err := fetchOIDCDiscoveryDocument(cfg.Auth.OIDC.Issuer)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OIDC discovery document: %w", err)
	}

	// Extract userinfo endpoint from discovery document - this is the standard OIDC way
	// to find the correct endpoint rather than hardcoding it
	userinfoEndpoint, ok := discoveryDoc["userinfo_endpoint"].(string)
	if !ok || userinfoEndpoint == "" {
		return nil, fmt.Errorf("no userinfo endpoint in discovery document")
	}

	logger.Infof("Using userinfo endpoint: %s", userinfoEndpoint)

	req, err := http.NewRequest("GET", userinfoEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create userinfo request: %w", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	client := &http.Client{Timeout: 10 * time.Second}
	userInfoResp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch userinfo: %w", err)
	}
	defer func() { _ = userInfoResp.Body.Close() }()

	if userInfoResp.StatusCode != http.StatusOK {
		body, _ := json.Marshal(userInfoResp.Body)
		return nil, fmt.Errorf("userinfo endpoint returned status %d: %s", userInfoResp.StatusCode, string(body))
	}

	var userInfo map[string]any
	if err := json.NewDecoder(userInfoResp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse userinfo response: %w", err)
	}

	// Add authentication flag needed by frontend to determine login state
	userInfo["authenticated"] = true

	// Debug information to help troubleshoot authentication issues
	logger.Infof("User info fetched successfully: %+v", userInfo)

	// Cache the result
	ttl := cfg.Auth.OIDC.UserInfoCacheTTL
	if ttl <= 0 {
		ttl = 60
	}
	userInfoCacheMu.Lock()
	userInfoCache[cacheKey] = userInfoCacheEntry{
		data:      userInfo,
		expiresAt: time.Now().Add(time.Duration(ttl) * time.Second),
	}
	userInfoCacheMu.Unlock()

	return userInfo, nil
}
