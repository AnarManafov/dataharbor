// Package apicontract loads an OpenAPI 3.1 specification and validates live
// HTTP responses against it.
//
// The specification is a single YAML (or JSON) document whose schemas refer to
// each other only through internal "#/components/..." references. Schemas are
// JSON Schema 2020-12 (the OpenAPI 3.1 dialect) and are validated with
// github.com/santhosh-tekuri/jsonschema/v6: the whole specification document
// is registered as one resource and every response schema is compiled by its
// JSON pointer inside that document, so internal $refs resolve naturally.
package apicontract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

// resourceURL is the identifier under which the specification document is
// registered with the schema compiler. It is never fetched.
const resourceURL = "https://apicontract.invalid/openapi.json"

// httpMethods are the operation keys of an OpenAPI Path Item Object.
var httpMethods = []string{"get", "put", "post", "delete", "options", "head", "patch", "trace"}

// Operation identifies one documented operation of the specification.
type Operation struct {
	Method      string // upper case, e.g. "GET"
	Path        string // path template as written in the spec, e.g. "/api/v1/xrd/upload/{uploadId}/status"
	OperationID string
}

// mediaType is one documented response media type.
type mediaType struct {
	name   string             // e.g. "application/json"
	schema *jsonschema.Schema // nil when the media type declares no schema
}

// response is one documented response of an operation.
type response struct {
	key   string // "200", "4XX" or "default"
	media []mediaType
}

// operation is the compiled form of an Operation.
type operation struct {
	Operation
	segments  []string // template split on "/"
	responses map[string]*response
}

// Spec is a loaded and compiled OpenAPI 3.1 specification.
type Spec struct {
	Version string // value of the "openapi" field
	ops     []*operation
}

// Load reads and compiles the specification at path.
func Load(path string) (*Spec, error) {
	data, err := os.ReadFile(path) //#nosec G304 -- the spec path is chosen by the operator running the tool or test
	if err != nil {
		return nil, fmt.Errorf("read spec: %w", err)
	}
	return Parse(data)
}

// Parse compiles a specification from YAML or JSON bytes. Every response
// schema is compiled eagerly, so a dangling $ref is reported here rather
// than at validation time.
func Parse(data []byte) (*Spec, error) {
	doc, err := decodeDocument(data)
	if err != nil {
		return nil, err
	}
	root, ok := doc.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("spec: top level is not a mapping")
	}
	version, _ := root["openapi"].(string)
	if !strings.HasPrefix(version, "3.1") {
		return nil, fmt.Errorf("spec: unsupported openapi version %q (want 3.1.x)", version)
	}

	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	// An empty loader map: only the registered specification document can be
	// resolved. External references fail at load time instead of being fetched.
	compiler.UseLoader(jsonschema.SchemeURLLoader{})
	if err := compiler.AddResource(resourceURL, doc); err != nil {
		return nil, fmt.Errorf("spec: register document: %w", err)
	}

	s := &Spec{Version: version}
	paths, _ := root["paths"].(map[string]any)
	if paths == nil {
		return nil, fmt.Errorf("spec: missing paths")
	}
	for _, tmpl := range sortedKeys(paths) {
		if isExtension(tmpl) {
			continue
		}
		if !strings.HasPrefix(tmpl, "/") {
			return nil, fmt.Errorf("spec: path %q does not start with /", tmpl)
		}
		item, itemPtr, err := resolveRef(root, paths[tmpl], "/paths/"+escapePointer(tmpl))
		if err != nil {
			return nil, err
		}
		for _, m := range httpMethods {
			rawOp, ok := item[m].(map[string]any)
			if !ok {
				continue
			}
			op, err := compileOperation(compiler, root, rawOp, itemPtr+"/"+m)
			if err != nil {
				return nil, fmt.Errorf("spec: %s %s: %w", strings.ToUpper(m), tmpl, err)
			}
			op.Method = strings.ToUpper(m)
			op.Path = tmpl
			op.segments = strings.Split(tmpl, "/")
			s.ops = append(s.ops, op)
		}
	}
	return s, nil
}

func compileOperation(c *jsonschema.Compiler, root map[string]any, raw map[string]any, ptr string) (*operation, error) {
	op := &operation{responses: map[string]*response{}}
	op.OperationID, _ = raw["operationId"].(string)
	responses, ok := raw["responses"].(map[string]any)
	if !ok || len(responses) == 0 {
		return nil, fmt.Errorf("no responses documented")
	}
	for _, key := range sortedKeys(responses) {
		if isExtension(key) {
			continue
		}
		resp, respPtr, err := resolveRef(root, responses[key], ptr+"/responses/"+escapePointer(key))
		if err != nil {
			return nil, err
		}
		r := &response{key: strings.ToUpper(key)}
		content, _ := resp["content"].(map[string]any)
		for _, name := range sortedKeys(content) {
			mt := mediaType{name: strings.ToLower(name)}
			mObj, _ := content[name].(map[string]any)
			if _, has := mObj["schema"]; has {
				loc := resourceURL + "#" + respPtr + "/content/" + escapePointer(name) + "/schema"
				sch, err := c.Compile(loc)
				if err != nil {
					return nil, fmt.Errorf("response %s %s: compile schema: %w", key, name, err)
				}
				mt.schema = sch
			}
			r.media = append(r.media, mt)
		}
		op.responses[r.key] = r
	}
	return op, nil
}

// Operations lists every documented (method, path template) pair, sorted by
// path then method.
func (s *Spec) Operations() []Operation {
	out := make([]Operation, 0, len(s.ops))
	for _, op := range s.ops {
		out = append(out, op.Operation)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Method < out[j].Method
	})
	return out
}

// FindOperation returns the operation documenting method on the concrete
// request path (no query string). Literal path segments win over templated
// ones, so "/upload/limits" matches before "/upload/{uploadId}".
func (s *Spec) FindOperation(method, path string) (Operation, bool) {
	op := s.find(method, path)
	if op == nil {
		return Operation{}, false
	}
	return op.Operation, true
}

func (s *Spec) find(method, path string) *operation {
	method = strings.ToUpper(method)
	segs := strings.Split(path, "/")
	var best *operation
	bestParams := -1
	for _, op := range s.ops {
		if op.Method != method || len(op.segments) != len(segs) {
			continue
		}
		params, ok := matchSegments(op.segments, segs)
		if !ok {
			continue
		}
		if best == nil || params < bestParams {
			best, bestParams = op, params
		}
	}
	return best
}

// matchSegments reports whether concrete matches template and how many
// templated segments were used.
func matchSegments(template, concrete []string) (int, bool) {
	params := 0
	for i, t := range template {
		if isParam(t) {
			if concrete[i] == "" {
				return 0, false
			}
			params++
			continue
		}
		if t != concrete[i] {
			return 0, false
		}
	}
	return params, true
}

// isExtension reports whether a key is a specification extension ("x-...").
func isExtension(key string) bool {
	return strings.HasPrefix(key, "x-")
}

func isParam(seg string) bool {
	return len(seg) > 2 && seg[0] == '{' && seg[len(seg)-1] == '}'
}

// GinPathToTemplate converts a Gin route path ("/upload/:uploadId/chunk",
// "/files/*path") to OpenAPI template form ("/upload/{uploadId}/chunk").
func GinPathToTemplate(p string) string {
	segs := strings.Split(p, "/")
	for i, s := range segs {
		if len(s) > 1 && (s[0] == ':' || s[0] == '*') {
			segs[i] = "{" + s[1:] + "}"
		}
	}
	return strings.Join(segs, "/")
}

// decodeDocument parses YAML (a superset of JSON) and converts it into the
// representation the schema library expects: string-keyed maps and
// json.Number values, obtained by a round trip through the library's own
// JSON decoder.
func decodeDocument(data []byte) (any, error) {
	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("spec: parse: %w", err)
	}
	norm, err := normalize(raw)
	if err != nil {
		return nil, err
	}
	js, err := json.Marshal(norm)
	if err != nil {
		return nil, fmt.Errorf("spec: convert to JSON: %w", err)
	}
	return jsonschema.UnmarshalJSON(bytes.NewReader(js))
}

// normalize stringifies map keys (yaml.v3 yields int keys for an unquoted
// "200:") so the document is JSON-marshalable.
func normalize(v any) (any, error) {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			n, err := normalize(val)
			if err != nil {
				return nil, err
			}
			out[k] = n
		}
		return out, nil
	case map[any]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			n, err := normalize(val)
			if err != nil {
				return nil, err
			}
			out[keyString(k)] = n
		}
		return out, nil
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			n, err := normalize(val)
			if err != nil {
				return nil, err
			}
			out[i] = n
		}
		return out, nil
	default:
		return v, nil
	}
}

func keyString(k any) string {
	switch t := k.(type) {
	case string:
		return t
	case int:
		return strconv.Itoa(t)
	default:
		return fmt.Sprint(t)
	}
}

// resolveRef follows internal "$ref"s of non-schema objects (Path Item and
// Response Objects) and returns the target mapping with its JSON pointer.
func resolveRef(root map[string]any, v any, ptr string) (map[string]any, string, error) {
	for range 16 {
		m, ok := v.(map[string]any)
		if !ok {
			return nil, "", fmt.Errorf("spec: %s is not a mapping", ptr)
		}
		ref, ok := m["$ref"].(string)
		if !ok {
			return m, ptr, nil
		}
		if !strings.HasPrefix(ref, "#/") {
			return nil, "", fmt.Errorf("spec: %s: only internal $refs are supported, got %q", ptr, ref)
		}
		target, err := lookupPointer(root, ref[1:])
		if err != nil {
			return nil, "", fmt.Errorf("spec: %s: %w", ptr, err)
		}
		v, ptr = target, ref[1:]
	}
	return nil, "", fmt.Errorf("spec: %s: $ref chain too deep", ptr)
}

func lookupPointer(root map[string]any, ptr string) (any, error) {
	var cur any = root
	for _, tok := range strings.Split(strings.TrimPrefix(ptr, "/"), "/") {
		tok = strings.ReplaceAll(strings.ReplaceAll(tok, "~1", "/"), "~0", "~")
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("$ref %q: not found", "#"+ptr)
		}
		if cur, ok = m[tok]; !ok {
			return nil, fmt.Errorf("$ref %q: not found", "#"+ptr)
		}
	}
	return cur, nil
}

// escapePointer escapes one JSON pointer token (RFC 6901). Braces are left
// as is; the library percent-decodes the fragment and accepts them.
func escapePointer(tok string) string {
	return strings.ReplaceAll(strings.ReplaceAll(tok, "~", "~0"), "/", "~1")
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
