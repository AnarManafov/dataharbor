package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// Distinctive sentinels: if one of these appears in output, a secret leaked.
const (
	sentA = "SentinelAlphaQx7Zk9"
	sentB = "SentinelBravoWv3Ym2"
	sentC = "SentinelCharlieRt5Pn8"
	// A JWT-shaped value (header {"alg":"HS256"}).
	sentJWT = "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJTZW50aW5lbEp3dCJ9.SentinelJwtSig4Lm"
	sentCLI = "dh_cli_SentinelCliTok8Hd2"
)

var allSentinels = []string{sentA, sentB, sentC, "SentinelJwtSig4Lm", "SentinelCliTok8Hd2", "eyJzdWIiOiJTZW50aW5lbEp3dCJ9"}

func assertNoSentinel(t *testing.T, out string) {
	t.Helper()
	for _, s := range allSentinels {
		if strings.Contains(out, s) {
			t.Fatalf("secret %q leaked in output: %s", s, out)
		}
	}
}

func assertContainsAll(t *testing.T, out string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Fatalf("expected %q to survive redaction, got: %s", w, out)
		}
	}
}

func TestRedactSensitiveKeyRule(t *testing.T) {
	sensitive := []string{
		"token", "access_token", "Refresh-Token", "ID_TOKEN_HINT", "client_secret", "password",
		"passwd", "pwd", "Authorization", "proxy-authorization", "cookie", "set-cookie",
		"credentials", "apikey", "api_key", "X-Api-Key", "session_id", "SessionID", "session_secret",
		"client_assertion", "X-Amz-Signature", "private_key", "privateKey", "code", "state", "nonce",
		"code_verifier", "codeVerifier", "code_challenge", "auth", "key", "X-CSRF-Token",
	}
	for _, k := range sensitive {
		if !isSensitiveKey(k) {
			t.Errorf("%q should be sensitive", k)
		}
	}
	notSensitive := []string{"user", "path", "status", "redirect_uri", "client_id", "scope", "error", "message", "status_code", "uploadId", "size"}
	for _, k := range notSensitive {
		if isSensitiveKey(k) {
			t.Errorf("%q should not be sensitive", k)
		}
	}
}

func TestRedactHeaders(t *testing.T) {
	tests := []struct {
		name string
		in   http.Header
		key  string
		want []string
	}{
		{"bearer canonical", http.Header{"Authorization": {"Bearer " + sentA}}, "Authorization", []string{"Bearer [REDACTED]"}},
		{"basic lowercase name", http.Header{"authorization": {"Basic " + sentA}}, "authorization", []string{"Basic [REDACTED]"}},
		{"no scheme", http.Header{"AUTHORIZATION": {sentA}}, "AUTHORIZATION", []string{"[REDACTED]"}},
		{"unknown first word", http.Header{"Authorization": {sentA + " " + sentB}}, "Authorization", []string{"[REDACTED]"}},
		{"proxy auth", http.Header{"Proxy-Authorization": {"Bearer " + sentA}}, "Proxy-Authorization", []string{"Bearer [REDACTED]"}},
		{"cookies one line", http.Header{"COOKIE": {"sid=" + sentA + "; theme=" + sentB + ";lang=" + sentC}}, "COOKIE",
			[]string{"sid=[REDACTED]; theme=[REDACTED]; lang=[REDACTED]"}},
		{"cookies several lines", http.Header{"Cookie": {"a=" + sentA, "b=" + sentB + "; c=" + sentC}}, "Cookie",
			[]string{"a=[REDACTED]", "b=[REDACTED]; c=[REDACTED]"}},
		{"set-cookie attributes", http.Header{"Set-Cookie": {"dh_session=" + sentA + "; Path=/; HttpOnly; Secure; Max-Age=3600; SameSite=Lax"}}, "Set-Cookie",
			[]string{"dh_session=[REDACTED]; Path=/; HttpOnly; Secure; Max-Age=3600; SameSite=Lax"}},
		{"sensitive custom header", http.Header{"x-api-key": {sentA}}, "x-api-key", []string{"[REDACTED]"}},
		{"csrf header", http.Header{"X-Csrf-Token": {sentA}}, "X-Csrf-Token", []string{"[REDACTED]"}},
		{"location with code and state", http.Header{"Location": {"https://app.example/oidc-callback?code=" + sentA + "&state=" + sentB + "&iss=https%3A%2F%2Fidp"}}, "Location",
			[]string{"https://app.example/oidc-callback?code=[REDACTED]&state=[REDACTED]&iss=https%3A%2F%2Fidp"}},
		{"referer with id_token_hint", http.Header{"Referer": {"https://idp.example/logout?id_token_hint=" + sentJWT + "&post_logout_redirect_uri=https%3A%2F%2Fapp"}}, "Referer",
			[]string{"https://idp.example/logout?id_token_hint=[REDACTED]&post_logout_redirect_uri=https%3A%2F%2Fapp"}},
		{"harmless header untouched", http.Header{"Content-Type": {"application/json"}}, "Content-Type", []string{"application/json"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := RedactHeaders(tt.in)
			got := out[tt.key]
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
			assertNoSentinel(t, strings.Join(got, "\n"))
			// Output must already be a fixed point of the logger's net.
			for _, v := range got {
				if RedactText(v) != v {
					t.Fatalf("RedactText changed header output %q -> %q", v, RedactText(v))
				}
			}
		})
	}
}

func TestRedactHeadersDoesNotMutateInput(t *testing.T) {
	in := http.Header{
		"Authorization": {"Bearer " + sentA},
		"Cookie":        {"a=" + sentB},
		"Location":      {"https://h/p?code=" + sentC},
	}
	snapshot := in.Clone()
	out := RedactHeaders(in)
	if !reflect.DeepEqual(in, snapshot) {
		t.Fatalf("input mutated: %v", in)
	}
	out["Authorization"][0] = "changed"
	if in["Authorization"][0] != "Bearer "+sentA {
		t.Fatal("output shares backing array with input")
	}
	if len(RedactHeaders(nil)) != 0 {
		t.Fatal("nil header should give empty map")
	}
}

func TestRedactURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"query params", "https://idp.example/authorize?client_id=dataharbor&state=" + sentA + "&nonce=" + sentB + "&scope=openid",
			"https://idp.example/authorize?client_id=dataharbor&state=[REDACTED]&nonce=[REDACTED]&scope=openid"},
		{"callback code", "/api/auth/callback?code=" + sentA + "&state=" + sentB,
			"/api/auth/callback?code=[REDACTED]&state=[REDACTED]"},
		{"userinfo", "https://alice:" + sentA + "@xrootd.example:1094/data/file",
			"https://alice:[REDACTED]@xrootd.example:1094/data/file"},
		{"implicit fragment", "https://app.example/cb#access_token=" + sentA + "&token_type=Bearer&expires_in=3600",
			"https://app.example/cb#access_token=[REDACTED]&token_type=[REDACTED]&expires_in=3600"},
		{"encoded key", "https://h.example/p?access%5Ftoken=" + sentA + "&x=1",
			"https://h.example/p?access%5Ftoken=[REDACTED]&x=1"},
		{"nested encoded redirect", "https://idp.example/login?redirect_uri=https%3A%2F%2Fapp%2Fcb%3Fcode%3D" + sentA + "%26state%3D" + sentB + "&x=1",
			"https://idp.example/login?redirect_uri=https%3A%2F%2Fapp%2Fcb%3Fcode%3D[REDACTED]%26state%3D[REDACTED]&x=1"},
		{"jwt in path", "https://h.example/verify/" + sentJWT, "https://h.example/verify/[REDACTED]"},
		{"cli token in query", "https://h.example/api?x=" + sentCLI, "https://h.example/api?x=dh_cli_[REDACTED]"},
		{"unparseable", "http://[::1%zz]/p?token=" + sentA, "http://[::1%zz]/p?token=[REDACTED]"},
		{"no secrets", "https://h.example/a/b?path=%2Fdata&page=2", "https://h.example/a/b?path=%2Fdata&page=2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RedactURL(tt.in)
			if got != tt.want {
				t.Fatalf("\n got: %s\nwant: %s", got, tt.want)
			}
			assertNoSentinel(t, got)
			if RedactText(got) != got {
				t.Fatalf("not a fixed point of RedactText: %s", got)
			}
		})
	}
}

func TestRedactBodyJSON(t *testing.T) {
	tests := []struct {
		name     string
		ct       string
		in       string
		want     string
		survives []string
	}{
		{"nested objects in arrays in objects", "application/json",
			`{"user":"alice","items":[{"name":"a","credentials":{"password":"` + sentA + `","note":"x"}},{"list":[{"access_token":"` + sentB + `"}]}]}`,
			`{"items":[{"credentials":"[REDACTED]","name":"a"},{"list":[{"access_token":"[REDACTED]"}]}],"user":"alice"}`, nil},
		{"numeric code kept, string code redacted", "application/json; charset=utf-8",
			`{"code":200,"data":{"code":"` + sentA + `","state":"` + sentB + `","status":"ok"},"message":"success"}`,
			`{"code":200,"data":{"code":"[REDACTED]","state":"[REDACTED]","status":"ok"},"message":"success"}`, nil},
		{"token response", "application/json",
			`{"access_token":"` + sentJWT + `","token_type":"Bearer","expires_in":3600,"refresh_token":"` + sentA + `","id_token":"` + sentJWT + `","scope":"openid"}`,
			`{"access_token":"[REDACTED]","expires_in":3600,"id_token":"[REDACTED]","refresh_token":"[REDACTED]","scope":"openid","token_type":"[REDACTED]"}`, nil},
		{"url-valued string", "application/json",
			`{"auth_url":"https://idp.example/authorize?client_id=dh&state=` + sentA + `&code_challenge=` + sentB + `"}`,
			`{"auth_url":"https://idp.example/authorize?client_id=dh&state=[REDACTED]&code_challenge=[REDACTED]"}`, nil},
		{"json in string", "application/json",
			`{"body":"{\"password\":\"` + sentA + `\",\"user\":\"bob\"}"}`,
			`{"body":"{\"password\":\"[REDACTED]\",\"user\":\"bob\"}"}`, nil},
		{"numbers preserved, html not escaped", "application/json",
			`{"size":12345678901234567890,"ratio":1.50,"path":"/a<b>&c"}`,
			`{"path":"/a<b>&c","ratio":1.50,"size":12345678901234567890}`, nil},
		{"detected without content type", "",
			`[{"secret":"` + sentA + `"},{"n":1}]`,
			`[{"secret":"[REDACTED]"},{"n":1}]`, nil},
		{"null and bool kept for sensitive keys", "application/json",
			`{"password":null,"has_token":true,"token_count":5}`,
			`{"has_token":true,"password":null,"token_count":"[REDACTED]"}`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RedactBody(tt.ct, []byte(tt.in))
			if got != tt.want {
				t.Fatalf("\n got: %s\nwant: %s", got, tt.want)
			}
			assertNoSentinel(t, got)
			if !json.Valid([]byte(got)) {
				t.Fatalf("output is not valid JSON: %s", got)
			}
			if RedactText(got) != got {
				t.Fatalf("not a fixed point of RedactText:\n%s\n%s", got, RedactText(got))
			}
		})
	}
}

func TestRedactBodyFormAndFailClosed(t *testing.T) {
	tests := []struct {
		name string
		ct   string
		in   []byte
		want string
	}{
		{"form body", "application/x-www-form-urlencoded",
			[]byte("grant_type=authorization_code&code=" + sentA + "&redirect_uri=https%3A%2F%2Fapp%2Fcb&client_secret=" + sentB + "&code_verifier=" + sentC),
			"grant_type=authorization_code&code=[REDACTED]&redirect_uri=https%3A%2F%2Fapp%2Fcb&client_secret=[REDACTED]&code_verifier=[REDACTED]"},
		{"form refresh", "application/x-www-form-urlencoded; charset=UTF-8",
			[]byte("grant_type=refresh_token&refresh_token=" + sentA),
			"grant_type=refresh_token&refresh_token=[REDACTED]"},
		{"html", "text/html; charset=utf-8", []byte("<html><input name=token value=" + sentA + "></html>"), "[" + itoa(len("<html><input name=token value="+sentA+"></html>")) + " bytes of text/html not logged]"},
		{"plain", "text/plain", []byte("hello " + sentA), "[" + itoa(len("hello "+sentA)) + " bytes of text/plain not logged]"},
		{"binary", "application/octet-stream", []byte{0x00, 0xff, 0xfe, 'S', 'e', 'n'}, "[6 bytes of application/octet-stream not logged]"},
		{"invalid json", "application/json", []byte(`{"token":"` + sentA), "[" + itoa(len(`{"token":"`+sentA)) + " bytes of application/json not logged]"},
		{"json trailing data", "application/json", []byte(`{"a":1} ` + sentA), "[" + itoa(len(`{"a":1} `+sentA)) + " bytes of application/json not logged]"},
		{"json scalar", "application/json", []byte(`"` + sentA + `"`), "[" + itoa(len(sentA)+2) + " bytes of application/json not logged]"},
		{"garbage content type", "\x00bad" + sentA, []byte("x" + sentA), "[" + itoa(len(sentA)+1) + " bytes of unknown content type not logged]"},
		{"empty", "application/json", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RedactBody(tt.ct, tt.in)
			if got != tt.want {
				t.Fatalf("\n got: %s\nwant: %s", got, tt.want)
			}
			assertNoSentinel(t, got)
		})
	}
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

func TestRedactText(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		survives []string
	}{
		{"bearer", "upstream rejected Authorization: Bearer " + sentA + " (401)", []string{"Authorization: Bearer [REDACTED]", "(401)"}},
		{"basic lowercase", "using basic " + sentA, []string{"basic [REDACTED]"}},
		{"chained scheme words", "invalid token Bearer " + sentA, []string{"invalid token Bearer [REDACTED]"}},
		{"token scheme", "header Token " + sentA, []string{"Token [REDACTED]"}},
		{"jwt alone", "id token is " + sentJWT + " here", []string{"id token is [REDACTED] here"}},
		{"jwe five segments", "x eyJhbGciOiJSU0EifQ." + sentA + "." + sentB + "." + sentC + ".tagSentinel y", []string{"x [REDACTED] y"}},
		{"cli token", "login with " + sentCLI + " failed", []string{"login with dh_cli_[REDACTED] failed"}},
		{"error string with url", `Get "https://idp.example/logout?id_token_hint=` + sentJWT + `&x=1": dial tcp: timeout`,
			[]string{`Get "https://idp.example/logout?id_token_hint=[REDACTED]&x=1": dial tcp: timeout`}},
		{"query string text", "callback code=" + sentA + "&state=" + sentB + " received", []string{"code=[REDACTED]&state=[REDACTED] received"}},
		{"url encoded nested", "redirect_uri=https%3A%2F%2Fapp%2Fcb%3Fcode%3D" + sentA + "%26state%3D" + sentB,
			[]string{"redirect_uri=https%3A%2F%2Fapp%2Fcb%3Fcode%3D[REDACTED]%26state%3D[REDACTED]"}},
		{"double encoded", "u=https%253A%252F%252Fa%252Fb%253Fcode%253D" + sentA + "%2526x%253D1", []string{"code%253D[REDACTED]%2526x%253D1"}},
		{"quoted kv", `password="` + sentA + ` with space" user=bob`, []string{`password="[REDACTED]" user=bob`}},
		{"json fragment", `body: {"client_secret": "` + sentA + `", "client_id":"dh"}`, []string{`"client_secret": "[REDACTED]"`, `"client_id":"dh"`}},
		{"escaped json fragment", `msg="{\"refresh_token\":\"` + sentA + `\",\"x\":1}"`, []string{`\"refresh_token\":\"[REDACTED]\"`, `\"x\":1`}},
		{"json numeric code kept", `{"code":200,"message":"ok"}`, []string{`{"code":200,"message":"ok"}`}},
		{"json bare password", `{"password":12345` + sentA + `}`, []string{`{"password":[REDACTED]}`}},
		{"go struct dump", "&{AccessToken:" + sentA + " TokenType:Bearer RefreshToken:" + sentB + " Expiry:2026-01-01}",
			[]string{"AccessToken:[REDACTED]", "TokenType:Bearer", "RefreshToken:[REDACTED]", "Expiry:2026-01-01"}},
		{"url.Values dump", "map[code:[" + sentA + "] state:[" + sentB + "] scope:[openid]]", []string{"code:[REDACTED]", "state:[REDACTED]", "scope:[openid]"}},
		{"cookie line", "request Cookie: a=" + sentA + "; b=" + sentB + "\nnext line", []string{"Cookie: [REDACTED]", "\nnext line"}},
		{"userinfo", "dial root://alice:" + sentA + "@xrootd.example:1094//data failed", []string{"root://alice:[REDACTED]@xrootd.example:1094//data failed"}},
		{"prose kept", "token is expired; status code: 500; key not found", []string{"token is expired; status code: 500; key not found"}},
		{"fingerprint kept", "token=" + Fingerprint(sentA) + " refresh", []string{"token=" + Fingerprint(sentA)}},
		{"invalid utf8", "\xff\xfetoken=" + sentA + "\xc3", []string{"token=[REDACTED]"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RedactText(tt.in)
			assertNoSentinel(t, got)
			assertContainsAll(t, got, tt.survives...)
			if again := RedactText(got); again != got {
				t.Fatalf("not idempotent:\n1: %s\n2: %s", got, again)
			}
		})
	}
}

// Exact-output cases for defects found in review: encoded delimiters inside a
// secret, whitespace around "=", username-only URL userinfo and PEM blocks.
func TestRedactTextReviewedDefects(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		// (1) A key matched in literal text ends at a LITERAL delimiter only:
		// an encoded delimiter inside the secret must not truncate redaction.
		{"encoded ampersand inside literal secret", "access_token=abc%2Bdef%26" + sentA, "access_token=[REDACTED]"},
		{"encoded space inside quoted url secret", `Post "https://idp/x?client_secret=p%20` + sentA + `": dial tcp`,
			`Post "https://idp/x?client_secret=[REDACTED]": dial tcp`},
		{"encoded hash and quote inside literal secret", "refresh_token=a%23b%22" + sentA + " next", "refresh_token=[REDACTED] next"},
		{"literal delimiter still ends the value", "token=x%26" + sentA + "&user=bob", "token=[REDACTED]&user=bob"},
		// A key found inside a once-encoded nested URL ends at a once-encoded
		// delimiter, but a twice-encoded one is part of its value.
		{"nested value keeps deeper-encoded delimiter", "redirect_uri=https%3A%2F%2Fa%2Fcb%3Fcode%3Dab%2526" + sentA + "%26x%3D1",
			"redirect_uri=https%3A%2F%2Fa%2Fcb%3Fcode%3D[REDACTED]%26x%3D1"},
		{"nested value ends at literal delimiter", "redirect_uri=https%3A%2F%2Fa%2Fcb%3Fcode%3D" + sentA + "&x=1",
			"redirect_uri=https%3A%2F%2Fa%2Fcb%3Fcode%3D[REDACTED]&x=1"},
		{"double-encoded value keeps triple-encoded delimiter", "u=https%253A%252F%252Fa%253Fcode%253Dab%252526" + sentA + "%2526x%253D1",
			"u=https%253A%252F%252Fa%253Fcode%253D[REDACTED]%2526x%253D1"},

		// (2) Optional horizontal whitespace around "=" for strong keys.
		{"spaces around equals", "password = hunter2", "password = [REDACTED]"},
		{"tabs around equals", "client_secret\t=\t" + sentA + " next", "client_secret\t=\t[REDACTED] next"},
		{"space before equals only", "api_key =" + sentA, "api_key =[REDACTED]"},
		{"space after equals only", "password= " + sentA, "password= [REDACTED]"},
		// "token" and "bearer" are also auth-scheme words: the scheme pass must
		// not take "=" for the credential and leave the value behind.
		{"scheme word as spaced key", "token = " + sentA, "token = [REDACTED]"},
		{"hyphenated scheme-word key", "refresh-token = " + sentA, "refresh-token = [REDACTED]"},
		{"scheme then spaced key", "Bearer password = " + sentA, "Bearer password = [REDACTED]"},
		{"scheme then spaced colon key", "Bearer password : " + sentA, "Bearer password : [REDACTED]"},

		// (3) URL userinfo without a password.
		{"username-only userinfo", "https://ghp_abcdef1234567890@github.com/x", "https://[REDACTED]@github.com/x"},
		{"username-only userinfo in prose", "clone ssh://" + sentA + "@git.example/repo failed", "clone ssh://[REDACTED]@git.example/repo failed"},
		{"username-only userinfo with port", "root://" + sentA + "@xrootd.example:1094//data", "root://[REDACTED]@xrootd.example:1094//data"},
		{"user and password keeps user", "https://alice:" + sentA + "@h.example/p", "https://alice:[REDACTED]@h.example/p"},

		// (4) PEM blocks, whatever key (if any) they are under.
		{"pem under colon key", "privateKey: -----BEGIN PRIVATE KEY-----\nMII" + sentA + "\n-----END PRIVATE KEY-----",
			"privateKey: [REDACTED]"},
		{"pem in prose", "loaded -----BEGIN EC PRIVATE KEY-----\n" + sentA + "\n" + sentB + "\n-----END EC PRIVATE KEY----- ok",
			"loaded [REDACTED] ok"},
		{"pem with escaped newlines under neutral json key", `{"pem":"-----BEGIN RSA PRIVATE KEY-----\nMII` + sentA + `\n-----END RSA PRIVATE KEY-----\n","x":1}`,
			`{"pem":"[REDACTED]\n","x":1}`},
		{"pem under sensitive json key", `{"private_key":"-----BEGIN PRIVATE KEY-----\nMII` + sentA + `\n-----END PRIVATE KEY-----\n"}`,
			`{"private_key":"[REDACTED]"}`},
		{"unterminated pem to end of input", "cfg -----BEGIN OPENSSH PRIVATE KEY-----\nb3Bl" + sentA + "\n" + sentB,
			"cfg [REDACTED]"},
		{"two pem blocks", "a -----BEGIN CERTIFICATE-----\nX" + sentA + "\n-----END CERTIFICATE----- mid -----BEGIN PRIVATE KEY-----\n" + sentB + "\n-----END PRIVATE KEY----- z",
			"a [REDACTED] mid [REDACTED] z"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RedactText(tt.in)
			if got != tt.want {
				t.Fatalf("\n  in: %q\n got: %q\nwant: %q", tt.in, got, tt.want)
			}
			assertNoSentinel(t, got)
			if again := RedactText(got); again != got {
				t.Fatalf("not idempotent:\n1: %q\n2: %q", got, again)
			}
		})
	}
}

// False-positive guards for the review fixes: these inputs must come out
// unchanged.
func TestRedactTextReviewedDefectsNegatives(t *testing.T) {
	for _, in := range []string{
		// (2) Weak keys do not take the spaced form: gRPC errors and status
		// words read "code = Unavailable", "state = running".
		"rpc error: code = Unavailable desc = connection refused",
		"worker state = running",
		// Non-sensitive keys with spaces around "=".
		"status = 500, path = /data, user = bob",
		// The separator-less form is deliberately not handled (prose).
		"PASSWORD hunter2",
		"the password field is required",
		// (3) E-mail addresses and "@" after the authority are not userinfo.
		"mailto:alice@example.com",
		"contact alice@example.com for access",
		"https://h.example/path/user@x",
		"https://h.example/p?q=a@b",
		"https://h.example/p#frag@x",
		"see https://h.example and bob@example.org",
		// (4) A lone marker word without the dashes is not a PEM block.
		"BEGIN PRIVATE KEY was not found",
	} {
		if got := RedactText(in); got != in {
			t.Errorf("changed:\n  in: %q\n got: %q", in, got)
		}
	}
}

func TestRedactError(t *testing.T) {
	if got := RedactError(nil); got != "<nil>" {
		t.Fatalf("nil error: %q", got)
	}
	err := errors.New(`Post "https://idp.example/token": refresh_token=` + sentA + `: 400`)
	got := RedactError(err)
	assertNoSentinel(t, got)
	assertContainsAll(t, got, "https://idp.example/token", "refresh_token=[REDACTED]", "400")
}

func TestRedactFingerprint(t *testing.T) {
	const secretX = "hunterSentinelXyzQwerty" // no 4 consecutive hex chars: cannot collide with hex output
	a1, a2 := Fingerprint(secretX), Fingerprint(secretX)
	if a1 != a2 {
		t.Fatalf("not stable within process: %s vs %s", a1, a2)
	}
	if b := Fingerprint(secretX + "2"); b == a1 {
		t.Fatal("different inputs gave the same fingerprint")
	}
	if !isFingerprint(a1) || !strings.HasPrefix(a1, "fp:") || len(a1) != 15 {
		t.Fatalf("unexpected format %q", a1)
	}
	if sharesSubstring(a1, secretX, 4) {
		t.Fatalf("fingerprint %s shares a substring with the secret", a1)
	}
	if Fingerprint("") != "" {
		t.Fatal("empty input must give empty fingerprint")
	}
	// Hex-looking secrets: the re-keying loop must still avoid substrings.
	for _, s := range []string{"0123456789abcdef", "deadbeefcafebabe0011", "abcd"} {
		if fp := Fingerprint(s); sharesSubstring(fp, s, 4) {
			t.Fatalf("fingerprint %s shares a substring with %s", fp, s)
		}
	}
	// Fingerprints survive the redactors.
	if RedactText("token="+a1) != "token="+a1 || RedactBody("application/json", []byte(`{"token":"`+a1+`"}`)) != `{"token":"`+a1+`"}` {
		t.Fatal("fingerprint was redacted")
	}
}

func TestRedactTextNoPanicAndIdempotentLoop(t *testing.T) {
	pieces := []string{"Bearer ", "token", "=", "%3D", "%26", "&", "\"", `\"`, ":", " ", "eyJ", ".", "dh_cli_", "abc", "[", "]", "{", "}",
		"code", "state", "https://", "user:", "@", "#", "Cookie: ", "\xff", "\n", "%", "%25", "password", sentA,
		" = ", "\t", "%2526", "%20", "-----BEGIN PRIVATE KEY-----", "-----END PRIVATE KEY-----", `\n`, "ssh://"}
	seed := uint32(2166136261)
	next := func() uint32 { seed ^= seed << 13; seed ^= seed >> 17; seed ^= seed << 5; return seed }
	for i := 0; i < 5000; i++ {
		var b strings.Builder
		n := int(next()%12) + 1
		for j := 0; j < n; j++ {
			b.WriteString(pieces[next()%uint32(len(pieces))])
		}
		in := b.String()
		once := RedactText(in)
		if twice := RedactText(once); twice != once {
			t.Fatalf("not idempotent for %q:\n1: %q\n2: %q", in, once, twice)
		}
		_ = RedactURL(in)
		_ = RedactBody("application/json", []byte(in))
		_ = RedactBody("application/x-www-form-urlencoded", []byte(in))
	}
}

func FuzzRedactText(f *testing.F) {
	for _, s := range []string{
		"Bearer abc", "token=abc&state=x", `{"password":"x"}`, `\"token\":\"x\"`, "https://u:p@h/p?code=1#access_token=2",
		sentJWT, sentCLI, "map[code:[x]]", "Cookie: a=b", "%3Fcode%3Dx%26", "\xff\xfe", "token token token",
		"password = x", "token = x", "access_token=a%26b", "%3Fcode%3Da%2526b%26c", "https://user@h/p",
		"k: -----BEGIN PRIVATE KEY-----\nx\n-----END PRIVATE KEY-----", `"-----BEGIN X-----\nabc`,
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		once := RedactText(s)
		if twice := RedactText(once); twice != once {
			t.Fatalf("not idempotent for %q:\n1: %q\n2: %q", s, once, twice)
		}
		_ = RedactURL(s)
		_ = RedactBody("", []byte(s))
	})
}

// --- redacting zap core ----------------------------------------------------

type stringerFunc func() string

func (f stringerFunc) String() string { return f() }

type nilStringer struct{ v string }

func (n *nilStringer) String() string { return n.v }

func TestRedactingCore(t *testing.T) {
	inner, logs := observer.New(zapcore.DebugLevel)
	logger := zap.New(newRedactingCore(inner)).With(zap.String("ctx", "Bearer "+sentC))

	logger.Info("call failed: Authorization: Bearer "+sentA+" jwt="+sentJWT+" cli "+sentCLI,
		zap.String("detail", "refresh_token="+sentB),
		zap.String("token", "opaque"+sentB),
		zap.String("code", "x"+sentB),
		zap.Int("status_code", 401),
		zap.Error(errors.New("dial https://u:"+sentA+"@h/p")),
		zap.Stringer("st", stringerFunc(func() string { return "Bearer " + sentB })),
		zap.Stringer("nilst", (*nilStringer)(nil)),
		zap.ByteString("raw", []byte("password="+sentA)),
		zap.Strings("list", []string{"Bearer " + sentA, "ok"}),
		zap.Any("hdr", map[string][]string{"Authorization": {"Bearer " + sentA}, "Accept": {"*/*"}}),
		zap.String("fp", Fingerprint(sentA)),
	)

	entries := logs.AllUntimed()
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	e := entries[0]
	assertNoSentinel(t, e.Message)
	assertContainsAll(t, e.Message, "call failed", "Bearer [REDACTED]", "dh_cli_[REDACTED]")

	enc := zapcore.NewMapObjectEncoder()
	for _, f := range e.Context {
		f.AddTo(enc)
	}
	dump, err := json.Marshal(enc.Fields)
	if err != nil {
		t.Fatal(err)
	}
	out := string(dump)
	assertNoSentinel(t, out)
	assertContainsAll(t, out, `"ctx":"Bearer [REDACTED]"`, `"detail":"refresh_token=[REDACTED]"`, `"token":"[REDACTED]"`,
		`"code":"[REDACTED]"`, `"status_code":401`, `"error":"dial https://u:[REDACTED]@h/p"`, `"st":"Bearer [REDACTED]"`,
		`"nilst":"\u003cnil\u003e"`, `"raw":"password=[REDACTED]"`, `"list":["Bearer [REDACTED]","ok"]`,
		`"hdr":{"Accept":["*/*"],"Authorization":"[REDACTED]"}`, `"fp":"`+Fingerprint(sentA)+`"`)
}

// Sugared key/value calls (Infow etc.) become structured fields: the download
// log keys must stay readable, while values still go through redaction.
func TestRedactingCoreSugaredKeyValues(t *testing.T) {
	inner, logs := observer.New(zapcore.DebugLevel)
	logger := zap.New(newRedactingCore(inner)).Sugar()

	logger.Infow("File download finished",
		"downloadID", "dl_42",
		"path", "/data/user/file.bin",
		"size", int64(10485760),
		"user", "user_a.b",
		"status", "completed",
		"duration", 3*time.Second,
		"error", errors.New("open failed: Bearer "+sentA))

	entries := logs.AllUntimed()
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Message != "File download finished" {
		t.Fatalf("message = %q", e.Message)
	}
	enc := zapcore.NewMapObjectEncoder()
	for _, f := range e.Context {
		f.AddTo(enc)
	}
	dump, err := json.Marshal(enc.Fields)
	if err != nil {
		t.Fatal(err)
	}
	out := string(dump)
	assertNoSentinel(t, out)
	assertContainsAll(t, out, `"downloadID":"dl_42"`, `"path":"/data/user/file.bin"`, `"size":10485760`,
		`"user":"user_a.b"`, `"status":"completed"`, `"duration":3000000000`, `"error":"open failed: Bearer [REDACTED]"`)
}

func TestRedactingCoreLevelsAndCheck(t *testing.T) {
	debugObs, debugLogs := observer.New(zapcore.DebugLevel)
	infoObs, infoLogs := observer.New(zapcore.InfoLevel)
	core := zapcore.NewTee(newRedactingCore(debugObs), newRedactingCore(infoObs))
	logger := zap.New(core)

	logger.Debug("debug Bearer " + sentA)
	logger.Info("info Bearer " + sentB)

	if debugLogs.Len() != 2 || infoLogs.Len() != 1 {
		t.Fatalf("per-core levels broken: debug=%d info=%d", debugLogs.Len(), infoLogs.Len())
	}
	for _, e := range append(debugLogs.AllUntimed(), infoLogs.AllUntimed()...) {
		assertNoSentinel(t, e.Message)
	}

	rc := newRedactingCore(infoObs)
	if newRedactingCore(rc) != rc {
		t.Fatal("double wrapping")
	}
	if ce := rc.Check(zapcore.Entry{Level: zapcore.DebugLevel}, nil); ce != nil {
		t.Fatal("disabled level must not be added")
	}
	if ce := rc.Check(zapcore.Entry{Level: zapcore.InfoLevel}, nil); ce == nil {
		t.Fatal("enabled level must be added")
	}
	if err := rc.Sync(); err != nil {
		t.Fatal(err)
	}
	if !rc.Enabled(zapcore.InfoLevel) || rc.Enabled(zapcore.DebugLevel) {
		t.Fatal("Enabled not delegated")
	}

	fields := []zapcore.Field{zap.String("detail", "Bearer "+sentA)}
	_ = redactFields(fields)
	if fields[0].String != "Bearer "+sentA {
		t.Fatal("redactFields mutated the caller's slice")
	}
}

func TestRedactingCoreInstalledByInitLogger(t *testing.T) {
	defer DestroyLogger()
	InitLogger()
	if _, ok := Logger.Desugar().Core().(*redactingCore); !ok {
		t.Fatalf("InitLogger: core is %T", Logger.Desugar().Core())
	}
	InitLoggerFromViper()
	if _, ok := Logger.Desugar().Core().(*redactingCore); !ok {
		t.Fatalf("InitLoggerFromViper: core is %T", Logger.Desugar().Core())
	}
	initLoggerWithConfig(nil)
	if _, ok := Logger.Desugar().Core().(*redactingCore); !ok {
		t.Fatalf("initLoggerWithConfig(nil): core is %T", Logger.Desugar().Core())
	}
}

// The logger's net runs RedactText over messages that already went through
// RedactHeaders. The second pass must neither swallow the headers that follow
// a credential header nor let a half-redacted raw header through.
func TestRedactTextOnRenderedHeaderMaps(t *testing.T) {
	const sentinel = "SENTINELhdrMap4Kq8Zt2Wn6"

	h := http.Header{}
	h.Set("Authorization", "Bearer "+sentinel)
	// Cookie names that are themselves sensitive keys ("session", "token").
	h.Add("Cookie", "sid="+sentinel+"; dataharbor-session="+sentinel+"; csrf_token="+sentinel)
	h.Set("User-Agent", "curl/8.0")
	h.Set("X-Marker", "visible-marker")
	rendered := fmt.Sprintf("Request Headers: %v", RedactHeaders(h))
	out := RedactText(rendered)

	if strings.Contains(out, sentinel) {
		t.Fatalf("sentinel survived: %s", out)
	}
	for _, want := range []string{
		"Authorization:[Bearer [REDACTED]]",
		"Cookie:[sid=[REDACTED]; dataharbor-session=[REDACTED]; csrf_token=[REDACTED]]",
		"User-Agent:[curl/8.0]",
		"X-Marker:[visible-marker]",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("want %q in %q", want, out)
		}
	}
	if again := RedactText(out); again != out {
		t.Errorf("not idempotent:\n%s\n%s", out, again)
	}

	// Raw header dumps from a statement that forgot the helper.
	for _, raw := range []string{
		"map[Cookie:[token=x; sid=" + sentinel + "] X-Marker:[visible-marker]]",
		"map[Cookie:[sid=" + sentinel + "] X-Marker:[visible-marker]]",
		"map[Cookie:[a=[REDACTED]; b=" + sentinel + "] X-Marker:[visible-marker]]",
		"map[Cookie:[a=[REDACTED] " + sentinel + "] X-Marker:[visible-marker]]",
		"map[Authorization:[Bearer [REDACTED] " + sentinel + "] X-Marker:[visible-marker]]",
		"map[Authorization:[" + sentinel + "] Cookie:[a=" + sentinel + "; b=" + sentinel + "] X-Marker:[visible-marker]]",
		"map[Set-Cookie:[sid=" + sentinel + "; Path=/; HttpOnly] X-Marker:[visible-marker]]",
		"map[Cookie:[session=[REDACTED]] token=" + sentinel + " X-Marker:[visible-marker]]",
	} {
		out := RedactText(raw)
		if strings.Contains(out, sentinel) {
			t.Errorf("sentinel survived raw dump %q: %s", raw, out)
		}
		if !strings.Contains(out, "X-Marker:[visible-marker]") {
			t.Errorf("header after the credential was swallowed: %q -> %q", raw, out)
		}
	}

	// A secret glued to a placeholder is not recognised as redacted. The rest
	// of the line may be lost with it; the secret must be.
	glued := "map[Cookie:[session=[REDACTED]" + sentinel + "] X-Marker:[visible-marker]]"
	if out := RedactText(glued); strings.Contains(out, sentinel) {
		t.Errorf("sentinel survived %q: %s", glued, out)
	}
}
