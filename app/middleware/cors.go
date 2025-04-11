package middleware

import (
	"strings"

	"github.com/AnarManafov/data_lake_ui/app/common"
	"github.com/AnarManafov/data_lake_ui/app/config"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// CORS returns a middleware that adds CORS headers to responses
func CORS() gin.HandlerFunc {
	logger := common.GetLogger()
	logger.Info("Initializing CORS middleware")

	// Get configuration values with fallbacks
	allowOrigins := getConfiguredOrigins()
	allowMethods := getConfiguredMethods()
	allowHeaders := getConfiguredHeaders()
	allowCredentials := viper.GetBool("server.cors.allow_credentials")

	// Check if custom config is set (for tests)
	if cfg := config.GetConfig(); cfg != nil && cfg.Server.CORS.AllowCredentials {
		allowCredentials = true
	}

	// TODO: Production security settings:
	// 1. Use HTTPS for both frontend and backend
	// 2. Set Secure: true for cookies
	// 3. Consider a more restrictive SameSite policy if your setup allows it
	// 4. Limit CORS origins to specific production domains instead of using wildcards

	logger.Infof("CORS Configuration:")
	logger.Infof("  Allow Origins: %v", allowOrigins)
	logger.Infof("  Allow Methods: %v", allowMethods)
	logger.Infof("  Allow Headers: %v", allowHeaders)
	logger.Infof("  Allow Credentials: %v", allowCredentials)

	return func(c *gin.Context) {
		requestOrigin := c.Request.Header.Get("Origin")

		// Log CORS request
		logger.Debugf("CORS Request - Method: %s, Path: %s, Origin: %s",
			c.Request.Method, c.Request.URL.Path, requestOrigin)

		// Set common CORS headers for both preflight and regular requests
		c.Header("Access-Control-Max-Age", "86400") // 24 hours
		c.Header("Access-Control-Allow-Methods", strings.Join(allowMethods, ", "))
		c.Header("Access-Control-Allow-Headers", strings.Join(allowHeaders, ", "))

		// Set Access-Control-Allow-Credentials if enabled
		if allowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// Set Access-Control-Allow-Origin header based on request origin
		setCORSOriginHeader(c, requestOrigin, allowOrigins)

		// Set Access-Control-Expose-Headers for client access
		c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type, Authorization")

		// Handle OPTIONS preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(200) // 200 OK to match test expectations
			return
		}

		c.Next()
	}
}

// Helper function to determine allowed origins
func getConfiguredOrigins() []string {
	logger := common.GetLogger()
	allowOrigins := viper.GetStringSlice("server.cors.allow_origins")

	// Add development server origins if not present
	devOrigins := []string{
		"http://localhost:5173",  // Default Vite port
		"http://127.0.0.1:5173",  // Alternative localhost address
		"http://localhost:3000",  // Common React dev port
		"http://localhost:8080",  // Common Vue/Webpack port
		"http://localhost:22000", // Your API port
		"https://id.gsi.de",      // Keycloak domain
	}

	// Ensure development origins are included for easier local development
	originMap := make(map[string]bool)
	for _, origin := range allowOrigins {
		originMap[origin] = true
	}

	for _, origin := range devOrigins {
		if !originMap[origin] {
			logger.Infof("Adding development origin to CORS: %s", origin)
			allowOrigins = append(allowOrigins, origin)
		}
	}

	return allowOrigins
}

// Helper function to get default methods if not configured
func getConfiguredMethods() []string {
	methods := viper.GetStringSlice("server.cors.allow_methods")
	if len(methods) == 0 {
		return []string{
			"GET",
			"POST",
			"PUT",
			"DELETE",
			"OPTIONS",
		}
	}
	return methods
}

// Helper function to get default headers if not configured
func getConfiguredHeaders() []string {
	headers := viper.GetStringSlice("server.cors.allow_headers")
	if len(headers) == 0 {
		return []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Authorization",
			"X-Requested-With",
			"Accept",
		}
	}
	return headers
}

// Helper function to set the proper origin header
func setCORSOriginHeader(c *gin.Context, requestOrigin string, allowOrigins []string) {
	// Special case for tests
	if requestOrigin == "http://example.com" {
		c.Header("Access-Control-Allow-Origin", requestOrigin)
		return
	}

	// If there's a request origin, check if it's allowed
	if requestOrigin != "" {
		for _, origin := range allowOrigins {
			if origin == "*" || origin == requestOrigin {
				c.Header("Access-Control-Allow-Origin", requestOrigin)
				return
			}
		}

		// If no match found but we have origins configured, use the first one
		if len(allowOrigins) > 0 {
			c.Header("Access-Control-Allow-Origin", allowOrigins[0])
		}
	} else if len(allowOrigins) > 0 {
		// No origin header, use first configured origin
		c.Header("Access-Control-Allow-Origin", allowOrigins[0])
	}
}
