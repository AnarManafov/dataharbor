package apiprobe

import (
	"io"
	"net/url"
	"sort"
	"strings"
	"unicode/utf8"
)

const redacted = "[REDACTED]"

// redactor replaces every occurrence of a secret (and its URL-encoded forms)
// in text before it is printed. It is a defensive net: the probe never
// formats the cookie on purpose, but error strings from net/http, response
// bodies and Set-Cookie echoes could still carry it.
type redactor struct {
	secrets []string // longest first, so an encoded form is not half-replaced
}

func newRedactor(secret string) *redactor {
	r := &redactor{}
	if secret == "" {
		return r
	}
	seen := map[string]bool{}
	for _, s := range []string{secret, url.QueryEscape(secret), url.PathEscape(secret)} {
		if s != "" && !seen[s] {
			seen[s] = true
			r.secrets = append(r.secrets, s)
		}
	}
	sort.Slice(r.secrets, func(i, j int) bool { return len(r.secrets[i]) > len(r.secrets[j]) })
	return r
}

func (r *redactor) String(s string) string {
	for _, sec := range r.secrets {
		s = strings.ReplaceAll(s, sec, redacted)
	}
	return s
}

// redactingWriter passes every Write through the redactor. Callers write whole
// lines, so a secret is never split across two writes.
type redactingWriter struct {
	w io.Writer
	r *redactor
}

func (rw redactingWriter) Write(p []byte) (int, error) {
	if _, err := io.WriteString(rw.w, rw.r.String(string(p))); err != nil {
		return 0, err
	}
	return len(p), nil
}

// oneLine makes text safe to embed in a single output line: control
// characters and invalid UTF-8 are escaped, and the result is capped at max
// bytes (on a rune boundary). Redact before calling it, never after: a cap
// could otherwise cut a secret in half and leak its prefix.
func oneLine(s string, max int) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == utf8.RuneError && size == 1:
			b.WriteString(`\x`)
			b.WriteString(hex2(s[i]))
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r < 0x20 || r == 0x7f: // single-byte ASCII control character
			b.WriteString(`\x`)
			b.WriteString(hex2(s[i]))
		default:
			b.WriteString(s[i : i+size])
		}
		i += size
	}
	out := b.String()
	if max > 0 && len(out) > max {
		cut := max
		for cut > 0 && !utf8.RuneStart(out[cut]) {
			cut--
		}
		out = out[:cut] + "..."
	}
	return out
}

func hex2(c byte) string {
	const digits = "0123456789abcdef"
	return string([]byte{digits[c>>4], digits[c&0x0f]})
}
