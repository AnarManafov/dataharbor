package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
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
		claims := map[string]interface{}{
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
	claims := map[string]interface{}{
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
	claims := map[string]interface{}{
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
