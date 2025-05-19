package common

import (
	"fmt"

	"go-hep.org/x/hep/xrootd/xrdproto/auth"
)

// TokenAuth provides Bearer token authentication for XRootD
type TokenAuth struct {
	token string
}

// NewTokenAuth creates a new token-based authenticator
func NewTokenAuth(token string) *TokenAuth {
	return &TokenAuth{token: token}
}

// Provider returns the name of the security provider
func (ta *TokenAuth) Provider() string {
	return "bearer"
}

// Request forms an authorization Request according to passed parameters
func (ta *TokenAuth) Request(params []string) (*auth.Request, error) {
	if ta.token == "" {
		return nil, fmt.Errorf("no token provided for bearer authentication")
	}

	req := &auth.Request{
		Type:        [4]byte{'b', 'e', 'a', 'r'}, // "bear" for bearer
		Credentials: fmt.Sprintf("Bearer %s", ta.token),
	}

	return req, nil
}

// NoAuth provides a no-op authenticator that skips authentication
type NoAuth struct{}

// NewNoAuth creates a new no-authentication provider
func NewNoAuth() *NoAuth {
	return &NoAuth{}
}

// Provider returns the name of the security provider
func (na *NoAuth) Provider() string {
	return "none"
}

// Request forms an authorization Request according to passed parameters
func (na *NoAuth) Request(params []string) (*auth.Request, error) {
	// Return nil to indicate no authentication is needed
	return nil, nil
}
