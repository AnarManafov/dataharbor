package common

import (
	"strings"
	"testing"
)

// Secrets for the truncation sweep. Each is checked as every 8-byte window,
// so no prefix, suffix or middle part of one may survive any cut.
const (
	truncStrong    = "Zq8Wv3Lm5Tn2Rb7Kx4Hd"
	truncStrongNum = "918273645546372819"
	truncNested    = "Pf6Gs1Jc9Yw4Ue2Ao7Mi"
	truncWeak      = "Nb3Vx8Kq1Lr6Zt5Wy2Ec"
	truncJWT       = "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0cnVuYyJ9.Qs4Dk7Hm2Fp9Lx3Vb"
	truncUserPass  = "Rk2Tm7Wq4Zx9Cv1Bn6Lp"
	truncBearer    = "Gh5Jd8Ks3Fl1Pq7Wm4Xz"
	truncCLI       = "Ty9Ui4Op2As6Df8Gh3Jk"
)

var truncSecrets = []string{truncStrong, truncStrongNum, truncNested, truncWeak, truncJWT, truncUserPass, truncBearer, "dh_cli_" + truncCLI}

func assertNoSecretFragment(t *testing.T, cut int, out string) {
	t.Helper()
	for _, s := range truncSecrets {
		for i := 0; i+8 <= len(s); i++ {
			if strings.Contains(out, s[i:i+8]) {
				t.Fatalf("cut at %d: fragment %q of secret %q leaked: %s", cut, s[i:i+8], s, out)
			}
		}
	}
}

// TestRedactTruncatedBody_SweepJSON cuts a JSON body at every offset: no
// fragment of a secret may appear, whether it is recognised by key (strong,
// weak, nested subtree) or only by shape (JWT, URL userinfo, bearer, CLI
// token under a neutral key).
func TestRedactTruncatedBody_SweepJSON(t *testing.T) {
	body := `{"path":"/data","note":"visible-head","access_token":"` + truncStrong + `",` +
		`"pin":` + truncStrongNum + `,"credentials":{"user":"u","inner":["` + truncNested + `",{"x":"` + truncNested + `"}]},` +
		`"state":"` + truncWeak + `","code":200,` +
		`"items":[{"name":"a","link":"https://alice:` + truncUserPass + `@example.org/x"},` +
		`{"name":"b","auth_header":"Bearer ` + truncBearer + `"},{"name":"c","id":"` + truncJWT + `"},` +
		`{"name":"d","cli":"dh_cli_` + truncCLI + `"},{"size":12345,"ok":true,"n":null}],` +
		`"note2":"visible-tail"}`

	// The same JSON sent as text/plain (fetch() with a string body) or with no
	// Content-Type is recognised by its first byte and cut the same way.
	for _, ct := range []string{"application/json; charset=utf-8", "text/plain;charset=UTF-8", ""} {
		sawHead, sawTail := false, false
		for cut := 0; cut < len(body); cut++ {
			out := RedactTruncatedBody(ct, []byte(body[:cut]))
			assertNoSecretFragment(t, cut, out)
			sawHead = sawHead || strings.Contains(out, "visible-head")
			sawTail = sawTail || strings.Contains(out, `"ok":true`)
		}
		if !sawHead || !sawTail {
			t.Fatalf("%q: non-secret values never rendered (head %v, tail %v): the sweep proves nothing", ct, sawHead, sawTail)
		}
	}
}

// TestRedactTruncatedBody_SweepForm does the same for a form body.
func TestRedactTruncatedBody_SweepForm(t *testing.T) {
	body := "path=%2Fdata&note=visible-head&access_token=" + truncStrong + "&state=" + truncWeak +
		"&next=https://alice:" + truncUserPass + "@example.org/x&id=" + truncJWT +
		"&cli=dh_cli_" + truncCLI + "&note2=visible-tail&last=1"

	sawHead, sawTail := false, false
	for cut := 0; cut < len(body); cut++ {
		out := RedactTruncatedBody("application/x-www-form-urlencoded", []byte(body[:cut]))
		assertNoSecretFragment(t, cut, out)
		sawHead = sawHead || strings.Contains(out, "visible-head")
		sawTail = sawTail || strings.Contains(out, "visible-tail")
	}
	if !sawHead || !sawTail {
		t.Fatalf("non-secret values never rendered (head %v, tail %v)", sawHead, sawTail)
	}
}

func TestRedactTruncatedBody(t *testing.T) {
	tests := []struct {
		name, ct, in, want string
	}{
		{"json prefix keeps complete members", "application/json",
			`{"a":"x","b":[1,2,{"c":"yy`, `{"a":"x","b":[1,2,{` + TruncatedBodyMarker},
		{"number at the cut is dropped", "application/json",
			`{"a":1,"size":1234`, `{"a":1` + TruncatedBodyMarker},
		{"strong key redacted on sight", "application/json",
			`{"a":"x","refresh_token":"abc`, `{"a":"x","refresh_token":"[REDACTED]"` + TruncatedBodyMarker},
		{"strong key subtree", "application/json",
			`{"secret":{"x":1},"a":"y","b`, `{"secret":"[REDACTED]","a":"y"` + TruncatedBodyMarker},
		{"weak key with a string", "application/json",
			`{"state":"abc","n":2,"x`, `{"state":"[REDACTED]","n":2` + TruncatedBodyMarker},
		{"weak key with a number is shown", "application/json",
			`{"code":200,"x":"y","z`, `{"code":200,"x":"y"` + TruncatedBodyMarker},
		{"+json type", "application/problem+json",
			`[{"a":"b"},{"c`, `[{"a":"b"},{` + TruncatedBodyMarker},
		{"whole document then trailing data", "application/json",
			`{"a":"b"} {"c":"d"}`, `{"a":"b"}` + TruncatedBodyMarker},
		{"scalar top level", "application/json",
			`"abcdef`, `[more than 7 bytes of application/json, truncated, not logged]`},
		{"nothing complete", "application/json",
			`{"abc`, `{` + TruncatedBodyMarker},
		{"garbage", "application/json",
			`<html>`, `[more than 6 bytes of application/json, truncated, not logged]`},
		{"form cut to the last pair", "application/x-www-form-urlencoded",
			"a=1&password=hunter2&b=xy", "a=1&password=[REDACTED]" + TruncatedBodyMarker},
		{"form without a complete pair", "application/x-www-form-urlencoded",
			"password=hunt", "[more than 13 bytes of application/x-www-form-urlencoded, truncated, not logged]"},
		{"other type", "text/plain",
			"hello", "[more than 5 bytes of text/plain, truncated, not logged]"},
		{"JSON sent as text/plain", "text/plain;charset=UTF-8",
			`{"a":"x","refresh_token":"abc`, `{"a":"x","refresh_token":"[REDACTED]"` + TruncatedBodyMarker},
		{"JSON with no type", "",
			` [{"a":"b"},{"c`, `[{"a":"b"},{` + TruncatedBodyMarker},
		{"invalid JSON as text/plain", "text/plain",
			`{password=hunter2`, `{` + TruncatedBodyMarker},
		{"bad content type", "\x00bad",
			"hello", "[more than 5 bytes of unknown content type, truncated, not logged]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RedactTruncatedBody(tt.ct, []byte(tt.in))
			if got != tt.want {
				t.Fatalf("got  %s\nwant %s", got, tt.want)
			}
			if again := RedactText(got); again != got {
				t.Fatalf("not a fixed point of RedactText:\n%s\n%s", got, again)
			}
		})
	}
}

func TestIsRedactableBody(t *testing.T) {
	// The type alone decides for form and anything that mentions "json".
	for _, ct := range []string{
		"application/json", "application/json; charset=utf-8", "Application/JSON",
		"application/problem+json", "application/x-www-form-urlencoded", "jsonish; bad=",
	} {
		for _, start := range []string{"", "  ", "<html>", "{"} {
			if got := IsRedactableBody(ct, []byte(start)); got != BodyRedactable {
				t.Errorf("IsRedactableBody(%q, %q) = %v, want BodyRedactable", ct, start, got)
			}
		}
	}
	// Any other type (or none) is decided by the first non-space byte.
	for _, ct := range []string{
		"text/plain", "text/plain;charset=UTF-8", "text/html; charset=utf-8", "application/xml",
		"multipart/form-data; boundary=x", "application/octet-stream", "", "not a type",
	} {
		for start, want := range map[string]BodyDecision{
			"":             BodyUndecided,
			" \r\n\t":      BodyUndecided,
			`{"a":1}`:      BodyRedactable,
			" \n [1,2]":    BodyRedactable,
			"{not json":    BodyRedactable, // rendered only if it parses
			"hello {":      BodyNotRedactable,
			"<html>{}":     BodyNotRedactable,
			"\x00\x01\x02": BodyNotRedactable,
		} {
			if got := IsRedactableBody(ct, []byte(start)); got != want {
				t.Errorf("IsRedactableBody(%q, %q) = %v, want %v", ct, start, got, want)
			}
		}
	}
}

// TestRedactBody_AgreesWithIsRedactableBody checks that RedactBody renders
// content exactly when IsRedactableBody allows it, so the debug middleware's
// capture decision cannot drift from what RedactBody would log.
func TestRedactBody_AgreesWithIsRedactableBody(t *testing.T) {
	bodies := []string{`{"a":"b"}`, ` [1,2]`, `{bad`, `hello`, `<p>{"a":1}</p>`, "a=1&b=2", "   "}
	types := []string{"application/json", "application/x-www-form-urlencoded", "text/plain", "", "text/html"}
	for _, ct := range types {
		for _, b := range bodies {
			out := RedactBody(ct, []byte(b))
			sizeLine := strings.HasPrefix(out, "[") && strings.HasSuffix(out, " not logged]")
			if !sizeLine && IsRedactableBody(ct, []byte(b)) != BodyRedactable {
				t.Errorf("RedactBody(%q, %q) = %q, but IsRedactableBody does not allow it", ct, b, out)
			}
		}
	}
}

// Sentinels for bodies that are JSON without saying so in their type.
const (
	sniffSecret   = "Hy4Kd9Wm2Lq7Zx3Vb8Tn"
	invalidSecret = "Jr6Pw1Ns8Ck3Xf5Gm9Qa"
)

// TestRedactBody_JSONSentAsTextPlain: fetch() sends a JSON string as
// text/plain;charset=UTF-8, and a client may send no Content-Type. Such a
// body is logged, redacted exactly like application/json.
func TestRedactBody_JSONSentAsTextPlain(t *testing.T) {
	body := `{"note":"visible","client_secret":"` + sniffSecret + `","nested":{"password":"` + sniffSecret + `"}}`
	for _, ct := range []string{"text/plain;charset=UTF-8", "", "application/octet-stream"} {
		out := RedactBody(ct, []byte(body))
		if out != `{"client_secret":"[REDACTED]","nested":{"password":"[REDACTED]"},"note":"visible"}` {
			t.Errorf("RedactBody(%q) = %s", ct, out)
		}
	}
}

// TestRedactBody_InvalidJSONSentAsTextPlain: a body that starts like JSON but
// is not JSON is never logged raw, whole or cut.
func TestRedactBody_InvalidJSONSentAsTextPlain(t *testing.T) {
	bodies := []string{
		`{password=` + invalidSecret + `}`,
		`{ "note": "x", password: "` + invalidSecret + `" }`,
		`[` + invalidSecret + `]`,
		`{"note":"x"} trailing ` + invalidSecret,
	}
	for _, ct := range []string{"text/plain", ""} {
		for _, b := range bodies {
			out := RedactBody(ct, []byte(b))
			if strings.Contains(out, invalidSecret[:8]) || !strings.HasSuffix(out, " not logged]") {
				t.Errorf("whole RedactBody(%q, %q) = %s", ct, b, out)
			}
			for cut := 0; cut < len(b); cut++ {
				out := RedactTruncatedBody(ct, []byte(b[:cut]))
				for i := 0; i+8 <= len(invalidSecret); i++ {
					if strings.Contains(out, invalidSecret[i:i+8]) {
						t.Fatalf("cut %q at %d leaked %q: %s", b, cut, invalidSecret[i:i+8], out)
					}
				}
			}
		}
	}
}

func TestDescribeBody(t *testing.T) {
	if got := DescribeBody("application/octet-stream", 1<<20); got != "[1048576 bytes of application/octet-stream not logged]" {
		t.Fatal(got)
	}
	if got := DescribeBody("", 3); got != "[3 bytes of unknown content type not logged]" {
		t.Fatal(got)
	}
}
