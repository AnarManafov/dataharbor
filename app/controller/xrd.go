package controller

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"go-hep.org/x/hep/xrootd/xrdfs"

	"github.com/AnarManafov/dataharbor/app/common"
	"github.com/AnarManafov/dataharbor/app/config"
	"github.com/AnarManafov/dataharbor/app/middleware"
	"github.com/AnarManafov/dataharbor/app/request"
	"github.com/AnarManafov/dataharbor/app/response"
)

// StreamingConfig defines buffer size for file streaming
const (
	// BufferSize optimized for high-throughput streaming
	// 512KB provides better network efficiency for large file transfers
	BufferSize = 512 * 1024

	// FlushInterval controls how often we flush the response
	// Flush every 2MB to balance responsiveness with performance
	FlushInterval = 2 * 1024 * 1024

	// Pagination defaults balance user experience with server performance constraints
	defaultPageSize uint32 = 500 // Minimizes round trips while preventing server overload
	minPageSize     uint32 = 5   // Prevents API abuse that could degrade service quality
)

// Download slot management prevents resource exhaustion from concurrent downloads
// Global state is necessary because XRootD connections are expensive resources
// that must be limited across the entire application instance to prevent server overload
var (
	userDownloadSlots = make(map[string]bool)
	downloadSlotMutex sync.Mutex
)

// DownloadFile streams files directly from XRootD using native client
// Single-client approach ensures reliability before adding complexity like connection pooling
func DownloadFile(c *gin.Context) {
	filePath := c.Query("path")
	if filePath == "" {
		response.Error(c, http.StatusBadRequest, "File path parameter is required")
		return
	}

	// Validate file path to prevent directory traversal attacks
	if err := validateFilePath(filePath); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid file path")
		return
	}

	// Rate limiting prevents server resource exhaustion from concurrent downloads
	// Get user token for both authentication and rate limiting
	var userToken string
	if token, exists := middleware.GetUserToken(c); exists {
		userToken = token
	}

	// TODO: TEMPORARILY DISABLED - Rate limiting needs investigation
	// The slot management logic appears to have issues where slots are not being released properly
	// when downloads complete, causing subsequent downloads to be blocked incorrectly.
	// This needs to be investigated and fixed before re-enabling.

	// Enforce one download per user to prevent resource abuse
	// if !acquireDownloadSlot(c) {
	// 	response.Error(c, http.StatusTooManyRequests, "Download already in progress. Please wait for current download to complete.")
	// 	return
	// }
	// Note: We release the slot explicitly after streaming completes

	// Use simple client
	simpleClient := common.GetXRDClient()

	// Get filesystem interface using simple client
	ctx := context.Background()
	fs, cleanup, err := simpleClient.GetFileSystem(ctx, userToken)
	if err != nil {
		common.GetLogger().Error("Failed to get filesystem client", "error", err)
		// releaseDownloadSlot(c) // Release slot on error - DISABLED while rate limiting is disabled
		response.Error(c, http.StatusInternalServerError, "Failed to connect to storage")
		return
	}
	defer cleanup()

	// Get file metadata first
	fileInfo, err := fs.Stat(ctx, filePath)
	if err != nil {
		common.GetLogger().Error("Failed to get file info", "error", err, "path", filePath)
		// releaseDownloadSlot(c) // Release slot on error - DISABLED while rate limiting is disabled
		response.Error(c, http.StatusNotFound, "File not found or inaccessible")
		return
	}

	// Log download attempt with unique ID for tracking
	downloadID := fmt.Sprintf("dl_%d", time.Now().UnixNano())
	common.GetLogger().Info("Starting file download",
		"downloadID", downloadID,
		"path", filePath,
		"size", fileInfo.Size(),
		"user", maskToken(userToken))

	// Track download start time for speed calculation
	downloadStartTime := time.Now()
	// Set response headers for browser download
	filename := filepath.Base(filePath)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", sanitizeFilename(filename)))
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")

	// Set the content length - this is actually important for browsers to show progress
	c.Header("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// Add Accept-Ranges header for future resume download support
	c.Header("Accept-Ranges", "bytes")

	// Set status and start streaming immediately
	c.Status(http.StatusOK)

	// Force headers to be sent to trigger download dialog
	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}

	// Start streaming file using native client
	streamErr := streamFileSimple(c, fs, filePath, userToken, downloadStartTime, fileInfo.Size(), downloadID)

	// Release the download slot immediately after streaming, regardless of success/failure
	// releaseDownloadSlot(c) // DISABLED while rate limiting is disabled
	// common.GetLogger().Info("Download slot released after streaming", "downloadID", downloadID, "user", maskToken(userToken))

	if streamErr != nil {
		common.GetLogger().Error("Failed to stream file", "downloadID", downloadID, "error", streamErr, "path", filePath)
		// Cannot send error response here as headers are already sent
		return
	}

	// Log successful completion
	common.GetLogger().Info("Download completed successfully", "downloadID", downloadID, "path", filePath, "user", maskToken(userToken))
}

// streamFileSimple implements basic streaming without pooling or parallelism
// This approach prioritizes reliability and debugging over performance optimization
func streamFileSimple(c *gin.Context, fs xrdfs.FileSystem, filePath string, userToken string, startTime time.Time, fileSize int64, downloadID string) error {
	// Use request context with longer timeout for large file downloads
	// Calculate timeout based on file size: minimum 5 minutes, plus 1 minute per GB
	timeoutMinutes := 5 + int(fileSize/(1024*1024*1024)) // 5 min base + 1 min per GB
	if timeoutMinutes < 30 {
		timeoutMinutes = 30 // At least 30 minutes for any download
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(timeoutMinutes)*time.Minute)
	defer cancel()

	common.GetLogger().Info("Opening file with simple approach", "downloadID", downloadID, "path", filePath)

	// Multiple open modes attempted because XRootD file permissions can vary by installation
	// OpenModeOtherRead tried first as it matches go-hep library test examples
	file, err := fs.Open(ctx, filePath, xrdfs.OpenModeOtherRead, xrdfs.OpenOptionsNone)
	if err != nil {
		common.GetLogger().Warn("Failed with OpenModeOtherRead, trying OpenModeOwnerRead", "error", err)
		// Create a new context for the retry attempt
		retryCtx, retryCancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer retryCancel()

		// Fallback needed because some XRootD setups require owner-level permissions
		file, err = fs.Open(retryCtx, filePath, xrdfs.OpenModeOwnerRead, xrdfs.OpenOptionsNone)
		if err != nil {
			common.GetLogger().Error("Failed to open file with any mode", "error", err, "path", filePath)
			return fmt.Errorf("failed to open file: %w", err)
		}
	}

	common.GetLogger().Info("File opened successfully", "downloadID", downloadID, "path", filePath)

	defer func() {
		if closeErr := file.Close(ctx); closeErr != nil {
			common.GetLogger().Error("Failed to close file", "error", closeErr, "path", filePath)
		}
	}()

	// Optimized buffer size for better streaming performance
	buffer := make([]byte, BufferSize) // 512KB for improved throughput
	offset := int64(0)
	totalRead := int64(0)
	lastFlushTime := time.Now()

	common.GetLogger().Info("Starting file streaming", "downloadID", downloadID, "path", filePath)

	// Log progress every 100MB for large files
	progressLogInterval := int64(100 * 1024 * 1024) // 100MB
	nextProgressLog := progressLogInterval

	// Send the first chunk immediately to kickstart browser progress
	n, err := file.ReadAt(buffer, offset)
	if n > 0 {
		// Write first chunk immediately
		if _, writeErr := c.Writer.Write(buffer[:n]); writeErr != nil {
			return fmt.Errorf("failed to write initial chunk to client: %w", writeErr)
		}

		// Flush immediately to start progress indication
		if flusher, ok := c.Writer.(http.Flusher); ok {
			flusher.Flush()
		}

		offset += int64(n)
		totalRead += int64(n)
	}

	if err != nil && err != io.EOF {
		common.GetLogger().Error("Initial read error", "error", err, "path", filePath)
		return fmt.Errorf("failed to read initial chunk from file: %w", err)
	}

	// Continue with the rest of the file
	for err != io.EOF {
		// ReadAt provides consistent behavior compared to Read() for XRootD protocol
		n, err := file.ReadAt(buffer, offset)

		if n > 0 {
			// Write data immediately
			if _, writeErr := c.Writer.Write(buffer[:n]); writeErr != nil {
				return fmt.Errorf("failed to write to client: %w", writeErr)
			}

			offset += int64(n)
			totalRead += int64(n)

			// Adaptive flushing based on data rate and time
			shouldFlush := false

			// Flush every 2MB (for high-speed connections)
			if totalRead%FlushInterval == 0 {
				shouldFlush = true
			}

			// Also flush every 500ms (for slow connections)
			timeSinceLastFlush := time.Since(lastFlushTime)
			if timeSinceLastFlush >= 500*time.Millisecond {
				shouldFlush = true
			}

			if shouldFlush {
				if flusher, ok := c.Writer.(http.Flusher); ok {
					flusher.Flush()
					lastFlushTime = time.Now()
				}
			}

			// Log progress for large files
			if totalRead >= nextProgressLog {
				common.GetLogger().Info("Download progress",
					"path", filePath,
					"bytesTransferred", totalRead,
					"MB", totalRead/(1024*1024))
				nextProgressLog += progressLogInterval
			}
		}

		// Handle read completion and errors
		if err != nil {
			if err == io.EOF {
				// Check if we've read all expected data
				if totalRead == fileSize {
					common.GetLogger().Info("File streaming completed successfully", "path", filePath, "totalRead", totalRead, "fileSize", fileSize)
				} else {
					common.GetLogger().Info("File streaming completed (partial)", "path", filePath, "totalRead", totalRead, "fileSize", fileSize)
				}
				break
			}
			common.GetLogger().Error("Read error", "error", err, "path", filePath, "offset", offset)
			return fmt.Errorf("failed to read from file: %w", err)
		}

		// Client disconnect detection prevents wasted server resources
		// Only check for disconnect if we haven't reached the expected file size
		if totalRead < fileSize {
			select {
			case <-c.Request.Context().Done():
				common.GetLogger().Warn("Client disconnected during download", "path", filePath, "totalRead", totalRead, "fileSize", fileSize)
				return fmt.Errorf("client disconnected during download")
			default:
				// Continue streaming
			}
		}
	}

	// Final flush to ensure all data is sent
	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}

	// Calculate download speed and log completion statistics
	downloadDuration := time.Since(startTime)
	downloadSpeedBytesPerSec := float64(totalRead) / downloadDuration.Seconds()
	downloadSpeedMBPerSec := downloadSpeedBytesPerSec / (1024 * 1024)

	// Determine if download was complete
	isComplete := totalRead == fileSize
	completionStatus := "COMPLETE"
	if !isComplete {
		completionStatus = "PARTIAL"
	}

	common.GetLogger().Info("File download finished",
		"downloadID", downloadID,
		"path", filePath,
		"user", maskToken(userToken),
		"status", completionStatus,
		"totalBytes", totalRead,
		"expectedBytes", fileSize,
		"duration", downloadDuration.String(),
		"speedMBps", fmt.Sprintf("%.2f", downloadSpeedMBPerSec),
		"speedBytesPerSec", fmt.Sprintf("%.0f", downloadSpeedBytesPerSec))

	return nil
}

// ListDirectory uses native XRootD API instead of command-line tools
// Direct API calls provide better error handling and avoid shell command overhead
func ListDirectory(c *gin.Context) {
	query := c.Request.URL.Query().Get("dir")
	if query == "" {
		response.Error(c, http.StatusBadRequest, "Directory parameter is required")
		return
	}

	// Use simple client
	simpleClient := common.GetXRDClient()

	// Get user token for authentication
	var authToken string
	if token, exists := middleware.GetUserToken(c); exists {
		authToken = token
	}

	// List directory contents with reasonable timeout - simple approach like working tests
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // Back to 1 minute - tests showed ~3ms
	defer cancel()

	entries, err := simpleClient.ListDirectory(ctx, query, authToken)
	if err != nil {
		common.GetLogger().Error("Failed to list directory", "error", err, "path", query)
		response.Error(c, http.StatusInternalServerError, "Failed to list directory")
		return
	}

	// Convert native entries to our API format
	apiEntries := make([]map[string]interface{}, 0, len(entries))
	for _, entry := range entries {
		apiEntry := map[string]interface{}{
			"name":  entry.Name(),
			"isDir": entry.IsDir(),
			"size":  entry.Size(),
			"mtime": entry.ModTime().Unix(),
		}
		apiEntries = append(apiEntries, apiEntry)
	}

	response.JSON(c, http.StatusOK, gin.H{
		"directory": query,
		"entries":   apiEntries,
	})
}

// GetFileInfo retrieves file metadata using native client
func GetFileInfo(c *gin.Context) {
	filePath := c.Query("path")
	if filePath == "" {
		response.Error(c, http.StatusBadRequest, "File path parameter is required")
		return
	}

	// Use simple client
	simpleClient := common.GetXRDClient()

	// Get user token for authentication
	var authToken string
	if token, exists := middleware.GetUserToken(c); exists {
		authToken = token
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fs, cleanup, err := simpleClient.GetFileSystem(ctx, authToken)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to connect to storage")
		return
	}
	defer cleanup()

	fileInfo, err := fs.Stat(ctx, filePath)
	if err != nil {
		response.Error(c, http.StatusNotFound, "File not found")
		return
	}

	response.JSON(c, http.StatusOK, gin.H{
		"name":  fileInfo.Name(),
		"size":  fileInfo.Size(),
		"mtime": fileInfo.ModTime().Unix(),
		"isDir": fileInfo.IsDir(),
	})
}

// FetchInitialDir returns the configured initial directory
func FetchInitialDir(c *gin.Context) {
	cfg := config.GetConfig()
	response.Success(c, cfg.XRD.InitialDir)
}

// FetchHostName returns the XRootD server hostname
func FetchHostName(c *gin.Context) {
	cfg := config.GetConfig()
	response.Success(c, cfg.XRD.Host)
}

// FetchDirItemsByPage handles paginated directory listing
func FetchDirItemsByPage(c *gin.Context) {
	var req request.DirectoryItemsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithErr(c, *response.SystemErr(err))
		return
	}

	// Validate the page number when pagination is enabled
	if req.Page < 1 {
		response.FailWithErr(c, *response.SystemErr(fmt.Errorf("invalid page number")))
		return
	}

	// Validate the directory path
	dirPath := req.Path
	if len(dirPath) == 0 {
		response.FailWithErr(c, *response.SystemErr(fmt.Errorf("empty directory path to list")))
		return
	}

	// Use simple client
	simpleClient := common.GetXRDClient()

	// Get user token for authentication
	var authToken string
	if token, exists := middleware.GetUserToken(c); exists {
		authToken = token
	}

	// List directory contents with reasonable timeout - simple approach like working tests
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // Back to 1 minute - tests showed ~3ms
	defer cancel()

	common.GetLogger().Debug("Starting directory listing: ", dirPath)

	entries, err := simpleClient.ListDirectory(ctx, dirPath, authToken)
	if err != nil {
		common.GetLogger().Error("Failed to list directory", "error", err, "path", dirPath)
		response.FailWithErr(c, *response.SystemErr(err))
		return
	}
	common.GetLogger().Debug("Successfully retrieved directory entries", "count", len(entries), "path", dirPath)

	// Handle empty directory case explicitly
	if len(entries) == 0 {
		common.GetLogger().Info("Directory is empty, returning empty response", "path", dirPath)
		emptyResponse := gin.H{
			"code":               200,
			"items":              []interface{}{}, // Empty array
			"totalItems":         0,
			"pageSize":           req.PageSize,
			"totalPages":         0,
			"totalFileCount":     0,
			"totalFolderCount":   0,
			"cumulativeFileSize": 0,
		}
		c.JSON(http.StatusOK, emptyResponse)
		return
	}

	// Convert to internal format for pagination
	common.GetLogger().Debug("Converting directory entries to internal format", "count", len(entries))
	files := make([]xrdDirEntry, 0, len(entries))
	for i, entry := range entries {
		// Add detailed logging for problematic entries
		if i < 5 || i >= len(entries)-5 { // Log first and last 5 entries
			common.GetLogger().Debug("Processing entry",
				"index", i,
				"name", entry.Name(),
				"size", entry.Size(),
				"isDir", entry.IsDir(),
				"modTime", entry.ModTime())
		}

		// Check for potential data issues
		entryName := entry.Name()

		// CRITICAL: Check for file names that can break XRootD protocol parsing
		if strings.Contains(entryName, "\n") {
			common.GetLogger().Error("Found file with newline in name - this breaks XRootD protocol parsing",
				"index", i,
				"filename", entryName,
				"path", dirPath,
				"suggestion", "File names with newlines are not supported by XRootD protocol")
			// Skip this file to prevent protocol parsing errors
			continue
		}

		if strings.Contains(entryName, "\r") {
			common.GetLogger().Warn("Found file with carriage return in name - potential parsing issue",
				"index", i,
				"filename", entryName,
				"path", dirPath)
		}

		if !utf8.ValidString(entryName) {
			common.GetLogger().Warn("Invalid UTF-8 sequence in entry name", "index", i, "name", entryName)
			entryName = strings.ToValidUTF8(entryName, "")
		}

		if len(entryName) == 0 {
			common.GetLogger().Warn("Found entry with empty name", "index", i)
			continue // Skip entries with empty names
		}

		// Validate size value
		entrySize := entry.Size()
		if entrySize < 0 {
			common.GetLogger().Warn("Found entry with negative size", "name", entryName, "size", entrySize)
			entrySize = 0 // Normalize negative sizes
		}

		// Validate modification time
		modTime := entry.ModTime()
		if modTime.IsZero() {
			common.GetLogger().Warn("Found entry with zero modification time", "name", entryName)
			modTime = time.Now() // Use current time as fallback
		}

		files = append(files, xrdDirEntry{
			name:  entryName,
			dt:    modTime,
			size:  uint64(entrySize),
			isDir: entry.IsDir(),
		})
	}
	common.GetLogger().Debug("Completed conversion to internal format", "finalCount", len(files))

	common.GetLogger().Debug("Fetched items from directory", "count", len(files), "path", dirPath)

	// Enforce minimum page size to prevent performance issues
	pageSize := req.PageSize
	if pageSize < minPageSize {
		pageSize = minPageSize
	}

	totalItems := uint32(len(files))
	totalPages := (totalItems + pageSize - 1) / pageSize // Ceiling division ensures partial pages are counted

	common.GetLogger().Debug("Pagination info", "page", req.Page, "pageSize", pageSize, "totalItems", totalItems, "totalPages", totalPages)

	if req.Page > uint32(totalPages) {
		response.FailWithErr(c, *response.SystemErr(fmt.Errorf("page number out of range")))
		return
	}

	// Calculate slice boundaries based on pagination settings
	startIndex := (req.Page - 1) * pageSize
	endIndex := min(startIndex+pageSize, totalItems)

	var items []response.DirectoryItemResponse
	var totalFileCount, totalFolderCount, cumulativeFileSize uint64

	// Calculate totals for ALL items first (efficient single pass)
	common.GetLogger().Debug("Calculating statistics for all items", "totalItems", len(files))
	for _, d := range files {
		if d.isDir {
			totalFolderCount++
		} else {
			totalFileCount++
			cumulativeFileSize += d.size
		}
	}
	common.GetLogger().Debug("Statistics calculated",
		"totalFileCount", totalFileCount,
		"totalFolderCount", totalFolderCount,
		"cumulativeFileSize", cumulativeFileSize)

	// Process only visible items for the current page
	common.GetLogger().Debug("Processing visible items for page",
		"startIndex", startIndex,
		"endIndex", endIndex,
		"itemsToProcess", endIndex-startIndex)
	for i, d := range files[startIndex:endIndex] {
		// Sanitize file names consistently using our comprehensive sanitization function
		sanitizedName := sanitizeFilename(d.name)

		// Format time safely
		var timeStr string
		if d.dt.IsZero() {
			timeStr = time.Now().Format("2006-01-02 15:04:05")
		} else {
			timeStr = d.dt.Format("2006-01-02 15:04:05")
		}

		item := response.DirectoryItemResponse{
			Name:     sanitizedName,
			DateTime: timeStr,
			Size:     d.size,
		}
		if d.isDir {
			item.Type = "dir"
		} else {
			item.Type = "file"
		}
		items = append(items, item)

		// Log first few items for debugging
		if i < 3 {
			common.GetLogger().Debug("Processed page item",
				"index", i,
				"name", item.Name,
				"type", item.Type,
				"size", item.Size)
		}
	}
	common.GetLogger().Debug("Completed processing visible items", "processedCount", len(items))

	// Return paginated response
	common.GetLogger().Debug("Preparing JSON response",
		"totalItems", totalItems,
		"pageSize", pageSize,
		"totalPages", totalPages,
		"itemsInResponse", len(items))

	response := gin.H{
		"code":               200,
		"items":              items,
		"totalItems":         totalItems,
		"pageSize":           pageSize,
		"totalPages":         totalPages,
		"totalFileCount":     totalFileCount,
		"totalFolderCount":   totalFolderCount,
		"cumulativeFileSize": cumulativeFileSize,
	}

	common.GetLogger().Debug("Sending JSON response", "responseKeys", len(response))
	c.JSON(http.StatusOK, response)
}

// GetInitialDirectory returns configurable initial directory with possible user customization
// Allows for user-specific starting points based on authentication claims
// GetInitialDirectory provides user-specific starting directories
// Allows for personalized file browsing experience based on authentication claims
func GetInitialDirectory(c *gin.Context) {
	claims, _ := middleware.GetUserClaims(c)

	// Root directory as default avoids assumptions about user permissions
	initialDir := "/"

	if claims != nil {
		// User-specific directories could enhance security and user experience
		if sub, ok := claims["sub"].(string); ok {
			common.GetLogger().Info("User accessing initial directory", "subject", sub)
			// Future enhancement: custom per-user directories
			// initialDir = "/users/" + username
		}
	}

	response.JSON(c, http.StatusOK, gin.H{
		"directory": initialDir,
	})
}

// GetHostName returns the XRootD server hostname for client configuration
func GetHostName(c *gin.Context) {
	cfg := config.GetConfig()
	response.JSON(c, http.StatusOK, gin.H{
		"hostname": cfg.XRD.Host,
	})
}

// GetDownloadSlotStatus returns the current download slot status for debugging
func GetDownloadSlotStatus(c *gin.Context) {
	downloadSlotMutex.Lock()
	defer downloadSlotMutex.Unlock()

	// Get current user key
	userKey := getUserKey(c)

	// Build response with slot information
	slotInfo := make(map[string]interface{})
	slotInfo["userKey"] = userKey
	slotInfo["hasActiveSlot"] = userDownloadSlots[userKey]
	slotInfo["totalActiveSlots"] = len(userDownloadSlots)

	// List all active slots (for debugging)
	activeSlots := make([]string, 0, len(userDownloadSlots))
	for key := range userDownloadSlots {
		activeSlots = append(activeSlots, key)
	}
	slotInfo["activeSlots"] = activeSlots

	response.JSON(c, http.StatusOK, slotInfo)
}

// ForceReleaseDownloadSlot forcefully releases a download slot for the current user
// This is a debugging/admin endpoint to help with stuck slots
func ForceReleaseDownloadSlot(c *gin.Context) {
	downloadSlotMutex.Lock()
	defer downloadSlotMutex.Unlock()

	userKey := getUserKey(c)

	if !userDownloadSlots[userKey] {
		response.JSON(c, http.StatusOK, gin.H{
			"message": "No active download slot found for user",
			"userKey": userKey,
		})
		return
	}

	// Force release the slot
	delete(userDownloadSlots, userKey)

	common.GetLogger().Warn("Forcefully released download slot", "userKey", userKey)

	response.JSON(c, http.StatusOK, gin.H{
		"message":        "Download slot forcefully released",
		"userKey":        userKey,
		"remainingSlots": len(userDownloadSlots),
	})
}

// Utility types and functions

// xrdDirEntry represents a directory entry
type xrdDirEntry struct {
	name  string
	dt    time.Time
	size  uint64
	isDir bool
}

// validateFilePath prevents path traversal attacks and ensures safe file access
// Strict validation is critical since paths are passed directly to XRootD
func validateFilePath(filePath string) error {
	// Directory traversal protection prevents access outside intended directories
	if strings.Contains(filePath, "..") {
		return fmt.Errorf("path contains directory traversal")
	}

	// Absolute paths prevent ambiguity in file resolution
	if !strings.HasPrefix(filePath, "/") {
		return fmt.Errorf("path must be absolute")
	}

	// Null bytes and control characters can cause protocol-level issues
	if strings.ContainsAny(filePath, "\x00\r\n") {
		return fmt.Errorf("path contains invalid characters")
	}

	return nil
}

// acquireDownloadSlot enforces one-download-per-user limit to prevent resource exhaustion
func acquireDownloadSlot(c *gin.Context) bool {
	downloadSlotMutex.Lock()
	defer downloadSlotMutex.Unlock()

	// Use stable user identifier from JWT claims instead of token hash
	userKey := getUserKey(c)
	common.GetLogger().Debug("Attempting to acquire download slot", "userKey", userKey, "activeSlots", len(userDownloadSlots))

	if userDownloadSlots[userKey] {
		common.GetLogger().Info("Download slot already in use", "userKey", userKey, "activeSlots", len(userDownloadSlots))
		return false // User already has an active download
	}

	userDownloadSlots[userKey] = true
	common.GetLogger().Info("Download slot acquired", "userKey", userKey, "activeSlots", len(userDownloadSlots))
	return true
}

// releaseDownloadSlot frees up download capacity for the user
func releaseDownloadSlot(c *gin.Context) {
	downloadSlotMutex.Lock()
	defer downloadSlotMutex.Unlock()

	userKey := getUserKey(c)
	delete(userDownloadSlots, userKey)
	common.GetLogger().Info("Download slot released", "userKey", userKey, "activeSlots", len(userDownloadSlots))
}

// getUserKey creates stable identifier using user's JWT subject claim
// This ensures consistent user identification across token refreshes
func getUserKey(c *gin.Context) string {
	// Get user claims from JWT token
	claims, ok := middleware.GetUserClaims(c)
	if ok && claims != nil {
		// Use the "sub" (subject) claim as stable user identifier
		if sub, ok := claims["sub"].(string); ok && sub != "" {
			return fmt.Sprintf("user_%s", sub)
		}
	}

	// Fallback to using token hash if no sub claim available
	if userToken, ok := middleware.GetUserToken(c); ok {
		hash := sha256.Sum256([]byte(userToken))
		return fmt.Sprintf("user_%x", hash[:8])
	}

	return "anonymous"
}

// maskToken protects sensitive data in logs while maintaining debugging capability
func maskToken(token string) string {
	if token == "" {
		return "anonymous"
	}
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

// sanitizeFilename removes or replaces dangerous characters in filenames
// Also ensures valid UTF-8 encoding to prevent protocol parsing issues
func sanitizeFilename(filename string) string {
	// First ensure UTF-8 validity
	if !utf8.ValidString(filename) {
		var buf strings.Builder
		for _, r := range filename {
			if r == utf8.RuneError {
				buf.WriteRune('_') // Replace invalid characters with underscore
			} else {
				buf.WriteRune(r)
			}
		}
		filename = buf.String()
	}

	// Remove path separators and other dangerous characters
	sanitized := strings.ReplaceAll(filename, "/", "_")
	sanitized = strings.ReplaceAll(sanitized, "\\", "_")
	sanitized = strings.ReplaceAll(sanitized, "..", "_")
	sanitized = strings.ReplaceAll(sanitized, "\x00", "")

	// Also sanitize other potentially problematic characters
	sanitized = strings.ReplaceAll(sanitized, "\n", "_")
	sanitized = strings.ReplaceAll(sanitized, "\r", "_")

	// Limit length
	if len(sanitized) > 255 {
		sanitized = sanitized[:255]
	}

	return sanitized
}

// min returns the smaller of two uint32 values
func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}
