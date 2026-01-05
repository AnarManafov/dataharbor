package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetUserToken_FromAuthorizationHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Bearer test-token-12345")

	token, ok := GetUserToken(c)

	assert.True(t, ok, "Should return true when token is in Authorization header")
	assert.Equal(t, "test-token-12345", token)
}

func TestGetUserToken_FromSessionContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Set("access_token", "session-token-67890")

	token, ok := GetUserToken(c)

	assert.True(t, ok, "Should return true when token is in session context")
	assert.Equal(t, "session-token-67890", token)
}

func TestGetUserToken_HeaderTakesPrecedence(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Bearer header-token")
	c.Set("access_token", "session-token")

	token, ok := GetUserToken(c)

	assert.True(t, ok)
	assert.Equal(t, "header-token", token, "Authorization header should take precedence")
}

func TestGetUserToken_NoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	token, ok := GetUserToken(c)

	assert.False(t, ok, "Should return false when no token is available")
	assert.Empty(t, token)
}

func TestGetUserToken_InvalidAuthorizationHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Basic some-credentials") // Not Bearer

	token, ok := GetUserToken(c)

	assert.False(t, ok, "Should return false for non-Bearer authorization")
	assert.Empty(t, token)
}

func TestGetUserToken_EmptySessionToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Set("access_token", "")

	token, ok := GetUserToken(c)

	assert.False(t, ok, "Should return false for empty session token")
	assert.Empty(t, token)
}

func TestGetUserClaims_Valid(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	expectedClaims := map[string]interface{}{
		"sub":   "user-123",
		"email": "user@example.com",
		"name":  "Test User",
	}
	c.Set("user_claims", expectedClaims)

	claims, ok := GetUserClaims(c)

	assert.True(t, ok, "Should return true when claims exist")
	assert.Equal(t, expectedClaims, claims)
	assert.Equal(t, "user-123", claims["sub"])
	assert.Equal(t, "user@example.com", claims["email"])
}

func TestGetUserClaims_NotSet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	claims, ok := GetUserClaims(c)

	assert.False(t, ok, "Should return false when no claims are set")
	assert.Nil(t, claims)
}

func TestGetUserClaims_WrongType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Set("user_claims", "not a map") // Wrong type

	claims, ok := GetUserClaims(c)

	assert.False(t, ok, "Should return false when claims are wrong type")
	assert.Nil(t, claims)
}

func TestGetUserClaims_EmptyMap(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Set("user_claims", map[string]interface{}{})

	claims, ok := GetUserClaims(c)

	assert.True(t, ok, "Should return true for empty but valid claims map")
	assert.NotNil(t, claims)
	assert.Empty(t, claims)
}
