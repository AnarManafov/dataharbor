package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// GetUserToken retrieves the authorization token from either the Authorization header
// or from the session context (for cookie-based auth flows)
func GetUserToken(c *gin.Context) (string, bool) {
	// Check Authorization header first (API/programmatic access pattern)
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer "), true
	}

	// Fall back to session-based token (browser-based auth flow)
	if token, exists := c.Get("access_token"); exists {
		if tokenStr, ok := token.(string); ok && tokenStr != "" {
			return tokenStr, true
		}
	}

	return "", false
}

// GetUserClaims retrieves the parsed user identity information that was previously
// extracted and validated from the JWT token by SessionAuthMiddleware
func GetUserClaims(c *gin.Context) (map[string]any, bool) {
	if claims, exists := c.Get("user_claims"); exists {
		if userClaims, ok := claims.(map[string]any); ok {
			return userClaims, true
		}
	}

	return nil, false
}
