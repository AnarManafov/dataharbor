package controller

// This file contains authentication-related controllers and middleware.
// Note: The authentication middleware (SessionAuthMiddleware) is implemented here
// rather than in middleware/auth_middleware.go to keep all authentication-related
// logic in one place.

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"

	"github.com/AnarManafov/data_lake_ui/app/common"
	"github.com/AnarManafov/data_lake_ui/app/config"
	"github.com/AnarManafov/data_lake_ui/app/response"
)

const (
	sessionName   = "data-lake-ui-session"
	sessionMaxAge = 86400 * 7 // 7 days - chosen to balance user convenience with security risks
)

// SessionStore holds the session information
// Will be initialized in init()
var SessionStore *sessions.CookieStore

// TokenStore stores tokens in memory with unique IDs to avoid cookie size limitations
// In a production environment with multiple instances, this should be replaced with a distributed store
//
// IMPORTANT: We use an in-memory token store because:
//  1. Cookie size limits (~4KB) prevent storing large tokens directly in cookies
//     (our tokens are ~5KB+ in size)
//  2. Browser localStorage/sessionStorage is not accessible to the backend
//  3. Storing tokens server-side provides better security as tokens never leave the server
//     except when making authorized API calls
//
// For high-availability deployments:
// - This in-memory implementation will not work properly with multiple server instances
// - Should be replaced with a distributed store like Redis or a database
// - Each request might be routed to a different instance that doesn't have the tokens
//
// Token cleanup considerations:
// - This implementation has no automatic cleanup mechanism for abandoned tokens
// - In a production setting, implement a token cleanup routine to prevent memory leaks
type TokenInfo struct {
	AccessToken  string
	RefreshToken string
	IDToken      string
	ExpiresAt    int64
}

var tokenStore = make(map[string]TokenInfo)

// generateTokenID creates a unique ID for storing tokens
func generateTokenID() string {
	return uuid.New().String()
}

// storeTokens saves tokens in the token store and returns an ID
func storeTokens(accessToken, refreshToken, idToken string, expiresAt int64) string {
	tokenID := generateTokenID()
	tokenStore[tokenID] = TokenInfo{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		IDToken:      idToken,
		ExpiresAt:    expiresAt,
	}
	return tokenID
}

// getTokens retrieves tokens from the store
func getTokens(tokenID string) (TokenInfo, bool) {
	tokens, ok := tokenStore[tokenID]
	return tokens, ok
}

// updateTokens updates tokens in the store
func updateTokens(tokenID string, accessToken, refreshToken, idToken string, expiresAt int64) bool {
	if _, exists := tokenStore[tokenID]; !exists {
		return false
	}

	tokenStore[tokenID] = TokenInfo{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		IDToken:      idToken,
		ExpiresAt:    expiresAt,
	}
	return true
}

// deleteTokens removes tokens from the store
func deleteTokens(tokenID string) {
	delete(tokenStore, tokenID)
}

func init() {
	cfg := config.GetConfig()
	logger := common.GetLogger()

	// Using session secret from config enables persistent sessions across restarts
	// Random generation is a fallback that prioritizes security over user convenience
	sessionSecret := cfg.Auth.OIDC.SessionSecret
	if sessionSecret == "" {
		// Generate a random secret
		b := make([]byte, 32)
		_, err := rand.Read(b)
		if err != nil {
			logger.Error("Failed to generate random session secret", "error", err)
			// Fallback to a default (but this is not secure for production)
			sessionSecret = "default-session-secret-replace-in-production"
		} else {
			sessionSecret = base64.StdEncoding.EncodeToString(b)
			logger.Warn("Generated random session secret. Sessions will be invalidated on server restart. Set auth.oidc.session_secret in your config for persistence.")
		}
	}

	// Initialize the session store with the secret
	SessionStore = sessions.NewCookieStore([]byte(sessionSecret))

	// TODO: Production security settings:
	// 1. Use HTTPS for both frontend and backend
	// 2. Set Secure: true for cookies
	// 3. Consider a more restrictive SameSite policy if your setup allows it (e.g., SameSiteLaxMode)

	// In development mode, we use more relaxed settings for ease of testing
	sameSiteMode := http.SameSiteStrictMode
	secureCookies := true

	// Development mode requires relaxed security for ease of testing
	if cfg.Env == "development" {
		// In development, we allow cookies to be shared between different ports on localhost
		sameSiteMode = http.SameSiteLaxMode
		secureCookies = false
		logger.Info("Running in development mode: using relaxed cookie settings")
	}

	// Configure session for security best practices
	SessionStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   sessionMaxAge,
		HttpOnly: true,          // Prevents JavaScript access, mitigating XSS risks
		Secure:   secureCookies, // Only use secure cookies in production
		SameSite: sameSiteMode,  // Allow in-site requests (Lax mode for dev, Strict for prod)
	}
}

// TokenResponse represents the token response from the OIDC provider
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// LoginInit initiates the OIDC login flow
// Returns the authorization URL that the frontend will redirect to
func LoginInit(c *gin.Context) {
	cfg := config.GetConfig()
	logger := common.GetLogger()

	// Enhanced debugging - Log all request details to diagnose the issue
	logger.Infof("LOGIN REQUEST - Headers: %v", c.Request.Header)
	logger.Infof("LOGIN REQUEST - Query params: %v", c.Request.URL.Query())
	logger.Infof("LOGIN REQUEST - Remote addr: %v", c.Request.RemoteAddr)

	// Very detailed auth config logging
	logger.Infof("Auth configuration - Enabled: %v", cfg.Auth.Enabled)
	logger.Infof("Auth configuration - OIDC Issuer: %v", cfg.Auth.OIDC.Issuer)
	logger.Infof("Auth configuration - OIDC ClientID: %v", cfg.Auth.OIDC.ClientID)
	logger.Infof("Auth configuration - OIDC ClientSecret set: %v", cfg.Auth.OIDC.ClientSecret != "")
	logger.Infof("Auth configuration - Session Secret set: %v", cfg.Auth.OIDC.SessionSecret != "")

	// Partial logging of secret prevents full exposure while allowing debug confirmation
	secretForLog := "not-set"
	if cfg.Auth.OIDC.ClientSecret != "" {
		if len(cfg.Auth.OIDC.ClientSecret) >= 4 {
			secretForLog = cfg.Auth.OIDC.ClientSecret[:4] + "..."
		} else {
			secretForLog = "[too short]"
		}
	}

	// Log the OIDC configuration for debugging
	logger.Infof("OIDC Config - Issuer: %s, ClientID: %s, ClientSecret: %s",
		cfg.Auth.OIDC.Issuer,
		cfg.Auth.OIDC.ClientID,
		secretForLog)

	// Check for required OIDC fields - log any issues found
	if cfg.Auth.Enabled {
		if cfg.Auth.OIDC.Issuer == "" {
			logger.Error("Auth is enabled but OIDC Issuer is empty - this will cause auth to be disabled")
		}
		if cfg.Auth.OIDC.ClientID == "" {
			logger.Error("Auth is enabled but OIDC ClientID is empty - this will cause auth to be disabled")
		}
		if cfg.Auth.OIDC.ClientSecret == "" {
			logger.Error("Auth is enabled but OIDC ClientSecret is empty - this will cause auth to be disabled")
		}
	}

	// Support disabled authentication for development/testing environments
	if !cfg.Auth.Enabled {
		logger.Warn("Authentication is disabled in configuration, but login endpoint was called")
		c.JSON(http.StatusOK, gin.H{
			"auth_url": "/auth/disabled",
			"message":  "Authentication is disabled",
		})
		return
	}

	// Fail-fast if misconfigured to prevent runtime errors
	if cfg.Auth.Enabled && (cfg.Auth.OIDC.Issuer == "" || cfg.Auth.OIDC.ClientID == "" || cfg.Auth.OIDC.ClientSecret == "") {
		logger.Error("Authentication is enabled but OIDC configuration is incomplete")
		logger.Error("Issuer URL: ", cfg.Auth.OIDC.Issuer)
		logger.Error("ClientID: ", cfg.Auth.OIDC.ClientID)
		logger.Error("ClientSecret is set: ", cfg.Auth.OIDC.ClientSecret != "")

		// Return "auth disabled" to frontend when config is incomplete
		c.JSON(http.StatusOK, gin.H{
			"auth_url": "/auth/disabled",
			"message":  "Authentication is disabled due to incomplete OIDC configuration",
		})
		return
	}

	// UUID ensures state is unpredictable, preventing CSRF attacks
	state := uuid.New().String()

	// For development, add a debug cookie to verify cookie behavior
	if cfg.Env == "development" {
		c.SetCookie("debug-cookie", "test-value", 3600, "/", "", false, false)
	}

	// Create a new session for this login attempt
	session, err := SessionStore.Get(c.Request, sessionName)
	if err != nil {
		logger.Warn("Failed to get existing session, creating new one: %s", err.Error())
		// This is fine, we'll create a new session
	}

	// Store the state in the session
	session.Values["oidc_state"] = state
	logger.Infof("Setting state in session: %s", state)

	// Set session options directly to ensure proper cross-origin behavior
	session.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   sessionMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
		Secure:   false, // Set to false for development, should be true in production
	}

	// Save the session
	if err := session.Save(c.Request, c.Writer); err != nil {
		logger.Error("Failed to save session", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to initialize login flow: " + err.Error(),
		})
		return
	}

	// Log the cookie header that was set
	logger.Infof("Set-Cookie header: %v", c.Writer.Header().Get("Set-Cookie"))

	// IMPORTANT FIX: Always use absolute URLs for redirect_uri
	// This is critical for OAuth/OIDC providers which require exact match of redirect URIs
	redirectURI := fmt.Sprintf("%s://%s/api/auth/callback", schemeFromRequest(c), c.Request.Host)

	// Log the redirect URI being used
	logger.Infof("Using redirect URI: %s", redirectURI)

	// Ensure issuer URL is properly formatted to prevent connection errors
	issuerURL := cfg.Auth.OIDC.Issuer

	// Try to load discovery document to ensure issuer is reachable
	discoveryDoc, err := fetchOIDCDiscoveryDocument(issuerURL)
	if err != nil {
		logger.Error("Failed to load OIDC discovery document", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to initialize OIDC provider: " + err.Error(),
		})
		return
	}

	// Use authorization_endpoint from discovery document
	authEndpoint, ok := discoveryDoc["authorization_endpoint"].(string)
	if !ok || authEndpoint == "" {
		logger.Error("No authorization endpoint in discovery document")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to initialize OIDC provider: no authorization endpoint found",
		})
		return
	}

	logger.Infof("Using authorization endpoint: %s", authEndpoint)

	// Use Keycloak-specific paths for the authorization endpoint
	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&state=%s&scope=openid%%20profile%%20email",
		authEndpoint,
		url.QueryEscape(cfg.Auth.OIDC.ClientID),
		url.QueryEscape(redirectURI),
		state)

	logger.Infof("Generated auth URL: %s", authURL)

	// Return authorization URL to frontend for redirect
	c.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
	})
}

// AuthCallback handles the OIDC callback with the authorization code
// This is where the actual token exchange happens after user authenticates
func AuthCallback(c *gin.Context) {
	cfg := config.GetConfig()
	logger := common.GetLogger()

	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		response.Error(c, http.StatusBadRequest, "Authorization code is missing")
		return
	}

	logger.Infof("Auth callback received with code and state: %s", state)
	logger.Infof("Cookies: %v", c.Request.Cookies())

	// State verification is critical to prevent CSRF attacks in the OAuth flow
	session, err := SessionStore.Get(c.Request, sessionName)
	if err != nil {
		logger.Error("Failed to get session", "error", err)
		// In development, we'll skip state validation if we can't get the session
		// This helps when testing cross-domain authentication
		if cfg.Env == "development" {
			logger.Warn("Development mode: Skipping state validation due to session retrieval failure")
		} else {
			response.Error(c, http.StatusInternalServerError, "Failed to validate session")
			return
		}
	}

	var sessionState string
	if v, ok := session.Values["oidc_state"].(string); ok && v != "" {
		sessionState = v
		logger.Infof("Found state in session: %s", sessionState)
	} else {
		logger.Error("No state in session")
		// In development, we'll continue even without state validation
		if cfg.Env == "development" {
			logger.Warn("Development mode: Proceeding without state validation")
		} else {
			response.Error(c, http.StatusBadRequest, "Invalid session state")
			return
		}
	}

	// Only validate state if we have one from the session
	if sessionState != "" && sessionState != state {
		logger.Error("State mismatch", "session", sessionState, "callback", state)
		// In development, we'll continue even with state mismatch
		if cfg.Env == "development" {
			logger.Warn("Development mode: Ignoring state mismatch")
		} else {
			response.Error(c, http.StatusBadRequest, "Invalid state parameter")
			return
		}
	}

	// Clear state after use to prevent replay attacks
	if sessionState != "" {
		delete(session.Values, "oidc_state")
	}

	// The redirect_uri must exactly match what was used in the initial request
	redirectURI := fmt.Sprintf("%s://%s/api/auth/callback", schemeFromRequest(c), c.Request.Host)

	// Try to load discovery document to get token endpoint
	issuerURL := cfg.Auth.OIDC.Issuer

	discoveryDoc, err := fetchOIDCDiscoveryDocument(issuerURL)
	if err != nil {
		logger.Error("Failed to load discovery document", "error", err)
		response.Error(c, http.StatusInternalServerError, "Failed to complete authentication")
		return
	}

	// Get token_endpoint from discovery document
	tokenEndpoint, ok := discoveryDoc["token_endpoint"].(string)
	if !ok || tokenEndpoint == "" {
		logger.Error("No token endpoint in discovery document")
		response.Error(c, http.StatusInternalServerError, "Failed to complete authentication")
		return
	}

	logger.Infof("Using token endpoint: %s", tokenEndpoint)

	// Exchange code for tokens using the token endpoint
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", cfg.Auth.OIDC.ClientID)
	data.Set("client_secret", cfg.Auth.OIDC.ClientSecret)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequest("POST", tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		logger.Error("Failed to create token request", "error", err)
		response.Error(c, http.StatusInternalServerError, "Failed to process authentication")
		return
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	// 10 second timeout prevents hanging requests during network issues
	client := &http.Client{Timeout: 10 * time.Second}
	tokenResp, err := client.Do(req)
	if err != nil {
		logger.Error("Failed to exchange code for tokens", "error", err)
		response.Error(c, http.StatusInternalServerError, "Failed to process authentication")
		return
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(tokenResp.Body)
		logger.Error("Token endpoint returned error", "status", tokenResp.StatusCode, "body", string(body))
		response.Error(c, http.StatusInternalServerError, "Failed to authenticate with provider")
		return
	}

	var tokenResponse TokenResponse
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenResponse); err != nil {
		logger.Error("Failed to parse token response", "error", err)
		response.Error(c, http.StatusInternalServerError, "Failed to process authentication")
		return
	}

	// Log token response details (excluding sensitive parts)
	logger.Infof("Token response received: access_token present: %v, refresh_token present: %v, id_token present: %v, expires_in: %d",
		tokenResponse.AccessToken != "",
		tokenResponse.RefreshToken != "",
		tokenResponse.IDToken != "",
		tokenResponse.ExpiresIn)

	// Store tokens in memory and save the token ID in the session
	expiresAt := time.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second).Unix()
	tokenID := storeTokens(tokenResponse.AccessToken, tokenResponse.RefreshToken, tokenResponse.IDToken, expiresAt)
	session.Values["token_id"] = tokenID

	if err := session.Save(c.Request, c.Writer); err != nil {
		logger.Error("Failed to save session with token ID", "error", err)
		response.Error(c, http.StatusInternalServerError, "Failed to complete authentication")
		return
	}

	// Determine the frontend URL to redirect to after authentication
	frontendURL := c.Query("redirect")
	if frontendURL == "" {
		// Use the frontend URL from configuration or fall back to a default
		frontendURL = cfg.Frontend.URL
		if frontendURL == "" {
			// In development, we're likely using Vite at port 5173
			if cfg.Env == "development" {
				frontendURL = "http://localhost:5173"
			} else {
				// In production, default to same host but assume it's being served from root
				frontendURL = fmt.Sprintf("%s://%s", schemeFromRequest(c), c.Request.Host)
			}
		}
	}

	// If a specific path was requested, append it
	redirectPath := c.Query("redirect_path")
	if redirectPath != "" && redirectPath != "/" {
		frontendURL = frontendURL + redirectPath
	} else {
		frontendURL = frontendURL + "/"
	}

	logger.Infof("Redirecting authenticated user to frontend: %s", frontendURL)

	// Redirect user to the frontend application
	c.Redirect(http.StatusTemporaryRedirect, frontendURL)
}

// Logout handles user logout by invalidating the session
func Logout(c *gin.Context) {
	logger := common.GetLogger()

	// Get session
	session, err := SessionStore.Get(c.Request, sessionName)
	if err == nil {
		// Get token ID to delete from store
		if tokenID, ok := session.Values["token_id"].(string); ok && tokenID != "" {
			// Delete tokens from the store for security
			deleteTokens(tokenID)
			logger.Info("Removed tokens from store during logout")
		}
	}

	// Clear session values
	session.Values = make(map[interface{}]interface{})
	session.Options.MaxAge = -1 // This will delete the cookie

	if err := session.Save(c.Request, c.Writer); err != nil {
		logger.Error("Failed to clear session", "error", err)
		response.Error(c, http.StatusInternalServerError, "Failed to complete logout")
		return
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Logout successful"})
}

// Health returns the health status of the service
// Used for monitoring and readiness checks
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"data":    "ok",
		"code":    200,
		"message": "success",
	})
}

// refreshToken refreshes the access token using the refresh token
func refreshToken(c *gin.Context) error {
	cfg := config.GetConfig()
	logger := common.GetLogger()

	// Get session
	session, err := SessionStore.Get(c.Request, sessionName)
	if err != nil {
		logger.Error("Failed to get session during token refresh", "error", err)
		return fmt.Errorf("failed to get session: %w", err)
	}

	tokenID, ok := session.Values["token_id"].(string)
	if !ok || tokenID == "" {
		logger.Error("No token ID available in session")
		return fmt.Errorf("no token ID available")
	}

	tokens, ok := getTokens(tokenID)
	if !ok {
		logger.Error("No tokens found for token ID")
		return fmt.Errorf("no tokens found for token ID")
	}

	if tokens.RefreshToken == "" {
		logger.Error("No refresh token available for token ID")
		return fmt.Errorf("no refresh token available")
	}

	// Try to load discovery document to get token endpoint
	issuerURL := cfg.Auth.OIDC.Issuer

	discoveryDoc, err := fetchOIDCDiscoveryDocument(issuerURL)
	if err != nil {
		logger.Error("Failed to load discovery document during token refresh", "error", err)
		return fmt.Errorf("failed to load discovery document: %w", err)
	}

	// Get token_endpoint from discovery document
	tokenEndpoint, ok := discoveryDoc["token_endpoint"].(string)
	if !ok || tokenEndpoint == "" {
		logger.Error("No token endpoint in discovery document during refresh")
		return fmt.Errorf("no token endpoint in discovery document")
	}

	logger.Infof("Using token endpoint for refresh: %s", tokenEndpoint)

	// Exchange the refresh token for a new access token
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", tokens.RefreshToken)
	data.Set("client_id", cfg.Auth.OIDC.ClientID)
	data.Set("client_secret", cfg.Auth.OIDC.ClientSecret)

	req, err := http.NewRequest("POST", tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		logger.Error("Failed to create refresh token request", "error", err)
		return fmt.Errorf("failed to create refresh token request: %w", err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 10 * time.Second}
	refreshResp, err := client.Do(req)
	if err != nil {
		logger.Error("Failed to execute refresh token request", "error", err)
		return fmt.Errorf("failed to refresh token: %w", err)
	}
	defer refreshResp.Body.Close()

	if refreshResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(refreshResp.Body)
		logger.Error("Token endpoint returned error during refresh",
			"status", refreshResp.StatusCode,
			"body", string(body))
		return fmt.Errorf("token refresh failed with status %d: %s", refreshResp.StatusCode, string(body))
	}

	// Parse the token response
	var tokenResponse TokenResponse
	if err := json.NewDecoder(refreshResp.Body).Decode(&tokenResponse); err != nil {
		logger.Error("Failed to parse refresh token response", "error", err)
		return fmt.Errorf("failed to parse refresh token response: %w", err)
	}

	// Verify we received a valid access token
	if tokenResponse.AccessToken == "" {
		logger.Error("Refresh token response did not include access token")
		return fmt.Errorf("refresh token response missing access token")
	}

	// Update tokens in the store
	expiresAt := time.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second).Unix()

	// Keep the existing refresh token if the provider didn't send a new one
	refreshTokenValue := tokenResponse.RefreshToken
	if refreshTokenValue == "" {
		refreshTokenValue = tokens.RefreshToken
	}

	// Keep the existing ID token if the provider didn't send a new one
	idTokenValue := tokenResponse.IDToken
	if idTokenValue == "" {
		idTokenValue = tokens.IDToken
	}

	if !updateTokens(tokenID, tokenResponse.AccessToken, refreshTokenValue, idTokenValue, expiresAt) {
		logger.Error("Failed to update tokens in store")
		return fmt.Errorf("failed to update tokens in store")
	}

	logger.Info("Token refreshed successfully")
	return nil
}

// SessionAuthMiddleware creates a middleware that validates session-based authentication
// and handles token refresh when needed. This approach centralizes auth management
// in the controller package rather than the middleware package.
func SessionAuthMiddleware() gin.HandlerFunc {
	cfg := config.GetConfig()
	logger := common.GetLogger()

	// If authentication is globally disabled, create a pass-through middleware
	// Useful for development or internal deployments with alternate security measures
	if !cfg.Auth.Enabled {
		logger.Info("Authentication is disabled")
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// skipAuth determines which paths bypass authentication requirements
	// Critical for public endpoints like login, health checks, and static assets
	skipAuth := func(path string) bool {
		for _, skipPath := range cfg.Auth.SkipAuthPaths {
			if strings.HasPrefix(path, skipPath) {
				return true
			}
		}
		return false
	}

	logger.Info("Initializing session-based authentication middleware")

	return func(c *gin.Context) {
		// Allow certain paths to bypass authentication checks based on configuration
		if skipAuth(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Validate the user's session contains required authentication data
		session, err := SessionStore.Get(c.Request, sessionName)
		if err != nil {
			logger.Error("Failed to get session", "error", err)
			response.Error(c, http.StatusUnauthorized, "Invalid session")
			c.Abort()
			return
		}

		// Verify token ID exists - the presence of a token ID indicates previous successful auth
		tokenID, ok := session.Values["token_id"].(string)
		if !ok || tokenID == "" {
			logger.Info("No token ID in session")
			response.Error(c, http.StatusUnauthorized, "Not authenticated")
			c.Abort()
			return
		}

		tokens, ok := getTokens(tokenID)
		if !ok {
			logger.Info("No tokens found for token ID")
			response.Error(c, http.StatusUnauthorized, "Not authenticated")
			c.Abort()
			return
		}

		// Automatically refresh tokens to maintain user sessions
		// Proactively refresh tokens that are about to expire (within 5 minutes)
		// This prevents disruption of user experience due to token expiration
		refreshBuffer := int64(300) // 5 minutes in seconds
		currentTime := time.Now().Unix()

		// If token is expired or will expire within buffer period
		if currentTime > tokens.ExpiresAt || (tokens.ExpiresAt-currentTime) < refreshBuffer {
			logger.Info("Token expired or expiring soon, refreshing")
			if err := refreshToken(c); err != nil {
				logger.Error("Failed to refresh token", "error", err)

				// If token is already expired, return unauthorized
				if currentTime > tokens.ExpiresAt {
					response.Error(c, http.StatusUnauthorized, "Session expired")
					c.Abort()
					return
				}
				// Otherwise, continue with existing token even if refresh failed
				// This gives user a chance to complete their current action
				logger.Warn("Using existing token despite failed refresh attempt")
			} else {
				// Get the new tokens after successful refresh
				tokens, _ = getTokens(tokenID)
				logger.Info("Successfully refreshed token before expiration")
			}
		}

		// Pass the access token to downstream handlers via request context
		// This allows them to use the token for authorization
		c.Set("access_token", tokens.AccessToken)
		c.Set("user_claims", map[string]interface{}{
			"authenticated": true,
		})

		// Pass the access token to downstream services via Authorization header
		// This allows microservices behind this app to validate the token independently
		c.Request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokens.AccessToken))

		c.Next()
	}
}

// Helper functions

// fetchOIDCDiscoveryDocument gets the discovery document from the OIDC provider
// This centralizes the common pattern used in multiple auth functions
func fetchOIDCDiscoveryDocument(issuerURL string) (map[string]interface{}, error) {
	logger := common.GetLogger()

	// Ensure issuer URL has proper format
	if !strings.HasPrefix(issuerURL, "http://") && !strings.HasPrefix(issuerURL, "https://") {
		issuerURL = "https://" + issuerURL
	}

	// Construct discovery URL
	discoveryURL := issuerURL
	if !strings.HasSuffix(discoveryURL, "/") {
		discoveryURL += "/"
	}
	discoveryURL += ".well-known/openid-configuration"

	logger.Infof("Loading discovery document from: %s", discoveryURL)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(discoveryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to load discovery document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery document returned status %d", resp.StatusCode)
	}

	var discoveryDoc map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&discoveryDoc); err != nil {
		return nil, fmt.Errorf("failed to parse discovery document: %w", err)
	}

	return discoveryDoc, nil
}

// schemeFromRequest determines the scheme (http or https) from the request
func schemeFromRequest(c *gin.Context) string {
	// Check for X-Forwarded-Proto header (used by load balancers)
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		return proto
	}

	// Check if we're using TLS
	if c.Request.TLS != nil {
		return "https"
	}

	return "http"
}
