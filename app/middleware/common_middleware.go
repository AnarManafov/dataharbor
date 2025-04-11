package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/AnarManafov/data_lake_ui/app/common"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Recovery returns a middleware that recovers from panics in the application
// This prevents the entire application from crashing when a single request handler fails,
// improving overall system stability and availability under unexpected conditions.
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			c.String(http.StatusInternalServerError, fmt.Sprintf("Server error: %s", err))
		}
		c.AbortWithStatus(http.StatusInternalServerError)
	})
}

// Note: CORS middleware has been moved to cors.go

// TraceRequest returns a middleware that adds distributed tracing capabilities
// by attaching a unique ID to each request. This enables:
// 1. End-to-end request tracking across services
// 2. Correlation of logs from different components handling the same request
// 3. Performance analysis of request processing times
func TraceRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate a unique trace ID using UUID to ensure global uniqueness
		// even in distributed deployments with high request volumes
		traceID := uuid.New().String()
		c.Set("tid", traceID)

		// Log the beginning of request processing to establish the trace boundary
		common.Infof(c, "Request started: %s %s", c.Request.Method, c.Request.URL.Path)

		// Capture start time to enable performance monitoring
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		// Use different log levels based on response status to simplify error detection
		// and enable proper alerting on request failures
		status := c.Writer.Status()
		if status >= 400 {
			common.Errorf(c, "Request failed: %s %s - status %d (took %v)",
				c.Request.Method, c.Request.URL.Path, status, duration)
		} else {
			common.Infof(c, "Request completed: %s %s - status %d (took %v)",
				c.Request.Method, c.Request.URL.Path, status, duration)
		}
	}
}
