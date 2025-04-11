package route

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"github.com/AnarManafov/data_lake_ui/app/common"
	"github.com/AnarManafov/data_lake_ui/app/controller"
	"github.com/AnarManafov/data_lake_ui/app/middleware"
	"github.com/spf13/viper"
)

// SetupRouter configures the API and static file routes for the application
// This function establishes the application's routing structure, middleware chain,
// and serves the SPA frontend for non-API routes
func SetupRouter(r *gin.Engine) {
	// Add recovery middleware for crash prevention and graceful error handling
	r.Use(middleware.Recovery())

	// Enable cross-origin requests to support frontend-backend separation
	r.Use(middleware.CORS())

	// Enable request/response logging in debug mode to aid development
	if viper.GetBool("server.debug") {
		r.Use(middleware.DebugRequestBody())
	}

	// Add request tracing for distributed system observability
	r.Use(middleware.TraceRequest())

	// Public health check endpoint for monitoring and load balancers
	r.GET("/health", controller.HealthCheck)

	// Authentication routes - intentionally kept outside auth middleware
	// to allow initial authentication flows
	auth := r.Group("/api/auth")
	{
		auth.GET("/login", controller.LoginInit)
		auth.GET("/callback", controller.AuthCallback)
		auth.GET("/userinfo", controller.GetUserInfo)
		auth.POST("/logout", controller.Logout)
		auth.GET("/user", controller.GetCurrentUser)
	}

	// Protected API routes - all require valid session
	api := r.Group("/api")
	api.Use(controller.SessionAuthMiddleware())

	// XRootD file system exploration and download endpoints
	api.GET("/xrd/ls", controller.ListDirectory)
	api.GET("/xrd/initialDir", controller.FetchInitialDir)
	api.POST("/xrd/stage", controller.FetchFileStagedForDownload)
	api.GET("/xrd/hostname", controller.FetchHostName)
	api.POST("/xrd/ls/paged", controller.FetchDirItemsByPage)

	// Locate the frontend static assets using a multi-stage search strategy
	// to support various deployment scenarios (development, container, etc.)
	logger := common.GetLogger()
	frontendPath := filepath.Join("web", "dist")
	if _, err := os.Stat(filepath.Join(frontendPath, "index.html")); os.IsNotExist(err) {
		// Try executable directory if working directory fails
		execDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err == nil {
			frontendPath = filepath.Join(execDir, "web", "dist")
		}

		// Development fallback path
		if _, err := os.Stat(filepath.Join(frontendPath, "index.html")); os.IsNotExist(err) {
			frontendPath = "/Users/anarmanafov/Documents/workspace/data-lake-ui/web/dist"
		}
	}

	logger.Infof("Serving frontend from: %s", frontendPath)

	// Serve frontend static assets
	r.StaticFS("/assets", http.Dir(filepath.Join(frontendPath, "assets")))
	r.StaticFile("/favicon.ico", filepath.Join(frontendPath, "assets", "favicon.ico"))
	r.StaticFile("/config.json", filepath.Join(frontendPath, "config.json"))

	// Handle all unmatched routes - API 404s vs SPA routing
	r.NoRoute(func(c *gin.Context) {
		// Return proper 404 for non-existent API paths
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"message": "API endpoint not found"})
			return
		}

		logger.Infof("Non-API path requested: %s, serving index.html", c.Request.URL.Path)

		// Serve the SPA frontend for client-side routing to handle
		c.File(filepath.Join(frontendPath, "index.html"))
	})
}

// RegisterRoutes is provided for backward compatibility
func RegisterRoutes(r *gin.Engine) {
	SetupRouter(r)
}
