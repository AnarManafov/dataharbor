package middleware

import (
	"bytes"
	"io"
	"strings"
	"time"

	"github.com/AnarManafov/dataharbor/app/common"

	"github.com/gin-gonic/gin"
)

// CustomResponseWriter captures the response body for logging purposes
// without interfering with the normal response flow. This allows for
// comprehensive audit logs while maintaining performance.
type CustomResponseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write captures the written data for logging while passing it through to the original writer
func (w CustomResponseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// WriteString captures string data for logging while passing it through to the original writer
func (w CustomResponseWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// sliceContain checks if a string exists in a slice
// Used to implement a whitelist approach for header logging, which is
// more secure than a blacklist approach when dealing with sensitive data
func sliceContain(sli []string, k string) bool {
	for _, i := range sli {
		if i == k {
			return true
		}
	}
	return false
}

// AccessLogger creates a middleware that logs incoming requests and their responses
// with timing information. This middleware balances security (by filtering sensitive headers)
// with observability needs for troubleshooting production issues.
func AccessLogger() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Explicitly whitelist headers to log rather than blacklisting sensitive ones
		// This prevents accidentally exposing new sensitive headers added in the future
		headerList := []string{"X-Tid"}

		t := time.Now()
		body, _ := ctx.GetRawData()
		var header string
		for k, v := range ctx.Request.Header {
			if sliceContain(headerList, k) {
				header = header + k + ":" + strings.Join(v, ",") + ";"
			}
		}

		bodystr := ""
		// Skip logging file upload bodies to prevent:
		// 1. Excessive log storage consumption
		// 2. Performance degradation from logging large binary data
		// 3. Potential PII/sensitive data exposure
		contentType := ctx.Request.Header.Get("Content-Type")
		if !strings.Contains(contentType, "multipart/form-data") {
			bodystr = string(body)
		}
		common.Infof(ctx, "access log request, uri: %s, method: %s, header: %s, params: %s",
			ctx.Request.RequestURI,
			ctx.Request.Method,
			header,
			bodystr,
		)
		// Restore the request body since reading it consumes the stream
		// This ensures downstream handlers can still access the original request data
		ctx.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		blw := &CustomResponseWriter{body: bytes.NewBufferString(""), ResponseWriter: ctx.Writer}
		ctx.Writer = blw
		ctx.Next()

		// Log response time to help identify performance bottlenecks
		// and track service level indicators (SLIs) for reliability monitoring
		costtime := time.Since(t).Microseconds()
		common.Infof(ctx, "access log response, costtime: %dms, result: %s", costtime, blw.body.String())
	}
}
