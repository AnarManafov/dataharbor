package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/AnarManafov/dataharbor/app/common"
	"github.com/AnarManafov/dataharbor/app/config"
	"github.com/AnarManafov/dataharbor/app/response"
)

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

// fetchUserInfo gets the user profile from Keycloak's userinfo endpoint
func fetchUserInfo(accessToken string, cfg *config.Config) (map[string]interface{}, error) {
	logger := common.GetLogger()

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
	defer userInfoResp.Body.Close()

	if userInfoResp.StatusCode != http.StatusOK {
		body, _ := json.Marshal(userInfoResp.Body)
		return nil, fmt.Errorf("userinfo endpoint returned status %d: %s", userInfoResp.StatusCode, string(body))
	}

	var userInfo map[string]interface{}
	if err := json.NewDecoder(userInfoResp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse userinfo response: %w", err)
	}

	// Add authentication flag needed by frontend to determine login state
	userInfo["authenticated"] = true

	// Debug information to help troubleshoot authentication issues
	logger.Infof("User info fetched successfully: %+v", userInfo)

	return userInfo, nil
}
