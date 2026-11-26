package route

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// openAPISpecPath is the specification, relative to this package directory.
var openAPISpecPath = filepath.Join("..", "..", "api", "openapi.yaml")

// frontendRoutes are the routes SetupRouter registers to serve the built SPA.
// They deliver static files, not API responses, so the specification does not
// describe them. Every other registered route must appear in the specification.
var frontendRoutes = map[string]bool{
	"GET /assets/*filepath":  true,
	"HEAD /assets/*filepath": true,
	"GET /favicon.ico":       true,
	"HEAD /favicon.ico":      true,
	"GET /config.json":       true,
	"HEAD /config.json":      true,
}

// ginParam matches a gin path parameter (":name" or "*name") as a whole segment.
var ginParam = regexp.MustCompile(`/[:*]([^/]+)`)

// ginToOpenAPIPath converts gin's "/upload/:uploadId/chunk" into the
// OpenAPI form "/upload/{uploadId}/chunk".
func ginToOpenAPIPath(path string) string {
	return ginParam.ReplaceAllString(path, "/{$1}")
}

// registeredOperations returns "METHOD /path" for every API route that
// SetupRouter registers, with paths in OpenAPI form.
func registeredOperations(t *testing.T) []string {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetupRouter(r)

	var ops []string
	for _, route := range r.Routes() {
		key := route.Method + " " + route.Path
		if frontendRoutes[key] {
			continue
		}
		ops = append(ops, route.Method+" "+ginToOpenAPIPath(route.Path))
	}
	sort.Strings(ops)
	return ops
}

// specOperations returns "METHOD /path" for every operation in the specification.
func specOperations(t *testing.T) []string {
	t.Helper()
	raw, err := os.ReadFile(openAPISpecPath)
	require.NoError(t, err, "the OpenAPI specification must exist at %s", openAPISpecPath)

	var doc struct {
		OpenAPI string                    `yaml:"openapi"`
		Paths   map[string]map[string]any `yaml:"paths"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &doc))
	require.True(t, strings.HasPrefix(doc.OpenAPI, "3.1."), "expected OpenAPI 3.1, got %q", doc.OpenAPI)

	var ops []string
	for path, item := range doc.Paths {
		for key := range item {
			if isOpenAPIMethod(key) {
				ops = append(ops, strings.ToUpper(key)+" "+path)
			}
		}
	}
	sort.Strings(ops)
	return ops
}

// TestOpenAPISpecMatchesRegisteredRoutes fails when a route is registered
// without being described in api/openapi.yaml, or the specification describes
// a route that is not registered.
func TestOpenAPISpecMatchesRegisteredRoutes(t *testing.T) {
	registered := registeredOperations(t)
	documented := specOperations(t)

	require.NotEmpty(t, registered, "no routes registered; the test setup is broken")

	inSpec := make(map[string]bool, len(documented))
	for _, op := range documented {
		inSpec[op] = true
	}
	inRouter := make(map[string]bool, len(registered))
	for _, op := range registered {
		inRouter[op] = true
	}

	for _, op := range registered {
		assert.True(t, inSpec[op], "route %q is registered in route/routes.go but missing from api/openapi.yaml", op)
	}
	for _, op := range documented {
		assert.True(t, inRouter[op], "operation %q is in api/openapi.yaml but not registered in route/routes.go", op)
	}
}

// TestFrontendRoutesAreRegistered keeps the exclusion list honest: an entry
// that no longer matches a registered route must be removed.
func TestFrontendRoutesAreRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetupRouter(r)

	registered := make(map[string]bool)
	for _, route := range r.Routes() {
		registered[route.Method+" "+route.Path] = true
	}
	for key := range frontendRoutes {
		assert.True(t, registered[key], "frontendRoutes lists %q, which is not a registered route", key)
	}
}

func TestGinToOpenAPIPath(t *testing.T) {
	cases := map[string]string{
		"/health":                         "/health",
		"/api/v1/xrd/upload/:uploadId":    "/api/v1/xrd/upload/{uploadId}",
		"/api/v1/xrd/upload/:id/chunk":    "/api/v1/xrd/upload/{id}/chunk",
		"/assets/*filepath":               "/assets/{filepath}",
		"/api/v1/:a/:b":                   "/api/v1/{a}/{b}",
		"/api/v1/xrd/ls/paged":            "/api/v1/xrd/ls/paged",
		"/api/v1/xrd/upload/:uploadId/x:": "/api/v1/xrd/upload/{uploadId}/x:",
	}
	for in, want := range cases {
		assert.Equal(t, want, ginToOpenAPIPath(in), in)
	}
}
