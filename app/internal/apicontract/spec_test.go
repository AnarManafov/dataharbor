package apicontract

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const jsonCT = "application/json; charset=utf-8"

func loadFixture(t *testing.T) *Spec {
	t.Helper()
	s, err := Load(filepath.Join("testdata", "fixture.yaml"))
	require.NoError(t, err)
	return s
}

func validate(t *testing.T, s *Spec, method, path string, status int, ct, body string) error {
	t.Helper()
	return s.Validate(method, path, Response{Status: status, ContentType: ct, Body: []byte(body)})
}

func contractErr(t *testing.T, err error, kind ErrorKind) *Error {
	t.Helper()
	require.Error(t, err)
	var ce *Error
	require.True(t, errors.As(err, &ce), "want *apicontract.Error, got %T: %v", err, err)
	assert.Equal(t, kind, ce.Kind, ce.Error())
	return ce
}

func requireKind(t *testing.T, err error, kind ErrorKind) {
	t.Helper()
	_ = contractErr(t, err, kind)
}

func TestLoad_Errors(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "does-not-exist.yaml"))
	assert.Error(t, err)

	cases := map[string]string{
		"not yaml":         "openapi: [unterminated",
		"not a mapping":    "- a\n- b\n",
		"openapi 3.0":      "openapi: 3.0.3\npaths: {}\n",
		"missing paths":    "openapi: 3.1.0\n",
		"relative path":    "openapi: 3.1.0\npaths:\n  health:\n    get:\n      responses:\n        '200': {description: x}\n",
		"no responses":     "openapi: 3.1.0\npaths:\n  /h:\n    get: {}\n",
		"dangling ref":     "openapi: 3.1.0\npaths:\n  /h:\n    get:\n      responses:\n        '200':\n          description: x\n          content:\n            application/json:\n              schema: {$ref: '#/components/schemas/Missing'}\n",
		"external ref":     "openapi: 3.1.0\npaths:\n  /h:\n    get:\n      responses:\n        '200':\n          description: x\n          content:\n            application/json:\n              schema: {$ref: 'other.yaml#/X'}\n",
		"dangling resp":    "openapi: 3.1.0\npaths:\n  /h:\n    get:\n      responses:\n        '200': {$ref: '#/components/responses/Missing'}\n",
		"external resp":    "openapi: 3.1.0\npaths:\n  /h:\n    get:\n      responses:\n        '200': {$ref: 'other.yaml#/R'}\n",
		"invalid schema":   "openapi: 3.1.0\npaths:\n  /h:\n    get:\n      responses:\n        '200':\n          description: x\n          content:\n            application/json:\n              schema: {type: 42}\n",
		"response not map": "openapi: 3.1.0\npaths:\n  /h:\n    get:\n      responses:\n        '200': nope\n",
	}
	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Parse([]byte(doc))
			assert.Error(t, err)
		})
	}
}

func TestLoad_JSONInput(t *testing.T) {
	doc := `{"openapi":"3.1.0","paths":{"/h":{"get":{"responses":{"200":{"description":"x","content":{"application/json":{"schema":{"type":"integer"}}}}}}}}}`
	s, err := Parse([]byte(doc))
	require.NoError(t, err)
	assert.NoError(t, validate(t, s, "GET", "/h", 200, jsonCT, "7"))
	requireKind(t, validate(t, s, "GET", "/h", 200, jsonCT, `"7"`), KindSchema)
}

func TestOperations(t *testing.T) {
	s := loadFixture(t)
	assert.Equal(t, "3.1.0", s.Version)
	assert.Equal(t, []Operation{
		{Method: "GET", Path: "/any", OperationID: "anyText"},
		{Method: "POST", Path: "/archive", OperationID: "archive"},
		{Method: "GET", Path: "/download", OperationID: "download"},
		{Method: "GET", Path: "/health", OperationID: "health"},
		{Method: "GET", Path: "/items/limits", OperationID: "itemLimits"},
		{Method: "DELETE", Path: "/items/{itemId}", OperationID: "deleteItem"},
		{Method: "GET", Path: "/items/{itemId}", OperationID: "getItem"},
		{Method: "GET", Path: "/items/{itemId}/status", OperationID: "itemStatus"},
	}, s.Operations())
}

func TestFindOperation(t *testing.T) {
	s := loadFixture(t)
	cases := []struct {
		method, path, want string
		ok                 bool
	}{
		{"GET", "/health", "/health", true},
		{"get", "/health", "/health", true},
		{"POST", "/health", "", false},
		{"GET", "/health/", "", false},
		{"GET", "/items/limits", "/items/limits", true}, // literal wins over {itemId}
		{"GET", "/items/abc", "/items/{itemId}", true},
		{"DELETE", "/items/limits", "/items/{itemId}", true}, // no literal DELETE, template applies
		{"GET", "/items/abc/status", "/items/{itemId}/status", true},
		{"GET", "/items//status", "", false}, // empty parameter value
		{"GET", "/items/a/b/status", "", false},
		{"GET", "/nope", "", false},
	}
	for _, c := range cases {
		op, ok := s.FindOperation(c.method, c.path)
		assert.Equal(t, c.ok, ok, "%s %s", c.method, c.path)
		assert.Equal(t, c.want, op.Path, "%s %s", c.method, c.path)
	}
}

func TestGinPathToTemplate(t *testing.T) {
	assert.Equal(t, "/api/v1/xrd/upload/{uploadId}/chunk", GinPathToTemplate("/api/v1/xrd/upload/:uploadId/chunk"))
	assert.Equal(t, "/files/{path}", GinPathToTemplate("/files/*path"))
	assert.Equal(t, "/health", GinPathToTemplate("/health"))
	assert.Equal(t, "/", GinPathToTemplate("/"))
}

func TestDocumentedStatuses(t *testing.T) {
	s := loadFixture(t)
	assert.Equal(t, []string{"200", "404", "4XX"}, s.DocumentedStatuses("GET", "/items/x/status"))
	assert.Equal(t, []string{"200", "default"}, s.DocumentedStatuses("GET", "/download"))
	assert.Nil(t, s.DocumentedStatuses("GET", "/nope"))
	// Specification extensions ("x-...") are not statuses (nor paths, see TestOperations).
	assert.Equal(t, []string{"200"}, s.DocumentedStatuses("GET", "/health"))
}

func TestValidate_StatusLookup(t *testing.T) {
	s := loadFixture(t)
	const errBody = `{"code":404,"error":"Unknown"}`
	// Exact status, response slot that is a $ref to components.responses.
	assert.NoError(t, validate(t, s, "GET", "/items/x/status", 404, jsonCT, errBody))
	// Range fallback 4XX.
	assert.NoError(t, validate(t, s, "GET", "/items/x/status", 409, jsonCT, `{"code":409,"error":"c"}`))
	// default fallback.
	assert.NoError(t, validate(t, s, "GET", "/download", 500, jsonCT, `{"code":500,"error":"e"}`))
	// Undocumented status.
	e := contractErr(t, validate(t, s, "GET", "/items/x/status", 500, jsonCT, errBody), KindUndocumentedStatus)
	assert.Contains(t, e.Error(), "200, 404, 4XX")
	// Undocumented operation.
	requireKind(t, validate(t, s, "PUT", "/health", 200, jsonCT, `{}`), KindUndocumentedOperation)
}

func TestValidate_ContentType(t *testing.T) {
	s := loadFixture(t)
	ok := `{"code":200,"data":"ok"}`
	assert.NoError(t, validate(t, s, "GET", "/health", 200, "application/json", ok))
	assert.NoError(t, validate(t, s, "GET", "/health", 200, "Application/JSON; charset=UTF-8", ok))
	requireKind(t, validate(t, s, "GET", "/health", 200, "text/html; charset=utf-8", ok), KindContentType)
	requireKind(t, validate(t, s, "GET", "/health", 200, "", ok), KindContentType)
	requireKind(t, validate(t, s, "GET", "/health", 200, "application/json; =bad", ok), KindContentType)

	// Binary: only status and media type are checked, the body is ignored.
	assert.NoError(t, validate(t, s, "GET", "/download", 200, "application/octet-stream", "\x00\x01not json"))
	requireKind(t, validate(t, s, "GET", "/download", 200, jsonCT, `{}`), KindContentType)
	// Several documented media types.
	assert.NoError(t, validate(t, s, "POST", "/archive", 200, "application/gzip", "\x1f\x8b"))
	assert.NoError(t, validate(t, s, "POST", "/archive", 200, "application/x-tar", "tar"))
	requireKind(t, validate(t, s, "POST", "/archive", 200, "application/zip", "zip"), KindContentType)
	// Wildcards and +json.
	assert.NoError(t, validate(t, s, "GET", "/any", 200, "text/plain", "hello"))
	assert.NoError(t, validate(t, s, "GET", "/any", 200, "application/problem+json", ok))
	requireKind(t, validate(t, s, "GET", "/any", 200, "application/problem+json", `{"code":1}`), KindSchema)
	requireKind(t, validate(t, s, "GET", "/any", 200, "image/png", "x"), KindContentType)
}

func TestValidate_NoContent(t *testing.T) {
	s := loadFixture(t)
	assert.NoError(t, validate(t, s, "DELETE", "/items/x", 204, "", ""))
	requireKind(t, validate(t, s, "DELETE", "/items/x", 204, jsonCT, `{"a":1}`), KindUnexpectedBody)
}

func TestValidate_JSONBodies(t *testing.T) {
	s := loadFixture(t)
	assert.NoError(t, validate(t, s, "GET", "/health", 200, jsonCT, `{"code":200,"data":"ok","message":"success"}`))
	requireKind(t, validate(t, s, "GET", "/health", 200, jsonCT, `{"code":200,`), KindInvalidJSON)
	requireKind(t, validate(t, s, "GET", "/health", 200, jsonCT, ``), KindInvalidJSON)

	e := contractErr(t, validate(t, s, "GET", "/health", 200, jsonCT, `{"code":200,"data":42}`), KindSchema)
	require.Len(t, e.Details, 1)
	assert.Equal(t, "/data", e.Details[0].InstanceLocation)
	assert.Equal(t, "#/components/schemas/StringEnvelope/properties/data/type", e.Details[0].SchemaLocation)
	assert.NotEmpty(t, e.Details[0].Message)
	assert.Contains(t, e.Error(), "at /data (schema #/components/schemas/StringEnvelope/properties/data/type)")

	// Integer enum from YAML must compare equal to a JSON number.
	requireKind(t, validate(t, s, "GET", "/health", 200, jsonCT, `{"code":201,"data":"ok"}`), KindSchema)
	// Root-level failure (missing required) is reported at "/".
	e = contractErr(t, validate(t, s, "GET", "/health", 200, jsonCT, `{"data":"ok"}`), KindSchema)
	assert.Contains(t, e.Error(), "at / (schema")
	// additionalProperties: false through a components.responses $ref.
	requireKind(t, validate(t, s, "GET", "/items/x/status", 404, jsonCT, `{"code":404,"error":"x","extra":1}`), KindSchema)
}

// TestValidate_OpenAPI31Constructs proves empirically, through the same
// Load -> Validate path the probe uses, that the validation library implements
// the JSON Schema 2020-12 dialect of OpenAPI 3.1 rather than the 3.0 subset.
func TestValidate_OpenAPI31Constructs(t *testing.T) {
	s := loadFixture(t)
	const path = "/items/abc/status"
	cases := []struct {
		name  string
		body  string
		valid bool
	}{
		{"minimal", `{"id":"a","tags":["x"]}`, true},
		// type: [array, 'null']
		{"type array accepts null", `{"id":"a","tags":null}`, true},
		{"type array accepts array", `{"id":"a","tags":[]}`, true},
		{"type array rejects string", `{"id":"a","tags":"x"}`, false},
		{"type array rejects number", `{"id":"a","tags":1}`, false},
		// const
		{"const accepts 200", `{"id":"a","tags":null,"code":200}`, true},
		{"const accepts 200.0", `{"id":"a","tags":null,"code":200.0}`, true},
		{"const rejects 201", `{"id":"a","tags":null,"code":201}`, false},
		{"const rejects string", `{"id":"a","tags":null,"code":"200"}`, false},
		// prefixItems with items: false
		{"prefixItems accepts tuple", `{"id":"a","tags":null,"point":[1.5,"x"]}`, true},
		{"prefixItems accepts prefix", `{"id":"a","tags":null,"point":[1.5]}`, true},
		{"prefixItems rejects wrong order", `{"id":"a","tags":null,"point":["x",1.5]}`, false},
		{"prefixItems rejects extra item", `{"id":"a","tags":null,"point":[1.5,"x",3]}`, false},
		// $ref with a sibling keyword: the target alone (pattern) accepts
		// "abcdefgh", the sibling maxLength rejects it.
		{"ref target accepts short", `{"id":"a","tags":null,"label":"abc"}`, true},
		{"ref sibling rejects long", `{"id":"a","tags":null,"label":"abcdefgh"}`, false},
		{"ref target rejects pattern", `{"id":"a","tags":null,"label":"ABC"}`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validate(t, s, "GET", path, 200, jsonCT, c.body)
			if c.valid {
				assert.NoError(t, err)
			} else {
				requireKind(t, err, KindSchema)
			}
		})
	}
}

// TestValidate_MutatedSchemaFails mirrors the acceptance check of the probe:
// a deliberately wrong schema in the specification must turn a conforming
// response into a schema violation.
func TestValidate_MutatedSchemaFails(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "fixture.yaml"))
	require.NoError(t, err)
	const body = `{"code":200,"data":"ok"}`

	good, err := Parse(data)
	require.NoError(t, err)
	require.NoError(t, validate(t, good, "GET", "/health", 200, jsonCT, body))

	mutated := strings.Replace(string(data), "data: {type: string}", "data: {type: integer}", 1)
	require.NotEqual(t, string(data), mutated)
	bad, err := Parse([]byte(mutated))
	require.NoError(t, err)
	requireKind(t, validate(t, bad, "GET", "/health", 200, jsonCT, body), KindSchema)
}

func TestErrorString(t *testing.T) {
	e := &Error{Kind: KindSchema, Message: "m", Details: []Detail{
		{SchemaLocation: "#/a", InstanceLocation: "/x", Message: "one"},
		{SchemaLocation: "#/b", InstanceLocation: "", Message: "two"},
	}}
	assert.Equal(t, "schema violation: m: at /x (schema #/a): one; at / (schema #/b): two", e.Error())
}

func TestInstancePointerEscaping(t *testing.T) {
	assert.Equal(t, "", instancePointer(nil))
	assert.Equal(t, "/a~1b/c~0d/0", instancePointer([]string{"a/b", "c~d", "0"}))
}
