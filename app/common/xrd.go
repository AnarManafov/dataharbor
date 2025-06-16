package common

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/AnarManafov/dataharbor/app/config"
	"go.uber.org/zap"
)

var (
	xrdClient     *XRDClient
	xrdClientOnce sync.Once
)

// XRDClient is a client for XRootD
type XRDClient struct {
	URL      string
	Logger   *zap.SugaredLogger
	XRDToken string
}

// GetXRDClient returns an XRootD client
func GetXRDClient() *XRDClient {
	xrdClientOnce.Do(func() {
		logger := GetLogger()
		cfg := config.GetConfig()

		url := cfg.XRD.URL
		if url == "" {
			url = "root://localhost:1094"
		}

		xrdClient = &XRDClient{
			URL:    url,
			Logger: logger,
		}
	})
	return xrdClient
}

// SetUserToken sets the user token for XRD requests
func (c *XRDClient) SetUserToken(token string) {
	c.XRDToken = token
}

// BuildXRDURLWithCGI builds a URL for XRootD with CGI parameters
func (c *XRDClient) BuildXRDURLWithCGI(path string, extraParams map[string]string) (string, error) {
	// Ensure URL has trailing slash
	url := c.URL
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	url += path

	// Collect all URL parameters
	params := make([]string, 0)

	// Handle authentication parameters
	if err := c.addAuthParams(&params); err != nil {
		return "", err
	}

	// Add extra parameters
	for k, v := range extraParams {
		params = append(params, fmt.Sprintf("%s=%s", k, v))
	}

	// Append parameters to URL
	if len(params) > 0 {
		url += "?" + strings.Join(params, "&")
	}

	return url, nil
}

// Helper method to add authentication parameters
func (c *XRDClient) addAuthParams(params *[]string) error {
	cfg := config.GetConfig()

	// Prefer token-based authentication if available
	if c.XRDToken != "" {
		*params = append(*params, fmt.Sprintf("authz=Bearer %s", c.XRDToken))
		return nil
	}

	// Fall back to user credentials if token isn't available but auth is required
	if cfg.XRD.UserRequired {
		user := cfg.XRD.User
		if user == "" {
			return errors.New("xrootd user is required but not configured")
		}

		*params = append(*params, fmt.Sprintf("xrd.u=%s", user))

		// Add optional user group
		if cfg.XRD.UserGroup != "" {
			*params = append(*params, fmt.Sprintf("xrd.g=%s", cfg.XRD.UserGroup))
		}

		// Add optional user password
		if cfg.XRD.UserPwd != "" {
			*params = append(*params, fmt.Sprintf("xrd.k=%s", cfg.XRD.UserPwd))
		}
	}

	return nil
}

// CreateTLSConfig creates a TLS configuration for the client
func (c *XRDClient) CreateTLSConfig() (*tls.Config, error) {
	cfg := config.GetConfig()
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Load client certificate if configured
	if cfg.XRD.ClientCert != "" && cfg.XRD.ClientKey != "" {
		cert, err := tls.LoadX509KeyPair(cfg.XRD.ClientCert, cfg.XRD.ClientKey)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// Enable TLS renegotiation for older XRootD servers
	tlsConfig.Renegotiation = tls.RenegotiateOnceAsClient

	return tlsConfig, nil
}

// CreateHTTPClient creates an HTTP client for XRD requests
func (c *XRDClient) CreateHTTPClient() (*http.Client, error) {
	cfg := config.GetConfig()
	if !cfg.XRD.TLS {
		return http.DefaultClient, nil
	}

	tlsConfig, err := c.CreateTLSConfig()
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return &http.Client{
		Transport: transport,
	}, nil
}
