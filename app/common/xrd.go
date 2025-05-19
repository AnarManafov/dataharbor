package common

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/AnarManafov/dataharbor/app/config"
	"go-hep.org/x/hep/xrootd"
	"go-hep.org/x/hep/xrootd/xrdfs"
	"go.uber.org/zap"
)

// XRDClient provides a simple, direct XRootD client without connection pooling
// Based on the successful test programs that work reliably
type XRDClient struct {
	address      string
	username     string
	logger       *zap.SugaredLogger
	userRequired bool
}

var (
	xrdClient     *XRDClient
	xrdClientOnce sync.Once
)

// GetXRDNativeClient returns the singleton XRootD client instance
// Kept for backward compatibility with existing controller code
func GetXRDNativeClient() *XRDClient {
	return GetXRDClient()
}

// GetXRDClient returns the singleton XRootD client instance
func GetXRDClient() *XRDClient {
	xrdClientOnce.Do(func() {
		xrdClient = NewXRDClient()
	})
	return xrdClient
}

// NewXRDClient creates a new XRootD client
func NewXRDClient() *XRDClient {
	cfg := config.GetConfig()
	logger := GetLogger()
	address := fmt.Sprintf("%s:%d", cfg.XRD.Host, cfg.XRD.Port)
	username := "dataharbor"
	if cfg.XRD.User != "" {
		username = cfg.XRD.User
	}

	return &XRDClient{
		address:      address,
		username:     username,
		logger:       logger,
		userRequired: cfg.XRD.UserRequired,
	}
}

// createClient creates a new XRootD client - simple and direct like our working tests
func (xc *XRDClient) createClient(ctx context.Context, authToken string) (*xrootd.Client, error) {
	var opts []xrootd.Option

	// Only add auth if required and token is available
	if xc.userRequired && authToken != "" {
		tokenAuth := NewTokenAuth(authToken)
		opts = append(opts, xrootd.WithAuth(tokenAuth))
		xc.logger.Debug("Creating XRootD client with token authentication", "address", xc.address)
	} else if xc.userRequired && authToken == "" {
		return nil, fmt.Errorf("authentication required but no token provided")
	} else {
		xc.logger.Debug("Creating XRootD client without authentication", "address", xc.address)
	}

	client, err := xrootd.NewClient(ctx, xc.address, xc.username, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create XRootD client: %w", err)
	}

	return client, nil
}

// ListDirectory lists directory contents using a fresh client per request
// This matches the approach of our successful test programs
func (xc *XRDClient) ListDirectory(ctx context.Context, dirPath string, authToken string) ([]xrdfs.EntryStat, error) {
	xc.logger.Info("Starting directory listing", "path", dirPath, "server", xc.address)
	start := time.Now()

	// Create a fresh client for this request - no pooling complexity
	client, err := xc.createClient(ctx, authToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			xc.logger.Warn("Error closing client", "error", closeErr)
		}
	}()

	// Get filesystem interface
	fs := client.FS()
	if fs == nil {
		return nil, fmt.Errorf("failed to get filesystem interface")
	}

	// Perform the directory listing - simple and direct
	xc.logger.Debug("Calling fs.Dirlist", "path", dirPath)
	entries, err := fs.Dirlist(ctx, dirPath)
	if err != nil {
		duration := time.Since(start)
		xc.logger.Error("Directory listing failed", "path", dirPath, "duration", duration, "error", err)

		// Check for protocol errors
		if strings.Contains(err.Error(), "wrong response size for the dirlist request") {
			xc.logger.Error("XRootD protocol parsing error - possibly files with newlines in names", "path", dirPath)
		}

		// Check for common XRootD error patterns that might indicate empty directories
		errStr := err.Error()
		if strings.Contains(errStr, "No such file or directory") ||
			strings.Contains(errStr, "directory not found") ||
			strings.Contains(errStr, "path does not exist") {
			// This is a non-existent directory, return the original error
			return nil, fmt.Errorf("failed to list directory %s: %w", dirPath, err)
		}

		// Check if this might be an empty directory that XRootD is reporting as an error
		// Some XRootD configurations return errors for empty directories instead of empty lists
		if strings.Contains(errStr, "empty directory") ||
			strings.Contains(errStr, "no entries") ||
			strings.Contains(errStr, "directory is empty") {
			xc.logger.Info("XRootD reported empty directory as error, treating as empty", "path", dirPath)
			return []xrdfs.EntryStat{}, nil // Return empty slice for empty directory
		}

		return nil, fmt.Errorf("failed to list directory %s: %w", dirPath, err)
	}

	duration := time.Since(start)
	xc.logger.Info("Successfully listed directory", "path", dirPath, "entries", len(entries), "duration", duration)

	// Validate file names for protocol compatibility (but don't filter - let caller decide)
	for i, entry := range entries {
		entryName := entry.Name()
		if strings.Contains(entryName, "\n") {
			xc.logger.Warn("File with newline in name detected", "filename", entryName, "path", dirPath, "index", i)
		}
	}

	return entries, nil
}

// GetFileSystem creates a fresh filesystem client for file operations
func (xc *XRDClient) GetFileSystem(ctx context.Context, authToken string) (xrdfs.FileSystem, func(), error) {
	client, err := xc.createClient(ctx, authToken)
	if err != nil {
		return nil, nil, err
	}

	fs := client.FS()
	if fs == nil {
		client.Close()
		return nil, nil, fmt.Errorf("failed to get filesystem interface")
	}

	cleanup := func() {
		if closeErr := client.Close(); closeErr != nil {
			xc.logger.Warn("Error closing client", "error", closeErr)
		}
	}

	return fs, cleanup, nil
}

// Legacy methods for backward compatibility with existing controller code

// SetUserToken is kept for compatibility but not used in the simplified approach
// Authentication token is now passed directly to methods that need it
func (xc *XRDClient) SetUserToken(token string) {
	// No-op for compatibility - token is passed directly to methods
	xc.logger.Debug("SetUserToken called - token will be passed directly to operations")
}

// GetFileSystem without context and authToken for backward compatibility
// Uses empty auth token - mainly for operations where auth isn't critical
func (xc *XRDClient) GetFileSystemLegacy() (xrdfs.FileSystem, func(), error) {
	ctx := context.Background()
	return xc.GetFileSystem(ctx, "")
}

// ListDirectory without authToken for backward compatibility
func (xc *XRDClient) ListDirectoryLegacy(ctx context.Context, dirPath string) ([]xrdfs.EntryStat, error) {
	return xc.ListDirectory(ctx, dirPath, "")
}
