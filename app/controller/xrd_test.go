package controller

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/AnarManafov/dataharbor/app/config"
	"go-hep.org/x/hep/xrootd/xrdfs"
)

// Helper function to create a mock gin context with user claims
func createMockContext(sub string, token string) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Set user claims in context
	if sub != "" {
		claims := map[string]any{
			"sub": sub,
		}
		c.Set("user_claims", claims)
	}

	// Set access token in context
	if token != "" {
		c.Set("access_token", token)
	}

	return c
}

// Test filename sanitization function
func TestSanitizeFilename(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"normal_file.txt", "normal_file.txt"},
		{"file with spaces.txt", "file with spaces.txt"},
		{"file-with-dashes.txt", "file-with-dashes.txt"},
		{"file.with.dots.txt", "file.with.dots.txt"},
		{"file_123.txt", "file_123.txt"},
		{"file/with/slashes.txt", "file_with_slashes.txt"},
		{"file\\with\\backslashes.txt", "file_with_backslashes.txt"},
		{"file..with..dots.txt", "file_with_dots.txt"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := sanitizeFilename(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// Test sanitizeFilename with special characters
func TestSanitizeFilename_SpecialChars(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "newline in filename",
			input:    "file\nname.txt",
			expected: "file_name.txt",
		},
		{
			name:     "carriage return in filename",
			input:    "file\rname.txt",
			expected: "file_name.txt",
		},
		{
			name:     "null byte in filename",
			input:    "file\x00name.txt",
			expected: "filename.txt",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeFilename(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// Test sanitizeFilename with long filenames
func TestSanitizeFilename_LongFilename(t *testing.T) {
	// Create a filename longer than 255 characters
	var longName strings.Builder
	for range 300 {
		longName.WriteString("a")
	}

	result := sanitizeFilename(longName.String())
	assert.LessOrEqual(t, len(result), 255, "Sanitized filename should be at most 255 characters")
}

// Test sanitizeFilename with invalid UTF-8
func TestSanitizeFilename_InvalidUTF8(t *testing.T) {
	// Create a string with invalid UTF-8 sequence
	// 0xff is an invalid UTF-8 byte
	invalidUTF8 := "test\xfffile.txt"

	result := sanitizeFilename(invalidUTF8)

	// Result should have the invalid byte replaced with underscore
	assert.Contains(t, result, "_")
	assert.Contains(t, result, "file.txt")
}

// Test sanitizeFilename with mixed valid and invalid UTF-8
func TestSanitizeFilename_MixedUTF8(t *testing.T) {
	// Valid UTF-8 string with Unicode characters
	validUTF8 := "tëst_filé.txt"

	result := sanitizeFilename(validUTF8)

	// Should remain unchanged (no dangerous chars)
	assert.Equal(t, validUTF8, result)
}

// Test sanitizeFilename with all dangerous characters
func TestSanitizeFilename_AllDangerous(t *testing.T) {
	dangerous := "file/with\\path..traversal\x00null\nnewline\rcarriage"
	result := sanitizeFilename(dangerous)

	// Should not contain any dangerous characters
	assert.NotContains(t, result, "/")
	assert.NotContains(t, result, "\\")
	assert.NotContains(t, result, "..")
	assert.NotContains(t, result, "\x00")
	assert.NotContains(t, result, "\n")
	assert.NotContains(t, result, "\r")
}

// Test for xrdDirEntry structure
func TestXrdDirEntry(t *testing.T) {
	entry := xrdDirEntry{
		name:  "test_file.txt",
		dt:    time.Now(),
		size:  1024,
		isDir: false,
	}

	assert.Equal(t, "test_file.txt", entry.name)
	assert.Equal(t, uint64(1024), entry.size)
	assert.False(t, entry.isDir)
}

// Test getUserKey function for download slot management
func TestGetUserKey(t *testing.T) {
	// Test anonymous user (no claims)
	c1 := createMockContext("", "")
	assert.Equal(t, "anonymous", getUserKey(c1))

	// Test users with different subject claims
	c2 := createMockContext("user123", "some-token")
	c3 := createMockContext("user456", "another-token")

	key1 := getUserKey(c2)
	key2 := getUserKey(c3)

	// Keys should be different for different users
	assert.NotEqual(t, key1, key2, "Different users should produce different user keys")

	// Keys should be consistent for same user
	c4 := createMockContext("user123", "different-token") // Same user, different token
	assert.Equal(t, key1, getUserKey(c4), "Same user should produce same key even with different token")

	// Keys should start with "user_" prefix
	assert.Contains(t, key1, "user_")
	assert.Contains(t, key2, "user_")

	// Test fallback to token hash when no sub claim
	c5 := createMockContext("", "token123456789")
	c6 := createMockContext("", "different12345")

	key3 := getUserKey(c5)
	key4 := getUserKey(c6)

	assert.NotEqual(t, key3, key4, "Different tokens should produce different keys when no sub claim")
	assert.NotEqual(t, "anonymous", key3, "Should not be anonymous when token is available")
	assert.NotEqual(t, "anonymous", key4, "Should not be anonymous when token is available")
}

// Test download slot acquisition and release
func TestDownloadSlotManagement(t *testing.T) {
	// Clean slate for testing
	userDownloadSlots = make(map[string]bool)

	// Create contexts for different users
	c1 := createMockContext("user1", "token1")
	c2 := createMockContext("user2", "token2")
	c3 := createMockContext("user1", "new-token-after-refresh") // Same user, refreshed token

	// First user should be able to acquire slot
	assert.True(t, acquireDownloadSlot(c1), "First user should acquire slot successfully")

	// Same user should not be able to acquire another slot
	assert.False(t, acquireDownloadSlot(c1), "Same user should not acquire multiple slots")

	// Same user with refreshed token should still not be able to acquire slot
	assert.False(t, acquireDownloadSlot(c3), "Same user with refreshed token should not acquire multiple slots")

	// Different user should be able to acquire slot
	assert.True(t, acquireDownloadSlot(c2), "Different user should acquire slot successfully")

	// Release first user's slot
	releaseDownloadSlot(c1)

	// First user should be able to acquire slot again
	assert.True(t, acquireDownloadSlot(c1), "User should be able to reacquire slot after release")

	// First user with refreshed token should also be able to acquire slot (since they're the same user)
	releaseDownloadSlot(c1)
	assert.True(t, acquireDownloadSlot(c3), "User with refreshed token should be able to acquire slot")

	// Clean up
	releaseDownloadSlot(c3)
	releaseDownloadSlot(c2)

	// Verify slots are cleaned up
	assert.Empty(t, userDownloadSlots, "All slots should be cleaned up")
}

// Test that demonstrates the fix for token refresh issue
func TestTokenRefreshRateLimiting(t *testing.T) {
	// Clean slate for testing
	userDownloadSlots = make(map[string]bool)

	// Simulate the same user with different tokens (token refresh scenario)
	userSub := "a.manafov"
	originalToken := "eyJh...GCPQ"
	refreshedToken := "eyJh...PEpg"

	// Create contexts representing the same user with different tokens
	c1 := createMockContext(userSub, originalToken)
	c2 := createMockContext(userSub, refreshedToken)

	// Verify that both contexts produce the same user key
	key1 := getUserKey(c1)
	key2 := getUserKey(c2)
	assert.Equal(t, key1, key2, "Same user should have same key regardless of token refresh")

	// First download should succeed
	assert.True(t, acquireDownloadSlot(c1), "First download should succeed")

	// Second download with refreshed token should fail (same user)
	assert.False(t, acquireDownloadSlot(c2), "Second download with refreshed token should fail - same user")

	// Release the slot
	releaseDownloadSlot(c1)

	// Now the user with refreshed token should be able to download
	assert.True(t, acquireDownloadSlot(c2), "User with refreshed token should be able to download after releasing slot")

	// Clean up
	releaseDownloadSlot(c2)
	assert.Empty(t, userDownloadSlots, "All slots should be cleaned up")
}

// TestDownloadSlotReleaseAfterCompletion verifies that download slots are properly released
// after a download completes, even when the HTTP request context is cancelled
func TestDownloadSlotReleaseAfterCompletion(t *testing.T) {
	// Create a test HTTP request context
	req, _ := http.NewRequest("GET", "/test", nil)

	// Create a context that will be cancelled to simulate normal HTTP completion
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	// Create a gin context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Set up user claims for stable user identification
	claims := map[string]any{
		"sub": "test.user@example.com",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	c.Set("user_claims", claims)

	// Test that slot can be acquired
	assert.True(t, acquireDownloadSlot(c), "Should be able to acquire download slot")

	// Verify slot is marked as in use
	assert.False(t, acquireDownloadSlot(c), "Should not be able to acquire second slot")

	// Cancel the context to simulate normal HTTP completion
	cancel()

	// Release the slot (this would normally happen via defer)
	releaseDownloadSlot(c)

	// Verify slot is released and can be acquired again
	assert.True(t, acquireDownloadSlot(c), "Should be able to acquire slot after release")

	// Clean up
	releaseDownloadSlot(c)
}

// TestDownloadSlotWithContextCancellation tests the scenario where the HTTP request
// context is cancelled (normal completion) and ensures the slot is still released
func TestDownloadSlotWithContextCancellation(t *testing.T) {
	// Create a test HTTP request context
	req, _ := http.NewRequest("GET", "/test", nil)

	// Create a cancellable context
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	// Create a gin context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Set up user claims for stable user identification
	claims := map[string]any{
		"sub": "test.user2@example.com",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	c.Set("user_claims", claims)

	// Simulate the download flow
	func() {
		// This simulates the defer releaseDownloadSlot(c) pattern
		defer releaseDownloadSlot(c)

		// Acquire slot
		assert.True(t, acquireDownloadSlot(c), "Should be able to acquire download slot")

		// Cancel context to simulate normal completion
		cancel()

		// Check that context is cancelled (this would happen in the streaming loop)
		select {
		case <-ctx.Done():
			// This is normal - the context is cancelled when the download completes
			t.Log("Context cancelled as expected (normal completion)")
		default:
			t.Error("Context should be cancelled")
		}

		// At this point, the function would return, and defer would release the slot
	}()

	// Verify slot is released after the function completes
	assert.True(t, acquireDownloadSlot(c), "Should be able to acquire slot after function completes")

	// Clean up
	releaseDownloadSlot(c)
}

// ============================================
// validateFilePath Tests
// ============================================

func TestValidateFilePath(t *testing.T) {
	testCases := []struct {
		name        string
		path        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid absolute path",
			path:        "/data/files/test.txt",
			expectError: false,
		},
		{
			name:        "valid nested path",
			path:        "/home/user/data/files/test.txt",
			expectError: false,
		},
		{
			name:        "valid root path",
			path:        "/",
			expectError: false,
		},
		{
			name:        "path with directory traversal",
			path:        "/data/../etc/passwd",
			expectError: true,
			errorMsg:    "path contains directory traversal",
		},
		{
			name:        "path with double dots in middle",
			path:        "/data/files/../secrets/key.txt",
			expectError: true,
			errorMsg:    "path contains directory traversal",
		},
		{
			name:        "path starting with directory traversal",
			path:        "/../data/file.txt",
			expectError: true,
			errorMsg:    "path contains directory traversal",
		},
		{
			name:        "relative path without leading slash",
			path:        "data/files/test.txt",
			expectError: true,
			errorMsg:    "path must be absolute",
		},
		{
			name:        "empty path",
			path:        "",
			expectError: true,
			errorMsg:    "path must be absolute",
		},
		{
			name:        "path with null byte",
			path:        "/data/files/test\x00.txt",
			expectError: true,
			errorMsg:    "path contains invalid characters",
		},
		{
			name:        "path with newline",
			path:        "/data/files/test\n.txt",
			expectError: true,
			errorMsg:    "path contains invalid characters",
		},
		{
			name:        "path with carriage return",
			path:        "/data/files/test\r.txt",
			expectError: true,
			errorMsg:    "path contains invalid characters",
		},
		{
			name:        "path with spaces",
			path:        "/data/files/test file.txt",
			expectError: false,
		},
		{
			name:        "path with unicode characters",
			path:        "/data/files/测试文件.txt",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateFilePath(tc.path)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ============================================
// FetchInitialDir and FetchHostName Tests
// ============================================

func setupTestXRDConfig() {
	testConfig := &config.Config{
		Env: "test",
		Server: config.ServerConfig{
			Address: ":8080",
		},
		XRD: config.XRDConfig{
			Host:       "test-xrd-server.example.com",
			Port:       1094,
			InitialDir: "/test/initial/dir",
			User:       "testuser",
			Download: config.DownloadConfig{
				BufferSize:    2097152,
				FlushInterval: 4194304,
			},
		},
	}
	config.SetConfig(testConfig)
}

func TestFetchInitialDir(t *testing.T) {
	setupTestXRDConfig()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/xrd/initial-dir", nil)

	FetchInitialDir(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, float64(http.StatusOK), response["code"])
	assert.Equal(t, "/test/initial/dir", response["data"])
}

func TestFetchHostName(t *testing.T) {
	setupTestXRDConfig()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/xrd/hostname", nil)

	FetchHostName(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, float64(http.StatusOK), response["code"])
	assert.Equal(t, "test-xrd-server.example.com", response["data"])
}

// ============================================
// GetInitialDirectory Tests
// ============================================

func TestGetInitialDirectory(t *testing.T) {
	setupTestXRDConfig()

	t.Run("without user claims", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/xrd/directory", nil)

		GetInitialDirectory(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "/", response["directory"])
	})

	t.Run("with user claims", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/xrd/directory", nil)

		// Set user claims
		claims := map[string]any{
			"sub": "test.user@example.com",
		}
		c.Set("user_claims", claims)

		GetInitialDirectory(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		// Currently returns "/" even with claims, future enhancement could use user-specific directories
		assert.Equal(t, "/", response["directory"])
	})
}

// ============================================
// GetHostName Tests
// ============================================

func TestGetHostName(t *testing.T) {
	setupTestXRDConfig()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/xrd/host", nil)

	GetHostName(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "test-xrd-server.example.com", response["hostname"])
}

// ============================================
// GetDownloadSlotStatus Tests
// ============================================

func TestGetDownloadSlotStatus(t *testing.T) {
	// Clear slots for clean test
	userDownloadSlots = make(map[string]bool)

	t.Run("no active slots", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/xrd/download-status", nil)
		c.Set("user_claims", map[string]any{"sub": "user1"})
		c.Set("access_token", "token1")

		GetDownloadSlotStatus(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, false, response["hasActiveSlot"])
		assert.Equal(t, float64(0), response["totalActiveSlots"])
	})

	t.Run("with active slot for current user", func(t *testing.T) {
		// Clear and acquire a slot
		userDownloadSlots = make(map[string]bool)
		c1 := createMockContext("user1", "token1")
		acquireDownloadSlot(c1)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/xrd/download-status", nil)
		c.Set("user_claims", map[string]any{"sub": "user1"})
		c.Set("access_token", "token1")

		GetDownloadSlotStatus(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, true, response["hasActiveSlot"])
		assert.Equal(t, float64(1), response["totalActiveSlots"])

		// Clean up
		releaseDownloadSlot(c1)
	})

	t.Run("with active slots for other users", func(t *testing.T) {
		// Clear and acquire slots for other users
		userDownloadSlots = make(map[string]bool)
		c1 := createMockContext("user1", "token1")
		c2 := createMockContext("user2", "token2")
		acquireDownloadSlot(c1)
		acquireDownloadSlot(c2)

		// Check status for a different user
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/xrd/download-status", nil)
		c.Set("user_claims", map[string]any{"sub": "user3"})
		c.Set("access_token", "token3")

		GetDownloadSlotStatus(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, false, response["hasActiveSlot"])
		assert.Equal(t, float64(2), response["totalActiveSlots"])

		// Clean up
		releaseDownloadSlot(c1)
		releaseDownloadSlot(c2)
	})
}

// ============================================
// ForceReleaseDownloadSlot Tests
// ============================================

func TestForceReleaseDownloadSlot(t *testing.T) {
	t.Run("no active slot to release", func(t *testing.T) {
		userDownloadSlots = make(map[string]bool)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/xrd/force-release", nil)
		c.Set("user_claims", map[string]any{"sub": "user1"})
		c.Set("access_token", "token1")

		ForceReleaseDownloadSlot(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["message"], "No active download slot found")
	})

	t.Run("force release active slot", func(t *testing.T) {
		// Clear and acquire a slot
		userDownloadSlots = make(map[string]bool)
		c1 := createMockContext("user1", "token1")
		acquireDownloadSlot(c1)

		// Verify slot is acquired
		assert.True(t, userDownloadSlots[getUserKey(c1)])

		// Force release
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/xrd/force-release", nil)
		c.Set("user_claims", map[string]any{"sub": "user1"})
		c.Set("access_token", "token1")

		ForceReleaseDownloadSlot(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["message"], "Download slot forcefully released")
		assert.Equal(t, float64(0), response["remainingSlots"])

		// Verify slot is released
		assert.False(t, userDownloadSlots[getUserKey(c1)])
	})

	t.Run("force release only affects current user", func(t *testing.T) {
		// Clear and acquire slots for multiple users
		userDownloadSlots = make(map[string]bool)
		c1 := createMockContext("user1", "token1")
		c2 := createMockContext("user2", "token2")
		acquireDownloadSlot(c1)
		acquireDownloadSlot(c2)

		// Force release user1's slot
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/xrd/force-release", nil)
		c.Set("user_claims", map[string]any{"sub": "user1"})
		c.Set("access_token", "token1")

		ForceReleaseDownloadSlot(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(1), response["remainingSlots"])

		// Verify user1's slot is released but user2's remains
		assert.False(t, userDownloadSlots[getUserKey(c1)])
		assert.True(t, userDownloadSlots[getUserKey(c2)])

		// Clean up
		releaseDownloadSlot(c2)
	})
}

// ============================================
// FetchDirItemsByPage Tests
// ============================================

func TestFetchDirItemsByPage(t *testing.T) {
	setupTestXRDConfig()

	t.Run("missing request body", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/xrd/dir-items", nil)
		c.Request.Header.Set("Content-Type", "application/json")

		FetchDirItemsByPage(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid page number - zero", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		reqBody := `{"path": "/test", "page": 0, "pageSize": 10}`
		c.Request = httptest.NewRequest("POST", "/api/xrd/dir-items", strings.NewReader(reqBody))
		c.Request.Header.Set("Content-Type", "application/json")

		FetchDirItemsByPage(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("empty directory path", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		reqBody := `{"path": "", "page": 1, "pageSize": 10}`
		c.Request = httptest.NewRequest("POST", "/api/xrd/dir-items", strings.NewReader(reqBody))
		c.Request.Header.Set("Content-Type", "application/json")

		FetchDirItemsByPage(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// ============================================
// ListDirectory Tests
// ============================================

func TestListDirectory(t *testing.T) {
	setupTestXRDConfig()

	t.Run("missing directory parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/xrd/list", nil)

		ListDirectory(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "Directory parameter is required")
	})

	t.Run("empty directory parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/xrd/list?dir=", nil)

		ListDirectory(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// ============================================
// GetFileInfo Tests
// ============================================

func TestGetFileInfo(t *testing.T) {
	setupTestXRDConfig()

	t.Run("missing path parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/xrd/file-info", nil)

		GetFileInfo(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "File path parameter is required")
	})

	t.Run("empty path parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/xrd/file-info?path=", nil)

		GetFileInfo(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// ============================================
// DownloadFile Tests
// ============================================

func TestDownloadFile(t *testing.T) {
	setupTestXRDConfig()

	t.Run("missing path parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/xrd/download", nil)

		DownloadFile(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "File path parameter is required")
	})

	t.Run("empty path parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/xrd/download?path=", nil)

		DownloadFile(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid path with directory traversal", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/xrd/download?path=/../etc/passwd", nil)

		DownloadFile(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "Invalid file path")
	})

	t.Run("invalid path - relative", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/api/xrd/download?path=relative/path/file.txt", nil)

		DownloadFile(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "Invalid file path")
	})

	t.Run("invalid path with null byte", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		// URL encode the null byte
		c.Request = httptest.NewRequest("GET", "/api/xrd/download?path=/data/file%00.txt", nil)

		DownloadFile(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid path with newline", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		// URL encode the newline
		c.Request = httptest.NewRequest("GET", "/api/xrd/download?path=/data/file%0A.txt", nil)

		DownloadFile(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// ============================================
// maskToken Tests
// ============================================

func TestMaskToken(t *testing.T) {
	testCases := []struct {
		name     string
		token    string
		expected string
	}{
		{
			name:     "empty token",
			token:    "",
			expected: "anonymous",
		},
		{
			name:     "short token (8 chars or less)",
			token:    "short",
			expected: "***",
		},
		{
			name:     "exactly 8 chars",
			token:    "12345678",
			expected: "***",
		},
		{
			name:     "normal token",
			token:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expected: "eyJh...VCJ9",
		},
		{
			name:     "longer token",
			token:    "abcdefghijklmnopqrstuvwxyz1234567890",
			expected: "abcd...7890",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := maskToken(tc.token)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// ============================================
// min function Tests
// ============================================

func TestMin(t *testing.T) {
	testCases := []struct {
		a, b, expected uint32
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{0, 10, 0},
		{10, 0, 0},
		{100, 200, 100},
	}

	for _, tc := range testCases {
		result := min(tc.a, tc.b)
		assert.Equal(t, tc.expected, result)
	}
}

// ============================================
// validateBatchFileName Tests
// ============================================

func TestValidateBatchFileName(t *testing.T) {
	testCases := []struct {
		name        string
		filename    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid simple filename",
			filename:    "data.txt",
			expectError: false,
		},
		{
			name:        "valid filename with spaces",
			filename:    "my data file.csv",
			expectError: false,
		},
		{
			name:        "valid filename with unicode",
			filename:    "données_2024.txt",
			expectError: false,
		},
		{
			name:        "valid filename with dots",
			filename:    "archive.tar.gz",
			expectError: false,
		},
		{
			name:        "empty filename",
			filename:    "",
			expectError: true,
			errorMsg:    "empty filename",
		},
		{
			name:        "filename with forward slash",
			filename:    "path/file.txt",
			expectError: true,
			errorMsg:    "path separator",
		},
		{
			name:        "filename with backslash",
			filename:    "path\\file.txt",
			expectError: true,
			errorMsg:    "path separator",
		},
		{
			name:        "filename with directory traversal",
			filename:    "..passwd",
			expectError: true,
			errorMsg:    "directory traversal",
		},
		{
			name:        "filename with double dots",
			filename:    "file..txt",
			expectError: true,
			errorMsg:    "directory traversal",
		},
		{
			name:        "filename with null byte",
			filename:    "file\x00.txt",
			expectError: true,
			errorMsg:    "invalid characters",
		},
		{
			name:        "filename with newline",
			filename:    "file\n.txt",
			expectError: true,
			errorMsg:    "invalid characters",
		},
		{
			name:        "filename with carriage return",
			filename:    "file\r.txt",
			expectError: true,
			errorMsg:    "invalid characters",
		},
		{
			name:        "filename with invalid UTF-8 bytes",
			filename:    "file\xff\xfe.txt",
			expectError: true,
			errorMsg:    "invalid UTF-8",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateBatchFileName(tc.filename)
			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ============================================
// DownloadMultipleFiles Tests
// ============================================

func TestDownloadMultipleFiles_Validation(t *testing.T) {
	setupTestXRDConfig()

	t.Run("missing request body", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/api/v1/xrd/download/batch", nil)
		c.Request.Header.Set("Content-Type", "application/json")

		DownloadMultipleFiles(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp["error"], "Invalid request body")
	})

	t.Run("empty files list", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"basePath": "/data", "files": []}`
		c.Request = httptest.NewRequest("POST", "/api/v1/xrd/download/batch", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		DownloadMultipleFiles(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp["error"], "No files specified")
	})

	t.Run("exceeds max file count", func(t *testing.T) {
		// Set config with low max
		testConfig := &config.Config{
			XRD: config.XRDConfig{
				Download: config.DownloadConfig{
					MaxBatchFiles:  2,
					MaxBatchSizeMB: 10240,
				},
			},
		}
		config.SetConfig(testConfig)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"basePath": "/data", "files": ["a.txt", "b.txt", "c.txt"]}`
		c.Request = httptest.NewRequest("POST", "/api/v1/xrd/download/batch", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		DownloadMultipleFiles(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp["error"], "Too many files")
		assert.Contains(t, resp["error"], "exceeds maximum of 2")
	})

	t.Run("invalid base path - relative", func(t *testing.T) {
		setupTestXRDConfig()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"basePath": "relative/path", "files": ["file.txt"]}`
		c.Request = httptest.NewRequest("POST", "/api/v1/xrd/download/batch", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		DownloadMultipleFiles(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp["error"], "Invalid base path")
	})

	t.Run("invalid base path - directory traversal", func(t *testing.T) {
		setupTestXRDConfig()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"basePath": "/data/../etc", "files": ["passwd"]}`
		c.Request = httptest.NewRequest("POST", "/api/v1/xrd/download/batch", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		DownloadMultipleFiles(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp["error"], "Invalid base path")
	})

	t.Run("invalid filename with path separator", func(t *testing.T) {
		setupTestXRDConfig()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"basePath": "/data", "files": ["sub/file.txt"]}`
		c.Request = httptest.NewRequest("POST", "/api/v1/xrd/download/batch", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		DownloadMultipleFiles(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp["error"], "Invalid filename")
		assert.Contains(t, resp["error"], "path separator")
	})

	t.Run("invalid filename with backslash", func(t *testing.T) {
		setupTestXRDConfig()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"basePath": "/data", "files": ["sub\\file.txt"]}`
		c.Request = httptest.NewRequest("POST", "/api/v1/xrd/download/batch", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		DownloadMultipleFiles(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp["error"], "Invalid filename")
	})

	t.Run("filename with null byte", func(t *testing.T) {
		setupTestXRDConfig()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := "{\"basePath\": \"/data\", \"files\": [\"file\\u0000.txt\"]}"
		c.Request = httptest.NewRequest("POST", "/api/v1/xrd/download/batch", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		DownloadMultipleFiles(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp["error"], "Invalid filename")
	})

	t.Run("missing basePath field", func(t *testing.T) {
		setupTestXRDConfig()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"files": ["file.txt"]}`
		c.Request = httptest.NewRequest("POST", "/api/v1/xrd/download/batch", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		DownloadMultipleFiles(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing files field", func(t *testing.T) {
		setupTestXRDConfig()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		body := `{"basePath": "/data"}`
		c.Request = httptest.NewRequest("POST", "/api/v1/xrd/download/batch", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		DownloadMultipleFiles(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// ============================================
// Mock types for streamFileToTar tests
// ============================================

// mockXRDFile is a minimal xrdfs.File implementation backed by an in-memory byte slice.
// readErr, if non-nil, is returned on every ReadAt call (with any bytes that fit).
type mockXRDFile struct {
	content []byte
	readErr error
}

func (m *mockXRDFile) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(m.content)) {
		if m.readErr != nil {
			return 0, m.readErr
		}
		return 0, io.EOF
	}
	n := copy(p, m.content[off:])
	if m.readErr != nil {
		return n, m.readErr
	}
	if off+int64(n) >= int64(len(m.content)) {
		return n, io.EOF
	}
	return n, nil
}

func (m *mockXRDFile) WriteAt(p []byte, off int64) (int, error)          { return 0, nil }
func (m *mockXRDFile) Compression() *xrdfs.FileCompression               { return nil }
func (m *mockXRDFile) Info() *xrdfs.EntryStat                            { return nil }
func (m *mockXRDFile) Handle() xrdfs.FileHandle                          { return xrdfs.FileHandle{} }
func (m *mockXRDFile) Close(ctx context.Context) error                   { return nil }
func (m *mockXRDFile) CloseVerify(ctx context.Context, size int64) error { return nil }
func (m *mockXRDFile) Sync(ctx context.Context) error                    { return nil }
func (m *mockXRDFile) ReadAtContext(ctx context.Context, p []byte, off int64) (int, error) {
	return m.ReadAt(p, off)
}
func (m *mockXRDFile) WriteAtContext(ctx context.Context, p []byte, off int64) error { return nil }
func (m *mockXRDFile) Truncate(ctx context.Context, size int64) error                { return nil }
func (m *mockXRDFile) Stat(ctx context.Context) (xrdfs.EntryStat, error) {
	return xrdfs.EntryStat{}, nil
}

func (m *mockXRDFile) StatVirtualFS(ctx context.Context) (xrdfs.VirtualFSStat, error) {
	return xrdfs.VirtualFSStat{}, nil
}
func (m *mockXRDFile) VerifyWriteAt(ctx context.Context, p []byte, off int64) error { return nil }

// mockXRDFileSystem is a minimal xrdfs.FileSystem implementation for testing.
type mockXRDFileSystem struct {
	file    xrdfs.File
	openErr error
}

func (m *mockXRDFileSystem) Open(ctx context.Context, path string, mode xrdfs.OpenMode, options xrdfs.OpenOptions) (xrdfs.File, error) {
	if m.openErr != nil {
		return nil, m.openErr
	}
	return m.file, nil
}

func (m *mockXRDFileSystem) Dirlist(ctx context.Context, path string) ([]xrdfs.EntryStat, error) {
	return nil, nil
}
func (m *mockXRDFileSystem) RemoveFile(ctx context.Context, path string) error { return nil }
func (m *mockXRDFileSystem) Truncate(ctx context.Context, path string, size int64) error {
	return nil
}

func (m *mockXRDFileSystem) Stat(ctx context.Context, path string) (xrdfs.EntryStat, error) {
	return xrdfs.EntryStat{}, nil
}

func (m *mockXRDFileSystem) VirtualStat(ctx context.Context, path string) (xrdfs.VirtualFSStat, error) {
	return xrdfs.VirtualFSStat{}, nil
}

func (m *mockXRDFileSystem) Mkdir(ctx context.Context, path string, perm xrdfs.OpenMode) error {
	return nil
}

func (m *mockXRDFileSystem) MkdirAll(ctx context.Context, path string, perm xrdfs.OpenMode) error {
	return nil
}
func (m *mockXRDFileSystem) RemoveDir(ctx context.Context, path string) error          { return nil }
func (m *mockXRDFileSystem) RemoveAll(ctx context.Context, path string) error          { return nil }
func (m *mockXRDFileSystem) Rename(ctx context.Context, oldpath, newpath string) error { return nil }
func (m *mockXRDFileSystem) Chmod(ctx context.Context, path string, mode xrdfs.OpenMode) error {
	return nil
}

func (m *mockXRDFileSystem) Statx(ctx context.Context, paths []string) ([]xrdfs.StatFlags, error) {
	return nil, nil
}

// ============================================
// streamFileToTar Tests
// ============================================

func TestStreamFileToTar_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	content := []byte("hello world content")
	fs := &mockXRDFileSystem{file: &mockXRDFile{content: content}}

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	n, err := streamFileToTar(c, fs, tw, "/test/file.txt", "file.txt", int64(len(content)), time.Now(), 32*1024, 4*1024*1024, "test-id")

	assert.NoError(t, err)
	assert.Equal(t, int64(len(content)), n)
	assert.NoError(t, tw.Close())

	// Verify the tar archive is valid and contains the expected entry
	tr := tar.NewReader(&buf)
	hdr, err := tr.Next()
	assert.NoError(t, err)
	assert.Equal(t, "file.txt", hdr.Name)
	assert.Equal(t, int64(len(content)), hdr.Size)

	data, err := io.ReadAll(tr)
	assert.NoError(t, err)
	assert.Equal(t, content, data)

	_, err = tr.Next()
	assert.Equal(t, io.EOF, err)
}

func TestStreamFileToTar_ShortRead(t *testing.T) {
	// File returns fewer bytes than declared in the header.
	// streamFileToTar must pad the tar entry to keep the archive valid.
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	content := []byte("short") // 5 bytes
	declaredSize := int64(20)  // header declares 20 bytes
	fs := &mockXRDFileSystem{file: &mockXRDFile{content: content}}

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	n, err := streamFileToTar(c, fs, tw, "/test/file.txt", "file.txt", declaredSize, time.Now(), 32*1024, 4*1024*1024, "test-id")

	assert.Error(t, err, "expected error on short read")
	assert.Contains(t, err.Error(), "short read")
	assert.Equal(t, int64(5), n)

	// The tar writer must still be closeable after padding
	assert.NoError(t, tw.Close(), "tar writer must remain valid after short read")

	// The archive must be parseable - the short entry should be readable
	tr := tar.NewReader(&buf)
	hdr, err := tr.Next()
	assert.NoError(t, err)
	assert.Equal(t, "file.txt", hdr.Name)
	assert.Equal(t, declaredSize, hdr.Size)

	// Consume the entry (padded to declaredSize)
	_, err = io.Copy(io.Discard, tr)
	assert.NoError(t, err)

	// No more entries; archive is not corrupt
	_, err = tr.Next()
	assert.Equal(t, io.EOF, err)
}

func TestStreamFileToTar_ReadError(t *testing.T) {
	// File returns a read error after partial data.
	// streamFileToTar must pad the remaining tar entry bytes and propagate the error.
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	readErr := fmt.Errorf("simulated XRD read error")
	content := []byte("partial") // 7 bytes then readErr
	declaredSize := int64(20)
	fs := &mockXRDFileSystem{file: &mockXRDFile{content: content, readErr: readErr}}

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	_, err := streamFileToTar(c, fs, tw, "/test/file.txt", "file.txt", declaredSize, time.Now(), 32*1024, 4*1024*1024, "test-id")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "simulated XRD read error")

	// Tar writer must still be closeable after padding
	assert.NoError(t, tw.Close(), "tar writer must remain valid after read error")
}

func TestStreamFileToTar_OpenError(t *testing.T) {
	// If both Open attempts fail, streamFileToTar returns an error without writing any data.
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	fs := &mockXRDFileSystem{openErr: fmt.Errorf("connection refused")}

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	_, err := streamFileToTar(c, fs, tw, "/test/file.txt", "file.txt", 10, time.Now(), 32*1024, 4*1024*1024, "test-id")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open file")
}

// ============================================
// Tar error manifest tests
// ============================================

func TestTarErrorManifest(t *testing.T) {
	// Verify that _DOWNLOAD_ERRORS.txt can be appended after file entries and the archive remains valid.
	// This mirrors the error manifest logic in DownloadMultipleFiles.
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Write a successful file entry
	fileContent := []byte("good file content")
	assert.NoError(t, tw.WriteHeader(&tar.Header{
		Name:    "good_file.txt",
		Size:    int64(len(fileContent)),
		Mode:    0o644,
		ModTime: time.Now(),
	}))
	_, err := tw.Write(fileContent)
	assert.NoError(t, err)

	// Simulate the error manifest written by DownloadMultipleFiles
	failedFiles := []string{"failed_file1.txt", "failed_file2.txt"}
	errContent := "The following files could not be fully downloaded:\n"
	for _, name := range failedFiles {
		errContent += "  - " + name + "\n"
	}
	assert.NoError(t, tw.WriteHeader(&tar.Header{
		Name:    "_DOWNLOAD_ERRORS.txt",
		Size:    int64(len(errContent)),
		Mode:    0o644,
		ModTime: time.Now(),
	}))
	_, err = tw.Write([]byte(errContent))
	assert.NoError(t, err)

	assert.NoError(t, tw.Close())

	// Verify archive structure
	tr := tar.NewReader(&buf)

	hdr1, err := tr.Next()
	assert.NoError(t, err)
	assert.Equal(t, "good_file.txt", hdr1.Name)
	data1, err := io.ReadAll(tr)
	assert.NoError(t, err)
	assert.Equal(t, fileContent, data1)

	hdr2, err := tr.Next()
	assert.NoError(t, err)
	assert.Equal(t, "_DOWNLOAD_ERRORS.txt", hdr2.Name)
	data2, err := io.ReadAll(tr)
	assert.NoError(t, err)
	assert.Contains(t, string(data2), "failed_file1.txt")
	assert.Contains(t, string(data2), "failed_file2.txt")

	_, err = tr.Next()
	assert.Equal(t, io.EOF, err)
}

// ============================================
// streamFileToTarTimeout Tests
// ============================================

func TestStreamFileToTarTimeout(t *testing.T) {
	const (
		minTimeout = 30 * time.Minute
		mib        = int64(1 << 20) // 1 MiB
	)

	tests := []struct {
		name        string
		fileSize    int64
		wantExact   time.Duration // zero means skip exact check
		wantAtLeast time.Duration // zero means skip lower-bound check
	}{
		{
			name:      "zero file size returns min timeout",
			fileSize:  0,
			wantExact: minTimeout,
		},
		{
			name:      "negative file size returns min timeout",
			fileSize:  -1,
			wantExact: minTimeout,
		},
		{
			name:      "small file (1 KB) returns min timeout",
			fileSize:  1024,
			wantExact: minTimeout,
		},
		{
			name:      "exact 1 MiB boundary returns min timeout",
			fileSize:  mib,
			wantExact: minTimeout, // sizeBasedTimeout=1s, timeout=301s < 30min
		},
		{
			name:        "large file (2 GiB) returns timeout above minimum",
			fileSize:    2 * 1024 * mib,
			wantAtLeast: minTimeout + time.Second, // must exceed 30-minute floor
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := streamFileToTarTimeout(tc.fileSize)
			assert.Positive(t, got, "timeout must be positive")
			if tc.wantExact != 0 {
				assert.Equal(t, tc.wantExact, got)
			}
			if tc.wantAtLeast != 0 {
				assert.GreaterOrEqual(t, got, tc.wantAtLeast)
			}
		})
	}
}

// ============================================
// streamFileSimple Tests
// ============================================

func TestStreamFileSimple_Success(t *testing.T) {
	setupTestXRDConfig()
	gin.SetMode(gin.TestMode)

	content := []byte("file content for streaming")
	fs := &mockXRDFileSystem{file: &mockXRDFile{content: content}}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	err := streamFileSimple(c, fs, "/data/test.txt", "token123", time.Now(), int64(len(content)), "dl-001")

	assert.NoError(t, err)
	assert.Equal(t, content, w.Body.Bytes())
}

func TestStreamFileSimple_OpenError(t *testing.T) {
	setupTestXRDConfig()
	gin.SetMode(gin.TestMode)

	openErr := fmt.Errorf("connection refused")
	fs := &mockXRDFileSystem{openErr: openErr}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	err := streamFileSimple(c, fs, "/data/test.txt", "token123", time.Now(), 1024, "dl-002")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open file")
}

func TestStreamFileSimple_InitialReadError(t *testing.T) {
	setupTestXRDConfig()
	gin.SetMode(gin.TestMode)

	readErr := fmt.Errorf("XRD read failure")
	// File has partial content but returns an error on read
	fs := &mockXRDFileSystem{file: &mockXRDFile{content: []byte("partial"), readErr: readErr}}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	// Declare a larger fileSize so the initial read error is not masked by EOF
	err := streamFileSimple(c, fs, "/data/test.txt", "token123", time.Now(), 1024, "dl-003")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read initial chunk from file")
}

func TestStreamFileSimple_EmptyFile(t *testing.T) {
	setupTestXRDConfig()
	gin.SetMode(gin.TestMode)

	// Empty file: ReadAt immediately returns 0, EOF
	fs := &mockXRDFileSystem{file: &mockXRDFile{content: []byte{}}}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	err := streamFileSimple(c, fs, "/data/empty.txt", "token123", time.Now(), 0, "dl-004")

	assert.NoError(t, err)
	assert.Empty(t, w.Body.Bytes())
}
