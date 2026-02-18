package controller

// Authentication implemented in a single file to maintain cohesion and simplify maintenance.

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

	"github.com/AnarManafov/dataharbor/app/common"
	"github.com/AnarManafov/dataharbor/app/config"
	"github.com/AnarManafov/dataharbor/app/response"
)

const (
	sessionName   = "dataharbor-session"
	sessionMaxAge = 86400 * 7 // 7 days - chosen to balance user convenience with security risks
)

// Will be initialized in init()
var SessionStore *sessions.CookieStore

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

func generateTokenID() string {
	return uuid.New().String()
}
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

func getTokens(tokenID string) (TokenInfo, bool) {
	tokens, ok := tokenStore[tokenID]
	return tokens, ok
}
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

func deleteTokens(tokenID string) {
	delete(tokenStore, tokenID)
}

// InitAuth initializes the authentication system
// This should be called during application startup after config is loaded
func InitAuth() {
	cfg := config.GetConfig()
	logger := common.GetLogger()
	// Prefer configured secret for persistent sessions between restarts
	sessionSecret := cfg.Auth.OIDC.SessionSecret
	if sessionSecret == "" {
		// Generate a secure but non-persistent secret as fallback
		b := make([]byte, 32)
		_, err := rand.Read(b)
		if err != nil {
			logger.Error("Failed to generate random session secret", "error", err)
			// Last resort fallback with explicit warning
			sessionSecret = "default-session-secret-replace-in-production"
		} else {
			sessionSecret = base64.StdEncoding.EncodeToString(b)
			logger.Warn("Generated random session secret. Sessions will be invalidated on server restart. Set auth.oidc.session_secret in your config for persistence.")
		}
	}

	// Initialize the session store with the secret
	SessionStore = sessions.NewCookieStore([]byte(sessionSecret))
	// Balance security with environment-specific needs
	sameSiteMode := http.SameSiteStrictMode
	secureCookies := true
	// Use relaxed settings in development unless SSL is enabled
	if cfg.Env == "development" && !cfg.Server.SSL.Enabled {
		sameSiteMode = http.SameSiteLaxMode
		secureCookies = false
		logger.Info("Running in development mode without SSL: using relaxed cookie settings")
	} else if cfg.Env == "development" && cfg.Server.SSL.Enabled {
		sameSiteMode = http.SameSiteLaxMode
		secureCookies = true
		logger.Info("Running in development mode with SSL: using secure cookies")
	}
	// Apply security settings appropriate for the environment
	SessionStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   sessionMaxAge,
		HttpOnly: true,          // Mitigate XSS risks
		Secure:   secureCookies, // Require HTTPS when SSL is enabled
		SameSite: sameSiteMode,  // Balance cross-origin needs vs CSRF protection
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

// LoginInit begins authentication by creating an OIDC authorization URL
func LoginInit(c *gin.Context) {
	cfg := config.GetConfig()
	logger := common.GetLogger() // Log detailed request info to help diagnose auth integration issues
	logger.Infof("LOGIN REQUEST - Headers: %v", c.Request.Header)
	logger.Infof("LOGIN REQUEST - Query params: %v", c.Request.URL.Query())
	logger.Infof("LOGIN REQUEST - Remote addr: %v", c.Request.RemoteAddr)
	logger.Infof("Auth configuration - Enabled: %v", cfg.Auth.Enabled)
	logger.Infof("Auth configuration - OIDC Issuer: %v", cfg.Auth.OIDC.Issuer)
	logger.Infof("Auth configuration - OIDC ClientID: %v", cfg.Auth.OIDC.ClientID)
	logger.Infof("Auth configuration - OIDC ClientSecret set: %v", cfg.Auth.OIDC.ClientSecret != "")
	logger.Infof("Auth configuration - Session Secret set: %v", cfg.Auth.OIDC.SessionSecret != "")

	// Show part of secret for debugging while maintaining security
	secretForLog := "not-set"
	if cfg.Auth.OIDC.ClientSecret != "" {
		if len(cfg.Auth.OIDC.ClientSecret) >= 4 {
			secretForLog = cfg.Auth.OIDC.ClientSecret[:4] + "..."
		} else {
			secretForLog = "[too short]"
		}
	}

	// Confirm OIDC settings are properly loaded
	logger.Infof("OIDC Config - Issuer: %s, ClientID: %s, ClientSecret: %s",
		cfg.Auth.OIDC.Issuer,
		cfg.Auth.OIDC.ClientID,
		secretForLog)

	// Identify missing OIDC configuration early
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

	// Allow auth bypass for testing and development
	if !cfg.Auth.Enabled {
		logger.Warn("Authentication is disabled in configuration, but login endpoint was called")
		c.JSON(http.StatusOK, gin.H{
			"auth_url": "/auth/disabled",
			"message":  "Authentication is disabled",
		})
		return
	}

	// Prevent confusing auth failures due to incomplete config
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

	// Cryptographically strong state parameter prevents CSRF attacks
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
	secureCookies := cfg.Server.SSL.Enabled
	sameSiteMode := http.SameSiteNoneMode

	// In development with SSL, use Lax mode for easier testing
	if cfg.Env == "development" && cfg.Server.SSL.Enabled {
		sameSiteMode = http.SameSiteLaxMode
	}

	session.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   sessionMaxAge,
		HttpOnly: true,
		SameSite: sameSiteMode,
		Secure:   secureCookies, // Use secure cookies when SSL is enabled
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

// AuthCallback completes the authentication flow after user authorizes with the IdP
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

	// Validate state to ensure request integrity and prevent CSRF
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
	// Timeout prevents hanging during network issues
	client := &http.Client{Timeout: 10 * time.Second}
	tokenResp, err := client.Do(req)
	if err != nil {
		logger.Error("Failed to exchange code for tokens", "error", err)
		response.Error(c, http.StatusInternalServerError, "Failed to process authentication")
		return
	}
	defer func() { _ = tokenResp.Body.Close() }()

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
				// Use HTTPS for development since Keycloak now requires it
				scheme := "https"
				if !cfg.Server.SSL.Enabled {
					scheme = "http"
				}
				frontendURL = fmt.Sprintf("%s://localhost:5173", scheme)
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

// Logout terminates both local and IdP sessions to prevent token reuse
func Logout(c *gin.Context) {
	logger := common.GetLogger()
	cfg := config.GetConfig()

	// Get session
	session, err := SessionStore.Get(c.Request, sessionName)
	if err == nil {
		// Get token ID to delete from store
		if tokenID, ok := session.Values["token_id"].(string); ok && tokenID != "" {
			// Get tokens before removing them, we need them to call Keycloak logout
			tokens, ok := getTokens(tokenID)
			if ok && tokens.IDToken != "" {
				// Try to load discovery document to get end session endpoint
				discoveryDoc, err := fetchOIDCDiscoveryDocument(cfg.Auth.OIDC.Issuer)
				if err == nil {
					// Get end_session_endpoint from discovery document
					if endSessionEndpoint, ok := discoveryDoc["end_session_endpoint"].(string); ok && endSessionEndpoint != "" {
						logger.Infof("Using end session endpoint: %s", endSessionEndpoint)

						// Build the logout URL with id_token_hint
						logoutURL := fmt.Sprintf("%s?id_token_hint=%s&client_id=%s",
							endSessionEndpoint,
							url.QueryEscape(tokens.IDToken),
							url.QueryEscape(cfg.Auth.OIDC.ClientID))

						// Create HTTP client and make request to log out from Keycloak
						client := &http.Client{Timeout: 5 * time.Second}
						req, err := http.NewRequest("GET", logoutURL, nil)
						if err == nil { // Run logout asynchronously to prevent delays in user experience while maintaining security
							go func() {
								resp, err := client.Do(req)
								if err != nil {
									logger.Warnf("Failed to logout from OIDC provider: %v", err)
								} else {
									defer func() { _ = resp.Body.Close() }()

									// HTTP status validation ensures the token is properly invalidated at the IdP level
									// This prevents potential security issues with lingering active sessions
									if resp.StatusCode >= 200 && resp.StatusCode < 300 {
										logger.Info("Successfully logged out from OIDC provider, status: %d", resp.StatusCode)
									} else {
										// Error details help diagnose integration issues with the OIDC provider
										body, readErr := io.ReadAll(resp.Body)
										if readErr != nil {
											logger.Warnf("OIDC provider logout returned status %d, but couldn't read response body: %v", resp.StatusCode, readErr)
										} else {
											logger.Warnf("OIDC provider logout failed with status %d: %s", resp.StatusCode, string(body))
										}
									}
								}
							}()
						} else {
							logger.Warnf("Failed to create request to OIDC logout endpoint: %v", err)
						}
					} else {
						logger.Warn("No end session endpoint found in OIDC discovery document")
					}
				} else {
					logger.Warnf("Failed to fetch OIDC discovery document for logout: %v", err)
				}
			}
			// Delete tokens from the store for security
			deleteTokens(tokenID)
			logger.Info("Removed tokens from store during logout")
		}
	}

	// Remove all session data and expire the cookie
	session.Values = make(map[any]any)
	session.Options.MaxAge = -1

	if err := session.Save(c.Request, c.Writer); err != nil {
		logger.Error("Failed to clear session", "error", err)
		response.Error(c, http.StatusInternalServerError, "Failed to complete logout")
		return
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Logout successful"})
}

// Health provides a lightweight endpoint for monitoring and readiness checks
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"data":    "ok",
		"code":    200,
		"message": "success",
	})
}

// refreshToken renews authentication without requiring user interaction
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
	defer func() { _ = refreshResp.Body.Close() }()

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

// SessionAuthMiddleware ensures routes are protected and handles automatic token refresh
func SessionAuthMiddleware() gin.HandlerFunc {
	cfg := config.GetConfig()
	logger := common.GetLogger()
	// Skip authentication when disabled to facilitate development and testing
	if !cfg.Auth.Enabled {
		logger.Info("Authentication is disabled")
		return func(c *gin.Context) {
			c.Next()
		}
	}
	// Allow certain paths to remain public for system functionality
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
		// Skip auth for public endpoints
		if skipAuth(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Verify session integrity
		session, err := SessionStore.Get(c.Request, sessionName)
		if err != nil {
			logger.Error("Failed to get session", "error", err)
			response.Error(c, http.StatusUnauthorized, "Invalid session")
			c.Abort()
			return
		}

		// Token ID presence confirms previous authentication
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
		// Proactively refresh tokens to prevent disruption of user experience
		// Use configured refresh buffer instead of hardcoded value
		cfg := config.GetConfig()
		refreshBuffer := cfg.Auth.OIDC.TokenRefreshBufferSec
		if refreshBuffer <= 0 {
			refreshBuffer = 60 // Fallback to 1 minute if not configured
		}
		currentTime := time.Now().Unix()

		// Handle expiring or expired tokens
		if currentTime > tokens.ExpiresAt || (tokens.ExpiresAt-currentTime) < refreshBuffer {
			logger.Info("Token expired or expiring soon, refreshing")
			if err := refreshToken(c); err != nil {
				logger.Error("Failed to refresh token", "error", err)

				// If token is already expired, return unauthorized
				if currentTime > tokens.ExpiresAt {
					response.Error(c, http.StatusUnauthorized, "Session expired")
					c.Abort()
					return
				} // Proceed with existing token to avoid disrupting user's current action
				logger.Warn("Using existing token despite failed refresh attempt")
			} else {
				// Get the new tokens after successful refresh
				tokens, _ = getTokens(tokenID)
				logger.Info("Successfully refreshed token before expiration")
			}
		}

		// Fetch user claims to ensure we have the subject for rate limiting
		userClaims := map[string]any{
			"authenticated": true,
		}

		// Try to get user info to extract the subject claim
		if userInfo, err := fetchUserInfo(tokens.AccessToken, cfg); err == nil {
			// Extract the subject claim for rate limiting
			if sub, ok := userInfo["sub"].(string); ok {
				userClaims["sub"] = sub
			}
		} else {
			logger.Warn("Failed to fetch user info for claims", "error", err)
		}

		// Make token available to downstream handlers for authorization
		c.Set("access_token", tokens.AccessToken)
		c.Set("user_claims", userClaims)
		// Forward token to enable microservice authorization
		c.Request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokens.AccessToken))

		c.Next()
	}
}

// Helper functions

// fetchOIDCDiscoveryDocument retrieves IdP configuration to avoid hardcoding endpoints
func fetchOIDCDiscoveryDocument(issuerURL string) (map[string]any, error) {
	logger := common.GetLogger()

	// Add protocol if missing to prevent connection errors
	if !strings.HasPrefix(issuerURL, "http://") && !strings.HasPrefix(issuerURL, "https://") {
		issuerURL = "https://" + issuerURL
	}

	// Ensure URL follows OIDC specification format
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery document returned status %d", resp.StatusCode)
	}

	var discoveryDoc map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&discoveryDoc); err != nil {
		return nil, fmt.Errorf("failed to parse discovery document: %w", err)
	}

	return discoveryDoc, nil
}

// schemeFromRequest handles both direct and proxy connections correctly
func schemeFromRequest(c *gin.Context) string {
	cfg := config.GetConfig()

	// Support for reverse proxy environments - check forwarded headers first
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		return proto
	}

	// Check for other common proxy headers
	if proto := c.GetHeader("X-Forwarded-Protocol"); proto != "" {
		return proto
	}

	if proto := c.GetHeader("X-Scheme"); proto != "" {
		return proto
	}

	// Check if we're using TLS directly
	if c.Request.TLS != nil {
		return "https"
	}

	// Check if SSL is enabled in configuration (for direct HTTPS connections)
	if cfg.Server.SSL.Enabled {
		return "https"
	}

	return "http"
}
