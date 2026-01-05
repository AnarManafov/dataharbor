package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTokenAuth(t *testing.T) {
	token := "test-bearer-token-12345"
	auth := NewTokenAuth(token)

	assert.NotNil(t, auth, "NewTokenAuth should return non-nil")
	assert.Equal(t, token, auth.token, "Token should be stored correctly")
}

func TestTokenAuth_Provider(t *testing.T) {
	auth := NewTokenAuth("some-token")
	provider := auth.Provider()

	assert.Equal(t, "bearer", provider, "Provider should return 'bearer'")
}

func TestTokenAuth_Request_WithToken(t *testing.T) {
	token := "my-jwt-access-token"
	auth := NewTokenAuth(token)

	req, err := auth.Request([]string{"param1", "param2"})

	assert.NoError(t, err, "Request should not return an error when token is provided")
	assert.NotNil(t, req, "Request should return non-nil auth.Request")
	assert.Equal(t, [4]byte{'b', 'e', 'a', 'r'}, req.Type, "Type should be 'bear'")
	assert.Equal(t, "Bearer my-jwt-access-token", req.Credentials, "Credentials should include Bearer prefix")
}

func TestTokenAuth_Request_EmptyToken(t *testing.T) {
	auth := NewTokenAuth("")

	req, err := auth.Request(nil)

	assert.Error(t, err, "Request should return an error when token is empty")
	assert.Nil(t, req, "Request should return nil when token is empty")
	assert.Contains(t, err.Error(), "no token provided", "Error message should indicate missing token")
}

func TestNewNoAuth(t *testing.T) {
	auth := NewNoAuth()

	assert.NotNil(t, auth, "NewNoAuth should return non-nil")
}

func TestNoAuth_Provider(t *testing.T) {
	auth := NewNoAuth()
	provider := auth.Provider()

	assert.Equal(t, "none", provider, "Provider should return 'none'")
}

func TestNoAuth_Request(t *testing.T) {
	auth := NewNoAuth()

	req, err := auth.Request([]string{"param1", "param2"})

	assert.NoError(t, err, "Request should not return an error")
	assert.Nil(t, req, "Request should return nil to skip authentication")
}

func TestNoAuth_Request_NilParams(t *testing.T) {
	auth := NewNoAuth()

	req, err := auth.Request(nil)

	assert.NoError(t, err, "Request should not return an error with nil params")
	assert.Nil(t, req, "Request should return nil")
}
