package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/AnarManafov/dataharbor/app/config"
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
