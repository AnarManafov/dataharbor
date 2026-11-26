package common

// Log redaction helpers (GH-82): no log line, at any level, may contain a
// credential. These helpers scrub UNKNOWN secrets, by key name (headers, JSON
// keys, query/form parameters) and by value shape (auth schemes, JWTs,
// dh_cli_ tokens, PEM blocks, key=value fragments, URL userinfo). They are meant to be
// applied at the call site before anything request- or response-derived is
// logged; the redacting zap core in logger.go applies RedactText again to
// every message and string-like field as a defence-in-depth net.
//
// Over-redaction is acceptable, under-redaction is not.

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

// RedactedPlaceholder replaces every redacted value.
const RedactedPlaceholder = "[REDACTED]"

// ---------------------------------------------------------------------------
// Sensitive key classification
// ---------------------------------------------------------------------------

// sensitiveKeyParts: a key is sensitive if its normalized form contains any of
// these.
var sensitiveKeyParts = []string{
	"token", "secret", "password", "passwd", "passphrase",
	"authorization", "cookie", "credential", "apikey", "accesskey",
	"session", "assertion", "signature", "privatekey",
	"authcode", "devicecode", "usercode", "csrf", "xsrf",
	"jwt", "bearer", "ticket", "saml",
}

// sensitiveKeyExact: a key is sensitive if its normalized form equals one of
// these (too short / too generic for substring matching).
var sensitiveKeyExact = map[string]bool{
	"pwd": true, "pass": true, "pin": true, "otp": true,
	"nonce": true, "codeverifier": true, "codechallenge": true,
	"auth": true, "key": true, "sig": true,
}

// weakKeyExact: sensitive, but in JSON (and Go-struct style "key: value" free
// text) only when the value is a string, because the API envelope carries a
// numeric "code" and status words are common. Always redacted in query
// strings, form bodies and key=value text.
var weakKeyExact = map[string]bool{"code": true, "state": true}

// normalizeKey lower-cases ASCII letters and drops '-', '_', '.' and spaces so
// "X-Api-Key", "x_api_key" and "xApiKey" all normalize to "xapikey".
func normalizeKey(k string) string {
	b := make([]byte, 0, len(k))
	for i := 0; i < len(k); i++ {
		c := k[i]
		switch {
		case c >= 'A' && c <= 'Z':
			b = append(b, c+('a'-'A'))
		case c == '-' || c == '_' || c == '.' || c == ' ':
		default:
			b = append(b, c)
		}
	}
	return string(b)
}

// isStrongKey reports whether k is sensitive regardless of its value type.
func isStrongKey(k string) bool {
	n := normalizeKey(k)
	if sensitiveKeyExact[n] {
		return true
	}
	for _, p := range sensitiveKeyParts {
		if strings.Contains(n, p) {
			return true
		}
	}
	return false
}

func isWeakKey(k string) bool { return weakKeyExact[normalizeKey(k)] }

// isSensitiveKey is the full rule used for headers, query strings and forms.
func isSensitiveKey(k string) bool { return isStrongKey(k) || isWeakKey(k) }

// authSchemes are the Authorization scheme words kept in front of a redacted
// credential.
var authSchemes = map[string]bool{
	"bearer": true, "basic": true, "token": true, "dpop": true,
	"negotiate": true, "ntlm": true, "digest": true, "mac": true,
	"hoba": true, "aws4-hmac-sha256": true,
}

func isSchemeWord(v string) bool { return authSchemes[strings.ToLower(v)] }

// isFingerprint reports whether v is exactly a Fingerprint output; such values
// are safe to log and are deliberately not redacted.
func isFingerprint(v string) bool {
	if len(v) != len(fpPrefix)+fpHexLen || !strings.HasPrefix(v, fpPrefix) {
		return false
	}
	for i := len(fpPrefix); i < len(v); i++ {
		c := v[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// keepValue reports values that never need redacting: empty, already
// redacted, or a fingerprint.
func keepValue(v string) bool {
	return v == "" || v == RedactedPlaceholder || isFingerprint(v)
}

// ---------------------------------------------------------------------------
// Headers
// ---------------------------------------------------------------------------

// RedactHeaders returns a redacted copy of h (h itself is never modified).
// Authorization/Proxy-Authorization keep a known scheme word only, Cookie keeps
// cookie names, Set-Cookie keeps the name and attributes, any header whose
// name is sensitive is fully redacted, and every other value goes through
// RedactText. Header names are matched case-insensitively.
func RedactHeaders(h http.Header) map[string][]string {
	out := make(map[string][]string, len(h))
	for name, vals := range h {
		if vals == nil {
			out[name] = nil
			continue
		}
		lname := strings.ToLower(strings.TrimSpace(name))
		red := make([]string, len(vals))
		for i, v := range vals {
			switch {
			case lname == "authorization" || lname == "proxy-authorization":
				red[i] = redactAuthValue(v)
			case lname == "cookie":
				red[i] = redactCookieValue(v)
			case lname == "set-cookie":
				red[i] = redactSetCookieValue(v)
			case isSensitiveKey(lname):
				red[i] = RedactedPlaceholder
			default:
				red[i] = RedactText(v)
			}
		}
		out[name] = red
	}
	return out
}

func redactAuthValue(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	scheme, _, found := strings.Cut(v, " ")
	if found && isSchemeWord(scheme) {
		return scheme + " " + RedactedPlaceholder
	}
	return RedactedPlaceholder
}

func redactCookieValue(v string) string {
	parts := strings.Split(v, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		name, _, found := strings.Cut(p, "=")
		if !found {
			out = append(out, RedactedPlaceholder)
			continue
		}
		out = append(out, RedactText(strings.TrimSpace(name))+"="+RedactedPlaceholder)
	}
	return strings.Join(out, "; ")
}

func redactSetCookieValue(v string) string {
	parts := strings.Split(v, ";")
	out := make([]string, 0, len(parts))
	for i, p := range parts {
		p = strings.TrimSpace(p)
		if i == 0 {
			name, _, found := strings.Cut(p, "=")
			if !found {
				out = append(out, RedactedPlaceholder)
			} else {
				out = append(out, RedactText(strings.TrimSpace(name))+"="+RedactedPlaceholder)
			}
			continue
		}
		if p != "" {
			out = append(out, RedactText(p))
		}
	}
	return strings.Join(out, "; ")
}

// ---------------------------------------------------------------------------
// URLs and query strings
// ---------------------------------------------------------------------------

// RedactURL redacts the values of sensitive query parameters and of sensitive
// key=value pairs in the fragment (keys are kept), the userinfo password, and
// finally applies RedactText. The original encoding, key order, host and path
// are preserved: net/url is used only to validate, never to re-serialize. A
// string that does not parse as a URL goes through RedactText alone.
func RedactURL(raw string) string {
	if _, err := url.Parse(raw); err != nil {
		return RedactText(raw)
	}
	rest, frag, hasFrag := strings.Cut(raw, "#")
	base, query, hasQuery := strings.Cut(rest, "?")
	var b strings.Builder
	b.WriteString(base)
	if hasQuery {
		b.WriteByte('?')
		b.WriteString(redactPairs(query))
	}
	if hasFrag {
		b.WriteByte('#')
		b.WriteString(redactPairs(frag))
	}
	return RedactText(b.String())
}

// redactPairs redacts values of sensitive keys in an &-separated key=value
// list, keeping the raw encoding of everything else.
func redactPairs(s string) string {
	parts := strings.Split(s, "&")
	for i, p := range parts {
		k, v, found := strings.Cut(p, "=")
		if !found || keepValue(v) {
			continue
		}
		dk, err := url.QueryUnescape(k)
		if err != nil {
			dk = k
		}
		if isSensitiveKey(dk) || isSensitiveKey(k) {
			parts[i] = k + "=" + RedactedPlaceholder
		}
	}
	return strings.Join(parts, "&")
}

// ---------------------------------------------------------------------------
// Bodies
// ---------------------------------------------------------------------------

// RedactBody returns a loggable rendering of an HTTP body. JSON is redacted by
// key recursively and re-emitted as compact JSON (object key order is
// normalized), application/x-www-form-urlencoded is redacted like a query
// string, an empty body yields "", and everything else (HTML, plain text, XML,
// multipart, binary, unparseable or scalar JSON) FAILS CLOSED as
// "[<n> bytes of <media type> not logged]".
func RedactBody(contentType string, body []byte) string {
	if len(body) == 0 {
		return ""
	}
	mt, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mt = ""
	}
	if mt == "application/x-www-form-urlencoded" {
		return RedactText(redactPairs(string(body)))
	}
	trimmed := bytes.TrimSpace(body)
	looksJSON := len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[')
	if strings.Contains(strings.ToLower(contentType), "json") || looksJSON {
		if out, ok := redactJSONDocument(body, false); ok {
			return out
		}
	}
	return describeBody(mt, len(body))
}

func describeBody(mt string, n int) string {
	if mt == "" || len(mt) > 64 || !safeMediaType(mt) {
		mt = "unknown content type"
	}
	return fmt.Sprintf("[%d bytes of %s not logged]", n, mt)
}

func safeMediaType(mt string) bool {
	for i := 0; i < len(mt); i++ {
		c := mt[i]
		ok := c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || strings.IndexByte("/.+-_", c) >= 0
		if !ok {
			return false
		}
	}
	return true
}

// redactJSONDocument decodes exactly one JSON value, redacts it and re-encodes
// it compactly. Unless allowScalar is set the top-level value must be an
// object or an array.
func redactJSONDocument(body []byte, allowScalar bool) (string, bool) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return "", false
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return "", false // trailing data
	}
	if !allowScalar {
		switch v.(type) {
		case map[string]any, []any:
		default:
			return "", false
		}
	}
	return encodeJSON(redactJSONValue(v))
}

func encodeJSON(v any) (string, bool) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "", false
	}
	return strings.TrimRight(buf.String(), "\n"), true
}

func redactJSONValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[RedactText(k)] = redactJSONMember(k, val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = redactJSONValue(e)
		}
		return out
	case string:
		return RedactText(t)
	default: // json.Number, bool, nil
		return v
	}
}

func redactJSONMember(k string, val any) any {
	switch {
	case isStrongKey(k):
		switch t := val.(type) {
		case nil, bool:
			return val
		case string:
			if keepValue(t) {
				return t
			}
		}
		return RedactedPlaceholder // strings, numbers and whole subtrees
	case isWeakKey(k):
		if s, ok := val.(string); ok {
			if keepValue(s) {
				return s
			}
			return RedactedPlaceholder
		}
		return redactJSONValue(val)
	default:
		return redactJSONValue(val)
	}
}

// ---------------------------------------------------------------------------
// Free text
// ---------------------------------------------------------------------------

var (
	// (1) one or more scheme words followed by a credential. Repeating the
	// scheme group handles "token Bearer abc" without leaving "abc" exposed.
	reAuthScheme = regexp.MustCompile(`(?i)\b((?:(?:bearer|basic|token|dpop|negotiate|ntlm)\s+)+)([A-Za-z0-9\-._~+/=]+)`)
	// Cookie / Set-Cookie header lines in free text: the rest of the line.
	reCookieLine = regexp.MustCompile(`(?i)\b((?:set-)?cookie[ \t]*:[ \t]*)([^\r\n]+)`)
	// (2) JWS (3 segments) and JWE (5 segments); signature may be empty.
	reJWT = regexp.MustCompile(`eyJ[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+(?:\.[A-Za-z0-9_\-]*){0,3}`)
	// (3) DataHarbor CLI tokens.
	reCLIToken = regexp.MustCompile(`dh_cli_[A-Za-z0-9_\-]+`)
	// PEM blocks ("-----BEGIN <label>-----" ... "-----END <label>-----"),
	// multi-line or with JSON-escaped "\n"; an unterminated block runs to the
	// end of the input. RE2 has no back-references, so labels are not paired.
	rePEM = regexp.MustCompile(`(?s)-----BEGIN [^\r\n-]*-----.*?(?:-----END [^\r\n-]*-----|\z)`)
	// (4) key=value, the key optionally preceded by percent-escapes (so
	// "%3Fcode%3D..." yields key "code") and "=" possibly percent-encoded
	// (once or more). Group 2 is the separator: a literal "=" may have
	// horizontal whitespace around it ("password = x").
	reKV = regexp.MustCompile(`(?:%(?:25)*[0-9A-Fa-f]{2})*([A-Za-z0-9_.\-]+)([ \t]*=[ \t]*|%(?:25)*3[Dd])`)
	// (5) "key": / \"key\": JSON-ish members.
	// Keys are identifier-like so a quoted URL followed by ':' (as in Go's
	// `Get "https://...": dial tcp`) is not taken for a key.
	reJSONKey = regexp.MustCompile(`(\\*)"([A-Za-z0-9_.\-$@]{1,128})\\*"[ \t\r\n]*:[ \t\r\n]*`)
	// Go struct / YAML style "Key: value" and "Key:value".
	reColon = regexp.MustCompile(`\b([A-Za-z][A-Za-z0-9_\-]*)[ \t]*:[ \t]*`)
	// (6) URL userinfo: "scheme://user:password@" or "scheme://user@". The
	// userinfo cannot contain "/", "?" or "#", so an "@" in the path, query or
	// fragment is not taken for one; "mailto:a@b" has no "//".
	reUserinfo = regexp.MustCompile(`(?i)\b([a-z][a-z0-9+.\-]*://)([^\s/?#"'<>]+)@`)
)

// RedactText scrubs credential-shaped substrings from free text (error
// strings, messages, header values). Passes, in order: PEM blocks (whole
// block), auth-scheme credentials ("Bearer x" -> "Bearer [REDACTED]"), Cookie
// header lines, JWTs, dh_cli_ tokens (prefix kept), key=value pairs with a
// sensitive key (including percent-encoded ones, and "key = value" for strong
// keys), "key":"value" JSON-ish members (also with escaped quotes), Go-struct
// style "Key: value" with a sensitive key, and URL userinfo (the password, or
// the whole userinfo when it has no password). It is idempotent and safe on
// arbitrary bytes.
func RedactText(s string) string {
	if s == "" {
		return s
	}
	// First, so the marker words and base64 lines are not split up by the
	// key-based passes ("privateKey: -----BEGIN ...").
	s = rePEM.ReplaceAllLiteralString(s, RedactedPlaceholder)
	s = scrub(s, reAuthScheme, authSchemeRepl)
	s = scrub(s, reCookieLine, func(s string, m []int) (int, int, string, bool) {
		v := s[m[4]:m[5]]
		if v == RedactedPlaceholder || v[0] == '[' || v[0] == '{' {
			// A bracketed value is a Go map/slice rendering such as
			// "Cookie:[a=b] User-Agent:[curl]": the "Key: value" pass redacts
			// just the group, so the headers after it stay readable.
			return 0, 0, "", false
		}
		return m[4], m[5], RedactedPlaceholder, true
	})
	s = reJWT.ReplaceAllLiteralString(s, RedactedPlaceholder)
	s = reCLIToken.ReplaceAllLiteralString(s, "dh_cli_"+RedactedPlaceholder)
	s = scrub(s, reKV, kvRepl)
	s = scrub(s, reJSONKey, jsonKeyRepl)
	s = scrub(s, reColon, colonRepl)
	s = scrub(s, reUserinfo, userinfoRepl)
	return s
}

// userinfoRepl redacts the password of "user:password@", keeping the user, or
// the whole userinfo when there is no password ("https://<token>@host").
func userinfoRepl(s string, m []int) (int, int, string, bool) {
	start, end := m[4], m[5]
	if i := strings.IndexByte(s[start:end], ':'); i >= 0 {
		start += i + 1
	}
	if v := s[start:end]; v == "" || v == RedactedPlaceholder {
		return 0, 0, "", false
	}
	return start, end, RedactedPlaceholder, true
}

// RedactError returns RedactText(err.Error()), or "<nil>" for a nil error.
func RedactError(err error) string {
	if err == nil {
		return "<nil>"
	}
	return RedactText(err.Error())
}

// scrub walks the matches of re in s. For each match fn may return a span
// [start, end) to replace with repl. A match that starts inside an already
// replaced span is skipped; a match fn declines does not consume anything, so
// later matches inside its value are still examined (nested secrets).
func scrub(s string, re *regexp.Regexp, fn func(s string, m []int) (start, end int, repl string, ok bool)) string {
	ms := re.FindAllStringSubmatchIndex(s, -1)
	if len(ms) == 0 {
		return s
	}
	var b strings.Builder
	pos := 0
	changed := false
	for _, m := range ms {
		if m[0] < pos {
			continue
		}
		start, end, repl, ok := fn(s, m)
		if !ok || start < pos || end < start {
			continue
		}
		b.WriteString(s[pos:start])
		b.WriteString(repl)
		pos = end
		changed = true
	}
	if !changed {
		return s
	}
	b.WriteString(s[pos:])
	return b.String()
}

func authSchemeRepl(s string, m []int) (int, int, string, bool) {
	prefix := s[m[2]:m[3]]
	cred := strings.TrimRight(s[m[4]:m[5]], ".")
	if cred == "" || isSchemeWord(cred) || cred[0] == '=' {
		// A leading "=" is a separator, not a credential: "token = x" is a
		// key=value pair, left for the key=value pass.
		return 0, 0, "", false
	}
	end := m[4] + len(cred)
	if cred == "dh_cli_" && strings.HasPrefix(s[end:], RedactedPlaceholder) {
		return 0, 0, "", false // already redacted CLI token (keeps idempotence)
	}
	sep := end
	for sep < len(s) && (s[sep] == ' ' || s[sep] == '\t') {
		sep++
	}
	if sep < len(s) && (s[sep] == ':' || s[sep] == '=') && isSensitiveKey(cred) {
		// "TokenType:Bearer RefreshToken:x", "Bearer password = x" – the word
		// after the scheme is a sensitive key; leave it for the key-based
		// passes to redact its value.
		return 0, 0, "", false
	}
	if onlyTokenScheme(prefix) && !looksLikeCredential(cred) {
		// "token is expired", "token expired." – prose, not a credential.
		return 0, 0, "", false
	}
	return m[4], end, RedactedPlaceholder, true
}

// onlyTokenScheme reports whether every scheme word in prefix is "token"
// (the one scheme word that is also common in prose).
func onlyTokenScheme(prefix string) bool {
	for _, w := range strings.Fields(prefix) {
		if !strings.EqualFold(w, "token") {
			return false
		}
	}
	return true
}

func looksLikeCredential(v string) bool {
	if len(v) >= 20 {
		return true
	}
	return strings.ContainsAny(v, "0123456789-._~+/=")
}

func kvRepl(s string, m []int) (int, int, string, bool) {
	key := s[m[2]:m[3]]
	if !isSensitiveKey(key) {
		return 0, 0, "", false
	}
	sep := s[m[4]:m[5]]
	if sep != "=" && sep[0] != '%' && !isStrongKey(key) {
		// Spaced form only for strong keys: "code = Unavailable" (gRPC) and
		// "state = running" are status words, OIDC parameters come tight.
		return 0, 0, "", false
	}
	start, end := scanEqValue(s, m[1], encodingDepth(sep))
	v := s[start:end]
	if keepValue(v) || isSchemeWord(v) {
		return 0, 0, "", false
	}
	if rest, ok := strings.CutPrefix(v, RedactedPlaceholder); ok && strings.Trim(rest, "])}") == "" {
		// Already redacted, followed only by the closing brackets of a Go
		// map/slice rendering ("session=[REDACTED]]"). Replacing it would eat
		// a bracket and unbalance the group for the "Key: value" pass.
		return 0, 0, "", false
	}
	return start, end, RedactedPlaceholder, true
}

// encodingDepth returns how many times a key=value separator was
// percent-encoded: 0 for a literal "=" (with or without surrounding
// whitespace), 1 for "%3D", 2 for "%253D", and so on.
func encodingDepth(sep string) int {
	if sep == "" || sep[0] != '%' {
		return 0
	}
	return 1 + strings.Count(sep, "25")
}

// scanEqValue returns the span of the value starting at i in key=value text,
// for a key whose separator was found at encoding depth depth. A quoted value
// spans to its closing quote (quotes excluded); otherwise the value ends at
// a literal &, whitespace, quote, #, ;, \, < or >, or at one of those
// percent-encoded at most depth times. A deeper-encoded delimiter is part of
// the value: in literal text (depth 0) "access_token=a%26b" is one secret,
// while in a once-encoded nested URL "%3Fcode%3Da%26state%3Db" the "%26"
// ends the code. Erring this way over-redacts, never leaks.
func scanEqValue(s string, i, depth int) (int, int) {
	if i < len(s) && (s[i] == '"' || s[i] == '\'') {
		return scanQuoted(s, i)
	}
	j := i
	for j < len(s) {
		c := s[j]
		if c <= ' ' || c == 0x7f || strings.IndexByte("&#;\"'<>\\", c) >= 0 {
			break
		}
		if c == '%' && depth > 0 {
			if d := encodedTerminatorDepth(s[j:]); d > 0 && d <= depth {
				break
			}
		}
		j++
	}
	return i, j
}

// scanQuoted: s[i] is a quote; returns the content span up to the matching
// unescaped quote, or to the end of the line if there is none.
func scanQuoted(s string, i int) (int, int) {
	q := s[i]
	start := i + 1
	for j := start; j < len(s); j++ {
		switch s[j] {
		case '\\':
			j++
		case q, '\n':
			return start, j
		}
	}
	return start, len(s)
}

// encodedTerminatorDepth reports whether s starts with %XX, possibly encoded
// further as %25XX, %2525XX, ..., where XX is a value delimiter. It returns
// the encoding depth (1 for %XX, 2 for %25XX, ...), or 0 if s does not start
// with an encoded delimiter.
func encodedTerminatorDepth(s string) int {
	if len(s) < 3 || s[0] != '%' {
		return 0
	}
	i := 1
	depth := 1
	for i+3 < len(s) && s[i] == '2' && s[i+1] == '5' && isHex(s[i+2]) && isHex(s[i+3]) {
		i += 2
		depth++
	}
	if i+2 > len(s) {
		return 0
	}
	switch strings.ToLower(s[i : i+2]) {
	case "26", "23", "22", "27", "20", "3b", "3c", "3e", "0a", "0d", "09", "5c":
		return depth
	}
	return 0
}

func isHex(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

func jsonKeyRepl(s string, m []int) (int, int, string, bool) {
	key := s[m[4]:m[5]]
	strong, weak := isStrongKey(key), isWeakKey(key)
	if !strong && !weak {
		return 0, 0, "", false
	}
	i := m[1]
	k := 0
	for i+k < len(s) && s[i+k] == '\\' {
		k++
	}
	if i+k < len(s) && s[i+k] == '"' {
		start, end := scanJSONString(s, i+k+1, k)
		if keepValue(s[start:end]) {
			return 0, 0, "", false
		}
		return start, end, RedactedPlaceholder, true
	}
	if k > 0 || !strong {
		return 0, 0, "", false
	}
	start, end := scanBare(s, i, ",}]")
	switch s[start:end] {
	case "", "null", "true", "false", RedactedPlaceholder:
		return 0, 0, "", false
	}
	return start, end, RedactedPlaceholder, true
}

// scanJSONString scans string content starting at start whose closing quote
// is preceded by exactly k backslashes (k = escaping depth; 0 for plain JSON).
func scanJSONString(s string, start, k int) (int, int) {
	for j := start; j < len(s); j++ {
		if s[j] == '\n' {
			return start, j
		}
		if s[j] != '"' {
			continue
		}
		c := 0
		for j-1-c >= start && s[j-1-c] == '\\' {
			c++
		}
		if (k == 0 && c%2 == 0) || (k > 0 && c == k) {
			return start, j - k
		}
	}
	return start, len(s)
}

// scanBare scans an unquoted value: a balanced {...}/[...] group (quote
// aware), or a run up to whitespace or one of stop.
func scanBare(s string, i int, stop string) (int, int) {
	if i < len(s) && (s[i] == '{' || s[i] == '[') {
		depth := 0
		inStr := false
		for j := i; j < len(s); j++ {
			c := s[j]
			if inStr {
				switch c {
				case '\\':
					j++
				case '"':
					inStr = false
				}
				continue
			}
			switch c {
			case '"':
				inStr = true
			case '{', '[':
				depth++
			case '}', ']':
				depth--
				if depth == 0 {
					return i, j + 1
				}
			}
		}
		return i, len(s)
	}
	j := i
	for j < len(s) && s[j] > ' ' && s[j] != 0x7f && strings.IndexByte(stop, s[j]) < 0 {
		j++
	}
	return i, j
}

func colonRepl(s string, m []int) (int, int, string, bool) {
	key := s[m[2]:m[3]]
	strong, weak := isStrongKey(key), isWeakKey(key)
	if !strong && !weak {
		return 0, 0, "", false
	}
	i := m[1]
	if i >= len(s) {
		return 0, 0, "", false
	}
	c := s[i]
	quoted := c == '"' || c == '\''
	grouped := c == '[' || c == '{'
	if !strong && !quoted && !grouped {
		return 0, 0, "", false // "status code: 500"
	}
	var start, end int
	if quoted {
		start, end = scanQuoted(s, i)
	} else {
		start, end = scanBare(s, i, ",}])\"';&<>")
	}
	v := s[start:end]
	if keepValue(v) || isSchemeWord(v) || strings.HasPrefix(v, "//") {
		return 0, 0, "", false
	}
	if grouped && groupAlreadyRedacted(v) {
		return 0, 0, "", false
	}
	return start, end, RedactedPlaceholder, true
}

// groupAlreadyRedacted reports whether a bracketed value is the output of
// RedactHeaders for a credential header, e.g. "[Bearer [REDACTED]]" or
// "[a=[REDACTED]; b=[REDACTED]]". Such a group is kept so the scheme and the
// cookie names survive the logger's second pass. The check is strict: every
// ";"-separated part must be the placeholder itself, "<scheme> [REDACTED]" or
// "<name>=[REDACTED]". Anything else, such as a half-redacted raw header, is
// not recognised and the whole group is redacted.
func groupAlreadyRedacted(v string) bool {
	if len(v) < 2 || v[0] != '[' || v[len(v)-1] != ']' {
		return false
	}
	for _, part := range strings.Split(v[1:len(v)-1], ";") {
		part = strings.TrimSpace(part)
		if part == RedactedPlaceholder {
			continue
		}
		if scheme, rest, ok := strings.Cut(part, " "); ok && isSchemeWord(scheme) && rest == RedactedPlaceholder {
			continue
		}
		name, val, ok := strings.Cut(part, "=")
		if !ok || val != RedactedPlaceholder || name == "" || strings.ContainsAny(name, " \t[]{}\"'") {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// Fingerprint
// ---------------------------------------------------------------------------

const (
	fpPrefix = "fp:"
	fpHexLen = 12
)

var (
	fpKeyOnce sync.Once
	fpKey     []byte
)

func fingerprintKey() []byte {
	fpKeyOnce.Do(func() {
		k := make([]byte, 32)
		// Since Go 1.24 crypto/rand.Read never returns an error (it aborts the
		// process instead), so the error is not actionable.
		_, _ = rand.Read(k)
		fpKey = k
	})
	return fpKey
}

// Fingerprint returns a short, non-reversible tag ("fp:" + 12 hex chars) for
// correlating log lines that refer to the same secret, without logging it.
// It is HMAC-SHA256 keyed with a random per-process key, so low-entropy
// values (passwords, OAuth codes) cannot be brute-forced from logs. As a
// consequence fingerprints are stable only within one process: they do NOT
// correlate across restarts or between instances. The output never contains
// a 4+ character substring of the secret (a different HMAC domain is tried
// if it would; "fp:unavailable" if none fits). An empty secret yields "".
// Fingerprint values are recognised and left alone by the Redact* helpers.
func Fingerprint(secret string) string {
	if secret == "" {
		return ""
	}
	key := fingerprintKey()
	for round := 0; round < 256; round++ {
		mac := hmac.New(sha256.New, key)
		mac.Write([]byte{byte(round)})
		mac.Write([]byte(secret))
		out := fpPrefix + hex.EncodeToString(mac.Sum(nil))[:fpHexLen]
		if !sharesSubstring(out, secret, 4) {
			return out
		}
	}
	return fpPrefix + "unavailable"
}

// sharesSubstring reports whether any n-byte substring of a occurs in b.
func sharesSubstring(a, b string, n int) bool {
	for i := 0; i+n <= len(a); i++ {
		if strings.Contains(b, a[i:i+n]) {
			return true
		}
	}
	return false
}
