package middleware

import (
	"bytes"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/AnarManafov/dataharbor/app/common"
	"github.com/AnarManafov/dataharbor/app/config"
	"github.com/gin-gonic/gin"
)

// DebugRequestBody is a middleware that logs request and response bodies.
// This is useful for debugging API issues, especially for authentication
// endpoints, but should only be enabled in development environments.
// Headers and bodies are passed through common.Redact* and credentials are
// never logged.
//
// Memory per request is bounded by limit (bytes, normalised by
// config.EffectiveDebugBodyLogLimit: the default if limit <= 0, at most
// config.MaxDebugBodyLogLimit), whatever the body size:
//
//   - Routes in streaming (file downloads, upload chunks; matched by their
//     registered method and pattern, c.FullPath()) keep the request line,
//     status and headers, but their bodies are never read, wrapped or kept.
//   - Other routes capture at most limit bytes of each body, and only of the
//     bodies common.RedactBody can render (common.IsRedactableBody: form, a
//     type that mentions "json", or any body that starts with '{' or '[').
//     The request body is peeked (sniffLen bytes to classify it when its
//     Content-Type does not, then at most limit+1 bytes if it is loggable)
//     and handed on unchanged; the response is written straight through
//     while at most limit bytes are kept. Capture buffers grow on demand. A
//     body longer than limit is logged via common.RedactTruncatedBody.
//
// streaming may be nil (no streaming routes) and may be filled after the
// middleware is installed: gin installs global middleware before routes are
// registered.
func DebugRequestBody(limit int, streaming *RouteSet) gin.HandlerFunc {
	limit = config.EffectiveDebugBodyLogLimit(limit)
	return func(c *gin.Context) {
		// Skip high-volume and static asset paths entirely.
		if shouldSkipPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		startTime := time.Now()

		common.Debugf(c, "Request: %s %s", c.Request.Method, c.Request.URL.Path)
		common.Debugf(c, "Request Headers: %v", common.RedactHeaders(c.Request.Header))

		captureBody := !streaming.Has(c.Request.Method, c.FullPath())

		var blw *bodyLogWriter
		if captureBody {
			logRequestBody(c, limit)
			blw = &bodyLogWriter{ResponseWriter: c.Writer, limit: limit}
			c.Writer = blw
		}

		c.Next()

		duration := time.Since(startTime)

		common.Debugf(c, "Response Status: %d", c.Writer.Status())
		common.Debugf(c, "Response Duration: %v", duration)
		common.Debugf(c, "Response Headers: %v", common.RedactHeaders(c.Writer.Header()))

		if blw != nil && blw.total > 0 {
			common.Debugf(c, "Response Body: %s", blw.render())
		}
	}
}

// sniffLen is how many bytes of a request body are read at a time to
// classify it when its Content-Type alone does not (common.BodyUndecided).
const sniffLen = 512

// logRequestBody logs the request body without consuming it: the bytes read
// ahead are put back in front of the rest of the body, so the handler sees
// the full body unchanged. It runs before the handler so the body is logged
// even when a later middleware rejects the request without reading it (401).
//
// A body whose Content-Type decides (form, or a type mentioning "json") is
// read ahead up to limit+1 bytes. Any other body is first read sniffLen bytes
// at a time until its first non-whitespace byte is seen: if that is '{' or
// '[' the body is read on up to limit+1 bytes and logged like JSON; if not,
// reading stops and only its declared size is logged.
func logRequestBody(c *gin.Context, limit int) {
	body := c.Request.Body
	if body == nil || body == http.NoBody {
		return
	}
	ct := c.Request.Header.Get("Content-Type")

	// buf grows on demand (bytes.Buffer.ReadFrom), never to more than
	// limit+1 bytes of content.
	lr := io.LimitReader(body, int64(limit)+1)
	var buf bytes.Buffer
	var err error
	decision := common.IsRedactableBody(ct, nil)
	for decision == common.BodyUndecided {
		var n int64
		n, err = buf.ReadFrom(io.LimitReader(lr, sniffLen))
		decision = common.IsRedactableBody(ct, buf.Bytes())
		if err != nil || n < sniffLen {
			break // read error, or the (limited) body ended
		}
	}
	if decision == common.BodyRedactable && err == nil {
		_, err = buf.ReadFrom(lr)
	}
	peeked := buf.Bytes()

	if len(peeked) > 0 || err != nil {
		var rest io.Reader = body
		if err != nil {
			rest = errReader{err} // hand the read error on to the handler
		}
		c.Request.Body = &peekedBody{Reader: io.MultiReader(bytes.NewReader(peeked), rest), closer: body}
	}

	switch {
	case decision != common.BodyRedactable:
		// Not loggable (or whitespace only): its size, when declared.
		if c.Request.ContentLength > 0 {
			common.Debugf(c, "Request Body: %s", common.DescribeBody(ct, c.Request.ContentLength))
		}
	case len(peeked) > limit:
		common.Debugf(c, "Request Body: %s", common.RedactTruncatedBody(ct, peeked[:limit]))
	case err != nil && len(peeked) > 0:
		// The read failed part-way (client gone, short body): what was read
		// is cut like a truncated body, so its last unit may be partial.
		common.Debugf(c, "Request Body: %s", common.RedactTruncatedBody(ct, peeked))
	case len(peeked) > 0:
		common.Debugf(c, "Request Body: %s", common.RedactBody(ct, peeked))
	}
}

// peekedBody replays the bytes read ahead, then the rest of the original
// body; Close closes the original body.
type peekedBody struct {
	io.Reader
	closer io.Closer
}

func (b *peekedBody) Close() error { return b.closer.Close() }

type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }

// bodyLogWriter passes the response straight through and keeps at most limit
// bytes of it, and only of a body common.RedactBody can render: the
// Content-Type read at the first write decides for form and JSON types;
// otherwise the first non-whitespace byte written does ('{' or '[' is kept as
// JSON, anything else is not kept). It counts every byte written. The capture
// buffer grows on demand, never past limit.
type bodyLogWriter struct {
	gin.ResponseWriter
	limit       int
	started     bool
	decision    common.BodyDecision
	contentType string
	body        []byte // at most limit bytes
	total       int64
}

// Write writes the response and keeps the start of it for logging.
func (w *bodyLogWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	if r := w.room(); r > 0 && n > 0 {
		w.keep(b[:min(n, r)])
	}
	w.total += int64(n)
	return n, err
}

// WriteString is overridden too: the promoted one would bypass the capture.
func (w *bodyLogWriter) WriteString(s string) (int, error) {
	n, err := w.ResponseWriter.WriteString(s)
	if r := w.room(); r > 0 && n > 0 {
		w.keep([]byte(s[:min(n, r)]))
	}
	w.total += int64(n)
	return n, err
}

// room returns how many more bytes may be kept: none once the body is known
// not to be loggable or the limit is reached.
func (w *bodyLogWriter) room() int {
	if !w.started {
		w.started = true
		w.contentType = w.Header().Get("Content-Type")
		w.decision = common.IsRedactableBody(w.contentType, nil)
	}
	if w.decision == common.BodyNotRedactable {
		return 0
	}
	return w.limit - len(w.body)
}

// keep appends p (len(p) <= room()) to the capture, classifying the body
// first while it is undecided.
func (w *bodyLogWriter) keep(p []byte) {
	if w.decision == common.BodyUndecided {
		// What was kept so far is whitespace, so p alone decides.
		w.decision = common.IsRedactableBody(w.contentType, p)
		if w.decision == common.BodyNotRedactable {
			w.body = nil
			return
		}
	}
	w.body = appendCapped(w.body, p, w.limit)
	if w.decision == common.BodyUndecided && len(w.body) >= w.limit {
		// limit bytes of whitespace: nothing RedactBody would render.
		w.decision = common.BodyNotRedactable
		w.body = nil
	}
}

// minCaptureCap is the first capacity of a capture buffer (unless the limit
// is smaller).
const minCaptureCap = 512

// appendCapped appends p to dst, growing dst's capacity on demand but never
// past limit; len(dst)+len(p) must not exceed limit. Plain append could round
// the capacity up past limit.
func appendCapped(dst, p []byte, limit int) []byte {
	if need := len(dst) + len(p); need > cap(dst) {
		grown := make([]byte, len(dst), min(max(2*cap(dst), need, minCaptureCap), limit))
		copy(grown, dst)
		dst = grown
	}
	return append(dst, p...)
}

// render returns the loggable form of what was written.
func (w *bodyLogWriter) render() string {
	switch {
	case w.decision != common.BodyRedactable:
		return common.DescribeBody(w.contentType, w.total)
	case w.total > int64(len(w.body)):
		return common.RedactTruncatedBody(w.contentType, w.body)
	default:
		return common.RedactBody(w.contentType, w.body)
	}
}

// shouldSkipPath reports paths that are not logged at all: health checks
// (high volume) and static frontend assets.
func shouldSkipPath(path string) bool {
	skipPaths := []string{
		"/health",      // Health check endpoints
		"/favicon.ico", // Browser favicon requests
		"/static/",     // Static assets
		"/assets/",     // Static assets
	}
	for _, p := range skipPaths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// RouteSet is a set of registered routes, keyed by method and gin route
// pattern (the value of c.FullPath(), e.g. "/api/v1/xrd/upload/:uploadId/chunk").
// It is safe for concurrent use; a nil *RouteSet is empty.
type RouteSet struct {
	mu     sync.RWMutex
	routes map[string]struct{}
}

// NewRouteSet returns an empty RouteSet.
func NewRouteSet() *RouteSet { return &RouteSet{routes: map[string]struct{}{}} }

// Add adds the route method fullPath.
func (s *RouteSet) Add(method, fullPath string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.routes[method+" "+fullPath] = struct{}{}
}

// Has reports whether method fullPath is in the set. An unmatched route
// (fullPath "") is never in it.
func (s *RouteSet) Has(method, fullPath string) bool {
	if s == nil || fullPath == "" {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.routes[method+" "+fullPath]
	return ok
}

// List returns the routes as sorted "METHOD /pattern" strings.
func (s *RouteSet) List() []string {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.routes))
	for k := range s.routes {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
