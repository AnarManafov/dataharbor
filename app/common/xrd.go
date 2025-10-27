package common

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/AnarManafov/dataharbor/app/config"
	"go-hep.org/x/hep/xrootd"
	"go-hep.org/x/hep/xrootd/xrdfs"
	"go.uber.org/zap"
)

// XRootDAuthError represents authentication/authorization failures
type XRootDAuthError struct {
	message string
	cause   error
}

func (e *XRootDAuthError) Error() string {
	return e.message
}

func (e *XRootDAuthError) Unwrap() error {
	return e.cause
}

// IsAuthError checks if an error is an XRootD authentication/authorization error
func IsAuthError(err error) bool {
	var authErr *XRootDAuthError
	return errors.As(err, &authErr)
}

// XRDClient provides a simple, direct XRootD client without connection pooling
// Based on the successful test programs that work reliably
type XRDClient struct {
	address   string
	username  string
	logger    *zap.SugaredLogger
	enableZTN bool
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
		address:   address,
		username:  username,
		logger:    logger,
		enableZTN: cfg.XRD.EnableZTN,
	}
}

// createClient creates a new XRootD client with optional ZTN protocol support
func (xc *XRDClient) createClient(ctx context.Context, authToken string) (*xrootd.Client, error) {
	var opts []xrootd.Option

	if xc.enableZTN {
		// ZTN protocol enabled: Configure TLS + OAuth token authentication
		// Use InsecureSkipVerify for now - can be configured properly with real certs later
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
		}
		opts = append(opts, xrootd.WithTLS(tlsConfig))

		// Set bearer token in environment for ZTN token discovery
		if authToken == "" {
			return nil, fmt.Errorf("ZTN protocol enabled (enable_ztn=true) but no authentication token provided")
		}
		if err := os.Setenv("BEARER_TOKEN", authToken); err != nil {
			return nil, fmt.Errorf("failed to set BEARER_TOKEN environment variable: %w", err)
		}
		xc.logger.Debug("Creating XRootD client with ZTN (TLS + token)", "address", xc.address)
	} else {
		// Plain XRootD protocol: No TLS, no authentication
		xc.logger.Debug("Creating XRootD client with plain protocol (no ZTN)", "address", xc.address)
	}

	client, err := xrootd.NewClient(ctx, xc.address, xc.username, opts...)
	if err != nil {
		// Check if this is an authorization error
		if isAuthorizationError(err) {
			return nil, &XRootDAuthError{
				message: "XRootD authentication failed - user is not authorized to access the storage system",
				cause:   err,
			}
		}
		return nil, fmt.Errorf("failed to create XRootD client: %w", err)
	}

	return client, nil
}

// isAuthorizationError checks if an error indicates authentication/authorization failure
func isAuthorizationError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	authPatterns := []string{
		"authorization",
		"unauthorized",
		"authentication",
		"permission denied",
		"access denied",
		"not authorized",
		"token",
		"credentials",
		"aud",
		"audience",
		"claim verification",
		"scitokens",
	}

	for _, pattern := range authPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
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

		// Check if this is an authorization error
		if isAuthorizationError(err) {
			return nil, &XRootDAuthError{
				message: "Access denied - user is not authorized to access this directory",
				cause:   err,
			}
		}

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
		_ = client.Close()
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
