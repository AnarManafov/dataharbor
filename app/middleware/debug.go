package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/AnarManafov/dataharbor/app/common"
	"github.com/gin-gonic/gin"
)

// DebugRequestBody is a middleware that logs request and response bodies
// This is useful for debugging API issues, especially for authentication endpoints,
// but should only be enabled in development environments.
// Headers and bodies are passed through common.Redact* and credentials are never logged.
func DebugRequestBody() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip binary data paths to avoid flooding logs
		if shouldSkipPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Start timer for request duration
		startTime := time.Now()

		// Log request info
		common.Debugf(c, "Request: %s %s", c.Request.Method, c.Request.URL.Path)
		common.Debugf(c, "Request Headers: %v", common.RedactHeaders(c.Request.Header))

		// Streaming routes (file downloads, upload chunks) keep the request
		// line, status and headers but never have their bodies buffered.
		captureBody := !shouldSkipBody(c.Request.Method, c.Request.URL.Path)

		// Read and restore the request body for logging
		if captureBody && c.Request.Body != nil {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			if len(bodyBytes) > 0 {
				common.Debugf(c, "Request Body: %s", common.RedactBody(c.Request.Header.Get("Content-Type"), bodyBytes))
				// Restore the request body for other middleware and handlers
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		// Create a response writer that captures the response
		var blw *bodyLogWriter
		if captureBody {
			blw = &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
			c.Writer = blw
		}

		// Process the request
		c.Next()

		// Calculate duration
		duration := time.Since(startTime)

		// Log response info after request is processed
		common.Debugf(c, "Response Status: %d", c.Writer.Status())
		common.Debugf(c, "Response Duration: %v", duration)
		common.Debugf(c, "Response Headers: %v", common.RedactHeaders(c.Writer.Header()))

		// Only log response body for certain content types
		contentType := c.Writer.Header().Get("Content-Type")
		if blw != nil && shouldLogResponseBody(contentType) && blw.body.Len() > 0 {
			// Truncate very large responses to avoid flooding logs
			responseBody := common.RedactBody(contentType, blw.body.Bytes())
			if len(responseBody) > 4096 {
				responseBody = responseBody[:4096] + "... (truncated)"
			}
			common.Debugf(c, "Response Body: %s", responseBody)
		}
	}
}

// bodyLogWriter captures the response body for logging
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write captures the response and also writes to the original response writer
func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// Skip paths that might contain binary data or are high-volume
func shouldSkipPath(path string) bool {
	// List of paths to skip for logging
	skipPaths := []string{
		"/health",      // Health check endpoints
		"/favicon.ico", // Browser favicon requests
		"/static/",     // Static assets
		"/assets/",     // Static assets
		"/download/",   // Legacy file download prefix
	}

	for _, p := range skipPaths {
		if len(path) >= len(p) && path[:len(p)] == p {
			return true
		}
	}
	return false
}

// shouldSkipBody reports whether a request's bodies must not be captured:
// file downloads (whole files would be buffered in bodyLogWriter) and upload
// chunk PUTs (the chunk would be read fully into memory). Other upload routes
// (session create, complete, status, abort, limits) are small JSON and keep
// their body logs.
func shouldSkipBody(method, path string) bool {
	switch path {
	case "/api/v1/xrd/download", "/api/v1/xrd/download/batch":
		return true
	}
	if method == http.MethodPut && strings.HasPrefix(path, "/api/v1/xrd/upload/") && strings.HasSuffix(path, "/chunk") {
		return true
	}
	return false
}

// Only log response bodies for certain content types
func shouldLogResponseBody(contentType string) bool {
	// Only log these content types
	allowedTypes := []string{
		"application/json",
		"text/plain",
		"text/html",
		"application/xml",
		"text/xml",
	}

	for _, t := range allowedTypes {
		if len(contentType) >= len(t) && contentType[:len(t)] == t {
			return true
		}
	}
	return false
}
