package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/AnarManafov/data_lake_ui/app/common"
	"github.com/AnarManafov/data_lake_ui/app/config"
	"github.com/AnarManafov/data_lake_ui/app/response"
)

// GetCurrentUser returns information about the currently authenticated user.
// Used by the frontend to determine if a user is logged in and get their profile.
func GetCurrentUser(c *gin.Context) {
	logger := common.GetLogger()
	cfg := config.GetConfig()

	// Debug information to track authentication issues
	logger.Infof("GetCurrentUser called - Headers: %v", c.Request.Header)
	logger.Infof("GetCurrentUser called - Cookies: %v", c.Request.Cookies())

	session, err := SessionStore.Get(c.Request, sessionName)
	if err != nil {
		logger.Error("Failed to get session", "error", err)

		// CORS headers needed for cross-domain authentication flows
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Origin", c.Request.Header.Get("Origin"))

		response.Error(c, http.StatusUnauthorized, "Invalid session")
		return
	}

	accessToken, ok := session.Values["access_token"].(string)
	if !ok || accessToken == "" {
		logger.Info("No access token in session")

		// CORS headers needed for cross-domain authentication flows
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Origin", c.Request.Header.Get("Origin"))

		response.Error(c, http.StatusUnauthorized, "Not authenticated")
		return
	}

	userInfo, err := fetchUserInfo(accessToken, cfg)
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
