package route

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/AnarManafov/dataharbor/app/common"
	"github.com/AnarManafov/dataharbor/app/config"
)

func TestRegisterRoutes(t *testing.T) {
	// Set up gin in test mode
	gin.SetMode(gin.TestMode)

	// Initialize config for testing
	testConfig := &config.Config{
		Server: config.ServerConfig{
			Address: "localhost:8080",
			Debug:   false,
		},
		XRD: config.XRDConfig{
			Host:       "localhost",
			Port:       1094,
			InitialDir: "/tmp",
		},
		Logging: config.LoggingConfig{
			Level: "info",
			Console: config.ConsoleConfig{
				Enabled: true,
				Format:  "text",
				Level:   "info",
			},
			File: config.FileConfig{
				Enabled: false,
			},
		},
	}

	// Set the config
	config.SetConfig(testConfig)

	// Initialize logger
	common.InitLogger(&testConfig.Logging)

	r := gin.New()

	// Register routes
	RegisterRoutes(r)

	// Create test server
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Test health endpoint
	resp, err := http.Get(ts.URL + "/health")
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Test API endpoints with correct paths
	resp, err = http.Get(ts.URL + "/api/v1/xrd/initialDir")
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	resp, err = http.Get(ts.URL + "/api/v1/xrd/hostname")
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestHasIndexFile(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Test directory without index.html
	assert.False(t, hasIndexFile(tmpDir), "Directory without index.html should return false")

	// Create index.html
	indexPath := filepath.Join(tmpDir, IndexHTML)
	err := os.WriteFile(indexPath, []byte("<html></html>"), 0o644)
	assert.NoError(t, err)

	// Test directory with index.html
	assert.True(t, hasIndexFile(tmpDir), "Directory with index.html should return true")
}

func TestDirExists(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Test existing directory
	assert.True(t, dirExists(tmpDir), "Existing directory should return true")

	// Test non-existing directory
	assert.False(t, dirExists(filepath.Join(tmpDir, "nonexistent")), "Non-existing directory should return false")

	// Test file (not a directory)
	filePath := filepath.Join(tmpDir, "testfile.txt")
	err := os.WriteFile(filePath, []byte("content"), 0o644)
	assert.NoError(t, err)
	assert.False(t, dirExists(filePath), "File path should return false")
}

func TestFindBySandboxDirectory(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()

	// Create sandbox/public directory with index.html
	sandboxPublic := filepath.Join(tmpDir, "sandbox", "public")
	err := os.MkdirAll(sandboxPublic, 0o755)
	assert.NoError(t, err)

	indexPath := filepath.Join(sandboxPublic, IndexHTML)
	err = os.WriteFile(indexPath, []byte("<html></html>"), 0o644)
	assert.NoError(t, err)

	// Test finding sandbox directory
	var frontendPath string
	var indexFound bool
	var attemptedPaths []string

	findBySandboxDirectory(tmpDir, &frontendPath, &indexFound, &attemptedPaths)

	assert.True(t, indexFound, "Should find index.html in sandbox/public")
	assert.Equal(t, sandboxPublic, frontendPath)
}

func TestFindBySandboxDirectory_NotFound(t *testing.T) {
	// Create a temporary directory without sandbox
	tmpDir := t.TempDir()

	var frontendPath string
	var indexFound bool
	var attemptedPaths []string

	findBySandboxDirectory(tmpDir, &frontendPath, &indexFound, &attemptedPaths)

	assert.False(t, indexFound, "Should not find index.html")
	assert.Greater(t, len(attemptedPaths), 0, "Should have attempted some paths")
}

func TestLogFrontendNotFound(t *testing.T) {
	// Create a test logger
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()

	// This just tests that the function doesn't panic
	// Since it only logs, we verify it runs without error
	attemptedPaths := []string{
		"/path/one",
		"/path/two",
		"/path/three",
	}

	// Should not panic
	assert.NotPanics(t, func() {
		logFrontendNotFound(sugar, attemptedPaths)
	})
}

func TestFindByProjectRoot(t *testing.T) {
	// Create a temporary project structure
	tmpDir := t.TempDir()

	// Create app and web directories to simulate project root
	appDir := filepath.Join(tmpDir, "app")
	webDir := filepath.Join(tmpDir, "web")
	distDir := filepath.Join(webDir, "dist")

	err := os.MkdirAll(appDir, 0o755)
	assert.NoError(t, err)
	err = os.MkdirAll(distDir, 0o755)
	assert.NoError(t, err)

	// Create index.html in dist
	indexPath := filepath.Join(distDir, IndexHTML)
	err = os.WriteFile(indexPath, []byte("<html></html>"), 0o644)
	assert.NoError(t, err)

	// Test from subdirectory
	subDir := filepath.Join(tmpDir, "app", "sub")
	err = os.MkdirAll(subDir, 0o755)
	assert.NoError(t, err)

	var frontendPath string
	var indexFound bool
	var attemptedPaths []string

	findByProjectRoot(subDir, "dist", &frontendPath, &indexFound, &attemptedPaths)

	assert.True(t, indexFound, "Should find index.html by project root detection")
	assert.Equal(t, distDir, frontendPath)
}

func TestFindByProjectRoot_AlreadyFound(t *testing.T) {
	tmpDir := t.TempDir()

	frontendPath := "/already/found"
	indexFound := true
	attemptedPaths := []string{}

	// Should return early when already found
	findByProjectRoot(tmpDir, "dist", &frontendPath, &indexFound, &attemptedPaths)

	assert.True(t, indexFound)
	assert.Equal(t, "/already/found", frontendPath, "Path should not change when already found")
	assert.Empty(t, attemptedPaths, "No paths should be attempted when already found")
}

func TestFindFrontendPath_DefaultPath(t *testing.T) {
	// Get current directory
	cwd, err := os.Getwd()
	assert.NoError(t, err)

	// Create temporary web/dist directory with index.html
	tmpDir := t.TempDir()
	webDist := filepath.Join(tmpDir, "web", "dist")
	err = os.MkdirAll(webDist, 0o755)
	assert.NoError(t, err)

	indexPath := filepath.Join(webDist, IndexHTML)
	err = os.WriteFile(indexPath, []byte("<html></html>"), 0o644)
	assert.NoError(t, err)

	// Change to tmpDir temporarily
	err = os.Chdir(tmpDir)
	assert.NoError(t, err)
	defer func() { _ = os.Chdir(cwd) }()

	// Set up config
	testConfig := &config.Config{
		Frontend: config.FrontendConfig{
			DistDir: "dist",
		},
		Logging: config.LoggingConfig{
			Level:   "info",
			Console: config.ConsoleConfig{Enabled: true, Format: "text", Level: "info"},
		},
	}
	config.SetConfig(testConfig)
	common.InitLogger(&testConfig.Logging)

	// Test findFrontendPath
	path, found, attemptedPaths := findFrontendPath("dist")

	assert.True(t, found, "Should find index.html at default path")
	// The path returned is relative "web/dist" not absolute
	assert.Contains(t, path, "dist", "Path should contain dist")
	assert.Greater(t, len(attemptedPaths), 0, "Should have attempted paths")
}

func TestFindFrontendPath_NotFound(t *testing.T) {
	// Get current directory
	cwd, err := os.Getwd()
	assert.NoError(t, err)

	// Create temporary directory without web/dist
	tmpDir := t.TempDir()

	// Change to tmpDir temporarily
	err = os.Chdir(tmpDir)
	assert.NoError(t, err)
	defer func() { _ = os.Chdir(cwd) }()

	// Set up config with empty asset paths
	testConfig := &config.Config{
		Frontend: config.FrontendConfig{
			DistDir:    "dist",
			AssetPaths: []string{},
		},
		Logging: config.LoggingConfig{
			Level:   "info",
			Console: config.ConsoleConfig{Enabled: true, Format: "text", Level: "info"},
		},
	}
	config.SetConfig(testConfig)
	common.InitLogger(&testConfig.Logging)

	// Test findFrontendPath
	_, found, attemptedPaths := findFrontendPath("dist")

	assert.False(t, found, "Should not find index.html")
	assert.Greater(t, len(attemptedPaths), 0, "Should have attempted multiple paths")
}

func TestFindFrontendPath_ConfiguredAssetPath(t *testing.T) {
	// Get current directory
	cwd, err := os.Getwd()
	assert.NoError(t, err)

	// Create temporary directory structure
	tmpDir := t.TempDir()
	customPath := filepath.Join(tmpDir, "custom", "frontend")
	distPath := filepath.Join(customPath, "dist")
	err = os.MkdirAll(distPath, 0o755)
	assert.NoError(t, err)

	indexPath := filepath.Join(distPath, IndexHTML)
	err = os.WriteFile(indexPath, []byte("<html></html>"), 0o644)
	assert.NoError(t, err)

	// Change to tmpDir temporarily
	err = os.Chdir(tmpDir)
	assert.NoError(t, err)
	defer func() { _ = os.Chdir(cwd) }()

	// Set up config with custom asset path
	testConfig := &config.Config{
		Frontend: config.FrontendConfig{
			DistDir:    "dist",
			AssetPaths: []string{filepath.Join(tmpDir, "custom", "frontend")},
		},
		Logging: config.LoggingConfig{
			Level:   "info",
			Console: config.ConsoleConfig{Enabled: true, Format: "text", Level: "info"},
		},
	}
	config.SetConfig(testConfig)
	common.InitLogger(&testConfig.Logging)

	// Test findFrontendPath
	path, found, _ := findFrontendPath("dist")

	assert.True(t, found, "Should find index.html at configured asset path")
	assert.Equal(t, distPath, path)
}

func TestSetupStaticFiles_WithFrontend(t *testing.T) {
	// Get current directory
	cwd, err := os.Getwd()
	assert.NoError(t, err)

	// Create temporary directory structure with frontend files
	tmpDir := t.TempDir()
	webDist := filepath.Join(tmpDir, "web", "dist")
	assetsDir := filepath.Join(webDist, "assets")
	err = os.MkdirAll(assetsDir, 0o755)
	assert.NoError(t, err)

	// Create index.html
	indexPath := filepath.Join(webDist, IndexHTML)
	err = os.WriteFile(indexPath, []byte("<html></html>"), 0o644)
	assert.NoError(t, err)

	// Create favicon.ico
	faviconPath := filepath.Join(assetsDir, "favicon.ico")
	err = os.WriteFile(faviconPath, []byte("icon"), 0o644)
	assert.NoError(t, err)

	// Create config.json
	configPath := filepath.Join(webDist, "config.json")
	err = os.WriteFile(configPath, []byte("{}"), 0o644)
	assert.NoError(t, err)

	// Change to tmpDir temporarily
	err = os.Chdir(tmpDir)
	assert.NoError(t, err)
	defer func() { _ = os.Chdir(cwd) }()

	// Set up config
	testConfig := &config.Config{
		Frontend: config.FrontendConfig{
			DistDir: "dist",
		},
		Logging: config.LoggingConfig{
			Level:   "info",
			Console: config.ConsoleConfig{Enabled: true, Format: "text", Level: "info"},
		},
	}
	config.SetConfig(testConfig)
	common.InitLogger(&testConfig.Logging)

	gin.SetMode(gin.TestMode)
	r := gin.New()

	// This should not panic
	assert.NotPanics(t, func() {
		setupStaticFiles(r)
	})
}

func TestSetupStaticFiles_NoRoute(t *testing.T) {
	// Get current directory
	cwd, err := os.Getwd()
	assert.NoError(t, err)

	// Create temporary directory structure
	tmpDir := t.TempDir()
	webDist := filepath.Join(tmpDir, "web", "dist")
	assetsDir := filepath.Join(webDist, "assets")
	err = os.MkdirAll(assetsDir, 0o755)
	assert.NoError(t, err)

	indexPath := filepath.Join(webDist, IndexHTML)
	err = os.WriteFile(indexPath, []byte("<html></html>"), 0o644)
	assert.NoError(t, err)

	// Change to tmpDir temporarily
	err = os.Chdir(tmpDir)
	assert.NoError(t, err)
	defer func() { _ = os.Chdir(cwd) }()

	// Set up config
	testConfig := &config.Config{
		Frontend: config.FrontendConfig{
			DistDir: "dist",
		},
		Logging: config.LoggingConfig{
			Level:   "info",
			Console: config.ConsoleConfig{Enabled: true, Format: "text", Level: "info"},
		},
	}
	config.SetConfig(testConfig)
	common.InitLogger(&testConfig.Logging)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	setupStaticFiles(r)

	// Test NoRoute handler for API path
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/nonexistent", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "API endpoint not found")
}

func TestSetupStaticFiles_SPARouting(t *testing.T) {
	// Get current directory
	cwd, err := os.Getwd()
	assert.NoError(t, err)

	// Create temporary directory structure
	tmpDir := t.TempDir()
	webDist := filepath.Join(tmpDir, "web", "dist")
	assetsDir := filepath.Join(webDist, "assets")
	err = os.MkdirAll(assetsDir, 0o755)
	assert.NoError(t, err)

	indexPath := filepath.Join(webDist, IndexHTML)
	err = os.WriteFile(indexPath, []byte("<html>SPA</html>"), 0o644)
	assert.NoError(t, err)

	// Change to tmpDir temporarily
	err = os.Chdir(tmpDir)
	assert.NoError(t, err)
	defer func() { _ = os.Chdir(cwd) }()

	// Set up config
	testConfig := &config.Config{
		Frontend: config.FrontendConfig{
			DistDir: "dist",
		},
		Logging: config.LoggingConfig{
			Level:   "info",
			Console: config.ConsoleConfig{Enabled: true, Format: "text", Level: "info"},
		},
	}
	config.SetConfig(testConfig)
	common.InitLogger(&testConfig.Logging)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	setupStaticFiles(r)

	// Test NoRoute handler for non-API path (SPA routing)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/some/spa/route", nil)
	r.ServeHTTP(w, req)

	// SPA should serve index.html for non-API routes
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "SPA")
}
