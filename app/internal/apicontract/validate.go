package apicontract

import (
	"bytes"
	"fmt"
	"mime"
	"sort"
	"strconv"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// ErrorKind classifies a contract mismatch.
type ErrorKind string

// Mismatch kinds reported by Validate.
const (
	KindUndocumentedOperation ErrorKind = "undocumented operation"
	KindUndocumentedStatus    ErrorKind = "undocumented status"
	KindContentType           ErrorKind = "content type mismatch"
	KindUnexpectedBody        ErrorKind = "unexpected body"
	KindInvalidJSON           ErrorKind = "invalid JSON"
	KindSchema                ErrorKind = "schema violation"
)

// Detail is one leaf failure of a schema validation.
type Detail struct {
	SchemaLocation   string // absolute keyword location, e.g. "#/components/schemas/Health/properties/code/type"
	InstanceLocation string // JSON pointer into the response body, e.g. "/data/0/name"
	Message          string
}

// String renders the detail on one line.
func (d Detail) String() string {
	inst := d.InstanceLocation
	if inst == "" {
		inst = "/"
	}
	return fmt.Sprintf("at %s (schema %s): %s", inst, d.SchemaLocation, d.Message)
}

// Error describes why a response does not conform to the specification.
type Error struct {
	Kind    ErrorKind
	Method  string
	Path    string
	Status  int
	Message string
	Details []Detail // populated for KindSchema
}

func (e *Error) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %s", e.Kind, e.Message)
	for i, d := range e.Details {
		if i == 0 {
			b.WriteString(": ")
		} else {
			b.WriteString("; ")
		}
		b.WriteString(d.String())
	}
	return b.String()
}

// Response is the observed response to validate.
type Response struct {
	Status      int
	ContentType string // raw Content-Type header value
	Body        []byte
}

// Validate checks a response to method on the concrete path against the
// specification. It returns nil when the response conforms and *Error
// otherwise.
//
// Rules:
//   - the operation must be documented, and the status must be documented
//     exactly, by range ("4XX") or by "default";
//   - when the response documents content, the Content-Type must match one of
//     the documented media types (exact, "type/*" or "*/*", parameters such
//     as charset ignored); a response without documented content must have an
//     empty body;
//   - JSON media types ("application/json", "*+json") with a schema must have
//     a body that validates against it; other media types are not inspected.
func (s *Spec) Validate(method, path string, r Response) error {
	fail := func(kind ErrorKind, format string, args ...any) error {
		return &Error{Kind: kind, Method: strings.ToUpper(method), Path: path, Status: r.Status, Message: fmt.Sprintf(format, args...)}
	}
	op := s.find(method, path)
	if op == nil {
		return fail(KindUndocumentedOperation, "%s %s is not in the specification", strings.ToUpper(method), path)
	}
	resp := op.lookupResponse(r.Status)
	if resp == nil {
		return fail(KindUndocumentedStatus, "status %d is not documented for %s %s (documented: %s)",
			r.Status, op.Method, op.Path, strings.Join(op.documentedStatuses(), ", "))
	}
	if len(resp.media) == 0 {
		if len(bytes.TrimSpace(r.Body)) != 0 {
			return fail(KindUnexpectedBody, "response %s documents no content but the body has %d bytes", resp.key, len(r.Body))
		}
		return nil
	}
	got, _, err := mime.ParseMediaType(r.ContentType)
	if err != nil {
		return fail(KindContentType, "unparsable Content-Type %q (documented: %s)", r.ContentType, resp.mediaNames())
	}
	mt := resp.lookupMedia(got)
	if mt == nil {
		return fail(KindContentType, "Content-Type %q is not documented for status %s (documented: %s)", got, resp.key, resp.mediaNames())
	}
	if !isJSONMedia(got) {
		return nil
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(r.Body))
	if err != nil {
		return fail(KindInvalidJSON, "body is not valid JSON: %v", err)
	}
	if mt.schema == nil {
		return nil
	}
	if err := mt.schema.Validate(inst); err != nil {
		e := fail(KindSchema, "body does not match the schema for %s %s %s %s", op.Method, op.Path, resp.key, mt.name).(*Error)
		e.Details = flatten(err)
		return e
	}
	return nil
}

func (op *operation) lookupResponse(status int) *response {
	code := strconv.Itoa(status)
	if r, ok := op.responses[code]; ok {
		return r
	}
	if len(code) == 3 {
		if r, ok := op.responses[code[:1]+"XX"]; ok {
			return r
		}
	}
	return op.responses["DEFAULT"]
}

func (op *operation) documentedStatuses() []string {
	keys := make([]string, 0, len(op.responses))
	for k := range op.responses {
		if k == "DEFAULT" {
			k = "default"
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// DocumentedStatuses lists the response keys ("200", "4XX", "default")
// documented for the operation matching method and the concrete path.
func (s *Spec) DocumentedStatuses(method, path string) []string {
	op := s.find(method, path)
	if op == nil {
		return nil
	}
	return op.documentedStatuses()
}

func (r *response) lookupMedia(got string) *mediaType {
	got = strings.ToLower(got)
	major, _, _ := strings.Cut(got, "/")
	var wildcard, anyType *mediaType
	for i := range r.media {
		mt := &r.media[i]
		name, _, _ := strings.Cut(mt.name, ";")
		name = strings.TrimSpace(name)
		switch name {
		case got:
			return mt
		case major + "/*":
			wildcard = mt
		case "*/*":
			anyType = mt
		}
	}
	if wildcard != nil {
		return wildcard
	}
	return anyType
}

func (r *response) mediaNames() string {
	names := make([]string, len(r.media))
	for i, m := range r.media {
		names[i] = m.name
	}
	return strings.Join(names, ", ")
}

func isJSONMedia(mt string) bool {
	return mt == "application/json" || strings.HasSuffix(mt, "+json")
}

var printer = message.NewPrinter(language.English)

// flatten turns a validation error tree into its leaf failures, with schema
// locations relative to the specification document.
func flatten(err error) []Detail {
	ve, ok := err.(*jsonschema.ValidationError)
	if !ok {
		return []Detail{{Message: err.Error()}}
	}
	var out []Detail
	var walk func(*jsonschema.ValidationError)
	walk = func(v *jsonschema.ValidationError) {
		if len(v.Causes) == 0 {
			loc := strings.TrimPrefix(v.SchemaURL, resourceURL)
			for _, kw := range v.ErrorKind.KeywordPath() {
				loc += "/" + escapePointer(kw)
			}
			out = append(out, Detail{
				SchemaLocation:   loc,
				InstanceLocation: instancePointer(v.InstanceLocation),
				Message:          v.ErrorKind.LocalizedString(printer),
			})
			return
		}
		for _, c := range v.Causes {
			walk(c)
		}
	}
	walk(ve)
	return out
}

func instancePointer(loc []string) string {
	if len(loc) == 0 {
		return ""
	}
	parts := make([]string, len(loc))
	for i, p := range loc {
		parts[i] = escapePointer(p)
	}
	return "/" + strings.Join(parts, "/")
}
