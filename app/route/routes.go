package route

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/AnarManafov/dataharbor/app/common"
	"github.com/AnarManafov/dataharbor/app/config"
	"github.com/AnarManafov/dataharbor/app/controller"
	"github.com/AnarManafov/dataharbor/app/middleware"
)

const (
	// IndexHTML is the main entry point for the Single Page Application
	IndexHTML = "index.html"
	// DefaultDistDir is the default distribution directory name
	DefaultDistDir = "dist"
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
	cfg := config.GetConfig()
	if cfg.Server.Debug {
		r.Use(middleware.DebugRequestBody())
	}

	// Add request tracing for distributed system observability
	r.Use(middleware.TraceRequest())

	// Public health check endpoint for monitoring and load balancers
	r.GET("/health", controller.HealthCheck)

	// Additional health endpoint under /api path for compatibility
	r.GET("/api/health", controller.HealthCheck)

	// Authentication routes - intentionally kept outside auth middleware
	// to allow initial authentication flows
	auth := r.Group("/api/auth")
	{
		auth.GET("/login", controller.LoginInit)
		auth.GET("/callback", controller.AuthCallback)
		auth.POST("/logout", controller.Logout)
		auth.GET("/user", controller.GetCurrentUser)
	}

	// Protected API routes - all require valid session
	api := r.Group("/api")
	api.Use(controller.SessionAuthMiddleware())

	// API v1 routes group for versioning
	v1 := api.Group("/v1")

	// XRootD file system exploration and download endpoints
	v1.GET("/xrd/ls", controller.ListDirectory)
	v1.GET("/xrd/initialDir", controller.FetchInitialDir)
	v1.GET("/xrd/download", controller.DownloadFile)
	v1.GET("/xrd/hostname", controller.FetchHostName)
	v1.POST("/xrd/ls/paged", controller.FetchDirItemsByPage)

	// Future: Multi-file download endpoints (interface prepared, implementation pending)
	// v1.POST("/xrd/download/batch", controller.DownloadMultipleFiles)     // Start batch download
	// v1.GET("/xrd/download/status/:id", controller.GetDownloadStatus)     // Check batch progress
	// v1.DELETE("/xrd/download/:id", controller.CancelDownload)            // Cancel download

	// Setup frontend static files
	setupStaticFiles(r)
}

// setupStaticFiles configures routes to serve frontend static files
func setupStaticFiles(r *gin.Engine) {
	logger := common.GetLogger()
	cfg := config.GetConfig()

	// Get frontend config settings
	distDir := cfg.Frontend.DistDir
	if distDir == "" {
		distDir = DefaultDistDir
	}

	// Find the frontend path
	frontendPath, indexFound, attemptedPaths := findFrontendPath(distDir)

	// If no frontend assets were found, log a clear error message
	if !indexFound {
		logFrontendNotFound(logger, attemptedPaths)
	} else {
		logger.Infof("Serving frontend from: %s", frontendPath)
	}

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

		logger.Infof("Non-API path requested: %s, serving %s", c.Request.URL.Path, IndexHTML)

		// Serve the SPA frontend for client-side routing to handle
		c.File(filepath.Join(frontendPath, IndexHTML))
	})
}

// findFrontendPath attempts to locate the frontend files in various locations
func findFrontendPath(distDir string) (string, bool, []string) {
	logger := common.GetLogger()
	frontendPath := filepath.Join("web", distDir)
	indexFound := false
	attemptedPaths := []string{}

	// Helper function to check a path for index.html
	checkPath := func(path string, logMessage string) bool {
		attemptedPaths = append(attemptedPaths, path)
		if hasIndexFile(path) {
			frontendPath = path
			indexFound = true
			logger.Infof(logMessage, path)
			return true
		}
		return false
	}

	// 1. Check default path
	defaultPath := filepath.Join("web", distDir)
	if checkPath(defaultPath, "Found frontend assets at default path: %s") {
		return frontendPath, indexFound, attemptedPaths
	}

	// 2. Try project root relative to current working directory
	workingDir, _ := os.Getwd()
	projectRootPath := filepath.Join(workingDir, "..", "web", distDir)
	if checkPath(projectRootPath, "Found frontend assets relative to project root: %s") {
		return frontendPath, indexFound, attemptedPaths
	}

	// 3. Try configured asset paths
	cfg := config.GetConfig()
	assetPaths := cfg.Frontend.AssetPaths
	for _, path := range assetPaths {
		// Try absolute path
		possiblePath := filepath.Join(path, distDir)
		if checkPath(possiblePath, "Found frontend assets at configured path: %s") {
			return frontendPath, indexFound, attemptedPaths
		}

		// Try relative to working directory
		workingDirRelativePath := filepath.Join(workingDir, path, distDir)
		if checkPath(workingDirRelativePath, "Found frontend assets at working dir relative path: %s") {
			return frontendPath, indexFound, attemptedPaths
		}
	}

	// 4. Try executable directory
	if execDir, err := filepath.Abs(filepath.Dir(os.Args[0])); err == nil {
		// Direct executable directory
		possiblePath := filepath.Join(execDir, "web", distDir)
		if checkPath(possiblePath, "Found frontend assets relative to executable: %s") {
			return frontendPath, indexFound, attemptedPaths
		}

		// One directory up from executable
		possiblePathUp := filepath.Join(execDir, "..", "web", distDir)
		if checkPath(possiblePathUp, "Found frontend assets in parent of executable dir: %s") {
			return frontendPath, indexFound, attemptedPaths
		}
	}

	// 5. Try to find project root by looking for key folders
	findByProjectRoot(workingDir, distDir, &frontendPath, &indexFound, &attemptedPaths)

	// 6. Try looking for sandbox directory
	if !indexFound {
		findBySandboxDirectory(workingDir, &frontendPath, &indexFound, &attemptedPaths)
	}

	return frontendPath, indexFound, attemptedPaths
}

// findByProjectRoot searches for the project root by looking for key directories
func findByProjectRoot(startDir, distDir string, frontendPath *string, indexFound *bool, attemptedPaths *[]string) {
	if *indexFound {
		return
	}

	logger := common.GetLogger()
	currentDir := startDir

	// Try going up directories until we find the project root
	for i := 0; i < 5; i++ {
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break // Stop if we can't go up anymore
		}
		currentDir = parentDir

		// Check if this looks like the project root
		webDir := filepath.Join(currentDir, "web")
		appDir := filepath.Join(currentDir, "app")
		if dirExists(webDir) && dirExists(appDir) {
			possiblePath := filepath.Join(webDir, distDir)
			*attemptedPaths = append(*attemptedPaths, possiblePath)

			if hasIndexFile(possiblePath) {
				*frontendPath = possiblePath
				*indexFound = true
				logger.Infof("Found frontend assets by locating project root: %s", possiblePath)
				return
			}
		}
	}
}

// findBySandboxDirectory searches for the sandbox directory which may contain frontend files
func findBySandboxDirectory(startDir string, frontendPath *string, indexFound *bool, attemptedPaths *[]string) {
	logger := common.GetLogger()
	currentDir := startDir

	// Try going up directories until we find the sandbox directory
	for i := 0; i < 5; i++ {
		// Check current directory for sandbox
		sandboxPath := filepath.Join(currentDir, "sandbox", "public")
		*attemptedPaths = append(*attemptedPaths, sandboxPath)

		if hasIndexFile(sandboxPath) {
			*frontendPath = sandboxPath
			*indexFound = true
			logger.Infof("Found frontend assets in sandbox directory: %s", sandboxPath)
			return
		}

		// Try going up one level
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break // Stop if we can't go up anymore
		}
		currentDir = parentDir
	}
}

// logFrontendNotFound logs details when frontend assets can't be found
func logFrontendNotFound(logger *zap.SugaredLogger, attemptedPaths []string) {
	logger.Error("Could not find frontend assets (" + IndexHTML + ") at any configured location.")
	logger.Error("Please build the frontend or configure correct asset paths in the configuration.")
	logger.Error("Checked locations:")

	// Print all attempted paths in the error
	for _, path := range attemptedPaths {
		logger.Errorf("  - %s", path)
	}

	// Show more diagnostic information
	workingDir, _ := os.Getwd()
	logger.Errorf("Current working directory: %s", workingDir)
	logger.Error("The UI will not be available.")
}

// hasIndexFile checks if the given directory contains an index.html file
func hasIndexFile(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, IndexHTML))
	return err == nil
}

// dirExists checks if the given path exists and is a directory
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// RegisterRoutes is provided for backward compatibility
func RegisterRoutes(r *gin.Engine) {
	SetupRouter(r)
}
