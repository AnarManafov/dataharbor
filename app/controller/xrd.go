package controller

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/AnarManafov/dataharbor/app/common"
	"github.com/AnarManafov/dataharbor/app/config"
	"github.com/AnarManafov/dataharbor/app/middleware"
	"github.com/AnarManafov/dataharbor/app/response"
)

type xrdDirEntry struct {
	name  string
	dt    time.Time
	size  uint64
	isDir bool
}

type cacheEntry struct {
	data      []xrdDirEntry
	timestamp time.Time
}

var (
	cache      = make(map[string]cacheEntry)
	cacheMutex sync.Mutex

	// 60 minute TTL balances performance vs. freshness for directory listings
	// in environments where content changes infrequently
	cacheTTL = 60 * time.Minute
)

// Function types for dependency injection in unit tests
type execCommandFunc func(ctx context.Context, name string, arg ...string) *exec.Cmd
type RunXrdFsFunc func(execCmd execCommandFunc, arg ...string) (string, error)

// Retrieves directory data from cache if available and not expired
// to reduce network overhead and server load
func getCachedData(key string) ([]xrdDirEntry, bool) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	entry, exists := cache[key]
	if !exists || time.Since(entry.timestamp) > cacheTTL {
		return nil, false
	}
	return entry.data, true
}

// Updates cache with fresh directory data to improve subsequent request performance
func setCachedData(key string, data []xrdDirEntry) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	cache[key] = cacheEntry{
		data:      data,
		timestamp: time.Now(),
	}
}

// Executes a command with optional timeout to prevent blocking on hung processes
func runCommand(execCmd execCommandFunc, timeout uint, name string, arg ...string) (string, error) {
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
		defer cancel()
	}

	cmd := execCmd(ctx, name, arg...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// Variables instead of direct functions to enable mocking in tests
var (
	RunXrdFs       RunXrdFsFunc = runXrdFsImpl
	stageFileLocal              = stageFileLocalImpl
)

// Interfaces with the XRootD server by executing the xrdfs command
// which provides file and directory operations against XRootD storage
func runXrdFsImpl(execCmd execCommandFunc, arg ...string) (string, error) {
	cfg := config.GetConfig()
	common.Logger.Info("RunXrdFs: ", arg)
	return runCommand(execCmd, cfg.XRD.ProcessTimeout, path.Join(cfg.XRD.XrdClientBinPath, "xrdfs"), arg...)
}

// Creates a temporary staging area for XRD files before download
// Avoids filename collisions by using unique temporary directories
func stageFileLocalImpl(host string, port uint, file string) (string, error) {
	cfg := config.GetConfig()
	srdAddr := host + ":" + strconv.FormatUint(uint64(port), 10)
	tmpDir, err := os.MkdirTemp(cfg.XRD.StagingPath, cfg.XRD.StagingTmpDirPrefix)
	if err != nil {
		return "", err
	}
	stagedFilePath := path.Join(tmpDir, path.Base(file))

	// Copy file from XRootD server to local staging area for web access
	err = RunXrdCp(srdAddr, file, stagedFilePath)
	if err != nil {
		return "", err
	}

	return stagedFilePath, nil
}

// Copies files from XRootD server to local filesystem using xrdcp utility
// which handles authentication and transfer protocol details
func RunXrdCp(xrdAddr string, src string, dest string) error {
	cfg := config.GetConfig()
	common.Logger.Info("XRD: Staging " + src + " to " + dest)
	_, err := runCommand(exec.CommandContext, cfg.XRD.ProcessTimeout, path.Join(cfg.XRD.XrdClientBinPath, "xrdcp"), "--force", "xroot://"+xrdAddr+"/"+src, dest)
	return err
}

// Lists directory content with caching to reduce load on XRootD server
// Format of XRootD output is parsed into structured data for UI consumption
func ReadDir(xrdFS RunXrdFsFunc, host string, port uint, dir string) ([]xrdDirEntry, error) {
	srdAddr := host + ":" + strconv.FormatUint(uint64(port), 10)
	cacheKey := srdAddr + ":" + dir

	// Check cache first to reduce network overhead
	if data, found := getCachedData(cacheKey); found {
		return data, nil
	}

	// Fetch from server when cache misses or expired
	output, err := xrdFS(exec.CommandContext, srdAddr, "ls", "-l", dir)
	if err != nil {
		return nil, err
	}

	var retVal []xrdDirEntry
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		// Parse "drwxr-xr-x username staff 224 2024-02-14 12:14:47 /path/filename" format
		// which is returned by xrdfs ls -l command

		pattern := `\s+`
		regex := regexp.MustCompile(pattern)
		columns := regex.Split(scanner.Text(), 7)

		var item xrdDirEntry
		item.name = path.Base(columns[6])
		item.isDir = columns[0][0] == 'd'
		// Parse timestamp in format used by XRootD server
		const layoutTime = "2006-01-02 15:04:05"
		tt, err := time.Parse(layoutTime, columns[4]+" "+columns[5])
		if err == nil {
			item.dt = tt
		}
		// Parse file size
		s, err := strconv.ParseUint(columns[3], 10, 64)
		if err == nil {
			item.size = s
		}
		retVal = append(retVal, item)
	}

	// Cache results for future requests to improve performance
	setCachedData(cacheKey, retVal)

	return retVal, nil
}

// HTTP handler for directory listing that connects XRootD with web UI
// Supports authenticated requests with user token for authorization
func ListDirectory(c *gin.Context) {
	query := c.Request.URL.Query().Get("dir")
	if query == "" {
		response.Error(c, http.StatusBadRequest, "Directory parameter is required")
		return
	}

	xrdClient := common.GetXRDClient()

	// Apply user-specific authorization if available
	if token, exists := middleware.GetUserToken(c); exists {
		xrdClient.SetUserToken(token)
	}

	// URL-encode to handle special characters in XRootD paths
	encodedQuery := url.QueryEscape(query)

	// Build URL with auth parameters for XRootD server
	xrdURL, err := xrdClient.BuildXRDURLWithCGI(encodedQuery, map[string]string{})
	if err != nil {
		xrdClient.Logger.Error("Failed to build XRD URL", "error", err)
		response.Error(c, http.StatusInternalServerError, "Failed to build XRD URL")
		return
	}

	// Use custom TLS settings if configured
	client, err := xrdClient.CreateHTTPClient()
	if err != nil {
		xrdClient.Logger.Error("Failed to create HTTP client", "error", err)
		response.Error(c, http.StatusInternalServerError, "Failed to create HTTP client")
		return
	}

	resp, err := client.Get(xrdURL)
	if err != nil {
		xrdClient.Logger.Error("Failed to list directory", "error", err, "url", xrdURL)
		response.Error(c, http.StatusInternalServerError, "Failed to list directory")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		xrdClient.Logger.Error("XRD server returned error", "status", resp.Status, "body", string(bodyBytes))
		response.Error(c, resp.StatusCode, "Failed to list directory")
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		xrdClient.Logger.Error("Failed to read response body", "error", err)
		response.Error(c, http.StatusInternalServerError, "Failed to read response")
		return
	}

	// Convert raw directory listing to structured format
	entries, err := parseDirectoryListing(bodyBytes, query)
	if err != nil {
		xrdClient.Logger.Error("Failed to parse directory listing", "error", err)
		response.Error(c, http.StatusInternalServerError, "Failed to parse directory listing")
		return
	}

	response.JSON(c, http.StatusOK, gin.H{
		"directory": query,
		"entries":   entries,
	})
}

// Returns configurable initial directory with possible user customization
// Allows for user-specific starting points based on authentication claims
func GetInitialDirectory(c *gin.Context) {
	claims, _ := middleware.GetUserClaims(c)

	// Default starting point for file browsing
	initialDir := "/"

	if claims != nil {
		// Custom directories can be assigned based on user identity
		if sub, ok := claims["sub"].(string); ok {
			xrdClient := common.GetXRDClient()
			xrdClient.Logger.Info("User accessing initial directory", "subject", sub)

			// Example of how custom directories could be set based on user claims
			// initialDir = "/users/" + username
		}
	}

	response.JSON(c, http.StatusOK, gin.H{
		"directory": initialDir,
	})
}

// StageFileRequest represents the request body for file staging
type StageFileRequest struct {
	FilePath string `json:"filePath" binding:"required"`
}

// Initiates file staging to prepare data for download
// Using XRootD's "prepare" operation to ensure file is ready to access
func StageFile(c *gin.Context) {
	var req StageFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request")
		return
	}

	if req.FilePath == "" {
		response.Error(c, http.StatusBadRequest, "File path is required")
		return
	}

	xrdClient := common.GetXRDClient()

	if token, exists := middleware.GetUserToken(c); exists {
		xrdClient.SetUserToken(token)
	}

	encodedPath := url.QueryEscape(req.FilePath)

	// Use prepare operation to ensure file is staged from tape or cache
	xrdURL, err := xrdClient.BuildXRDURLWithCGI(encodedPath, map[string]string{"op": "prepare"})
	if err != nil {
		xrdClient.Logger.Error("Failed to build XRD URL", "error", err)
		response.Error(c, http.StatusInternalServerError, "Failed to build XRD URL")
		return
	}

	client, err := xrdClient.CreateHTTPClient()
	if err != nil {
		xrdClient.Logger.Error("Failed to create HTTP client", "error", err)
		response.Error(c, http.StatusInternalServerError, "Failed to create HTTP client")
		return
	}

	// PUT request triggers the staging process on the XRootD server
	httpReq, err := http.NewRequest(http.MethodPut, xrdURL, bytes.NewBuffer([]byte{}))
	if err != nil {
		xrdClient.Logger.Error("Failed to create stage request", "error", err)
		response.Error(c, http.StatusInternalServerError, "Failed to create stage request")
		return
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		xrdClient.Logger.Error("Failed to stage file", "error", err, "url", xrdURL)
		response.Error(c, http.StatusInternalServerError, "Failed to stage file")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		xrdClient.Logger.Error("XRD server returned error for staging", "status", resp.Status, "body", string(bodyBytes))
		response.Error(c, resp.StatusCode, "Failed to stage file")
		return
	}

	response.JSON(c, http.StatusOK, gin.H{
		"message": "File staged successfully",
		"path":    req.FilePath,
	})
}

// Returns the XRootD server hostname for client configuration
func GetHostName(c *gin.Context) {
	xrdClient := common.GetXRDClient()

	parsedURL, err := url.Parse(xrdClient.URL)
	if err != nil {
		xrdClient.Logger.Error("Failed to parse XRD URL", "error", err)
		response.Error(c, http.StatusInternalServerError, "Failed to parse XRD URL")
		return
	}

	hostname := parsedURL.Hostname()

	response.JSON(c, http.StatusOK, gin.H{
		"hostname": hostname,
	})
}

// Converts XRootD directory listing format to structured data for UI
func parseDirectoryListing(data []byte, dirPath string) ([]map[string]interface{}, error) {
	lines := strings.Split(string(data), "\n")
	entries := make([]map[string]interface{}, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 9 {
			continue
		}

		perms := parts[0]
		isDir := perms[0] == 'd'

		size := parts[4]
		date := parts[5] + " " + parts[6] + " " + parts[7]
		name := strings.Join(parts[8:], " ")

		fullPath := path.Join(dirPath, name)

		entry := map[string]interface{}{
			"name":      name,
			"path":      fullPath,
			"isDir":     isDir,
			"size":      size,
			"dateModif": date,
			"perms":     perms,
		}

		entries = append(entries, entry)
	}

	return entries, nil
}
