// Package apiprobe implements the DataHarbor contract probe: it calls the
// endpoints of a running backend and validates every live response against
// the OpenAPI specification (see internal/apicontract).
//
// Configuration that carries a secret comes from the environment only:
//
//	API_PROBE_BASE_URL  backend base URL, e.g. https://localhost:22000
//	API_PROBE_COOKIE    value of the "dataharbor-session" cookie
//	API_PROBE_INSECURE  "1" to skip TLS verification (self-signed dev certs)
//
// The session cookie is never accepted on the command line, never printed,
// and all output passes through a redactor that masks it defensively.
package apiprobe

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/AnarManafov/dataharbor/app/internal/apicontract"
)

// Exit codes.
const (
	ExitOK       = 0 // every check passed or was skipped
	ExitMismatch = 1 // at least one check failed
	ExitUsage    = 2 // usage or configuration error
)

// Environment variables read by the probe.
const (
	EnvBaseURL  = "API_PROBE_BASE_URL"
	EnvCookie   = "API_PROBE_COOKIE"
	EnvInsecure = "API_PROBE_INSECURE"
)

// SessionCookieName is the backend's session cookie (controller/auth.go).
const SessionCookieName = "dataharbor-session"

// Options configures one probe run.
type Options struct {
	BaseURL    *url.URL
	Cookie     string // session cookie value; never printed
	Insecure   bool
	SpecPath   string
	Upload     bool
	ScratchDir string
	Timeout    time.Duration
}

// Main parses flags and environment, runs the probe and returns the exit
// code. stdout receives the check lines; stderr receives usage errors.
func Main(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	// The cookie is known before anything is printed, so even flag-package
	// diagnostics go through the redactor.
	cookie := normalizeCookie(getenv(EnvCookie))
	red := newRedactor(cookie)
	out := redactingWriter{w: stdout, r: red}
	errOut := redactingWriter{w: stderr, r: red}

	opts, err := parseOptions(args, getenv, cookie, errOut)
	if err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			fmt.Fprintf(errOut, "api-probe: %s\n", oneLine(red.String(err.Error()), 500))
		}
		return ExitUsage
	}
	spec, err := apicontract.Load(opts.SpecPath)
	if err != nil {
		fmt.Fprintf(errOut, "api-probe: %s\n", oneLine(red.String(err.Error()), 500))
		return ExitUsage
	}
	return Run(opts, spec, newHTTPClient(opts), out)
}

func parseOptions(args []string, getenv func(string) string, cookie string, errOut io.Writer) (*Options, error) {
	fs := flag.NewFlagSet("api-probe", flag.ContinueOnError)
	fs.SetOutput(errOut)
	specPath := fs.String("spec", "../api/openapi.yaml", "path to the OpenAPI 3.1 specification")
	upload := fs.Bool("upload", false, "also exercise the upload endpoints (requires -scratch-dir)")
	scratch := fs.String("scratch-dir", "", "absolute remote directory the upload checks may write to")
	timeout := fs.Duration("timeout", 30*time.Second, "timeout per HTTP request")
	fs.Usage = func() {
		fmt.Fprintf(errOut, "usage: api-probe [-spec path] [-timeout d] [-upload -scratch-dir /remote/dir]\n\n"+
			"environment:\n"+
			"  %s   backend base URL, e.g. https://localhost:22000 (required)\n"+
			"  %s     value of the %s cookie (required; never pass it as an argument)\n"+
			"  %s   set to 1 to skip TLS certificate verification (self-signed dev certs)\n\nflags:\n",
			EnvBaseURL, EnvCookie, SessionCookieName, EnvInsecure)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() > 0 {
		// Do not echo the arguments: someone may have pasted the cookie here.
		return nil, fmt.Errorf("unexpected positional arguments (%d); the session cookie is read from %s only", fs.NArg(), EnvCookie)
	}

	rawBase := strings.TrimSpace(getenv(EnvBaseURL))
	if rawBase == "" {
		return nil, fmt.Errorf("%s is not set", EnvBaseURL)
	}
	base, err := url.Parse(rawBase)
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" {
		return nil, fmt.Errorf("%s must be an absolute http(s) URL", EnvBaseURL)
	}
	if base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return nil, fmt.Errorf("%s must not contain credentials, a query or a fragment", EnvBaseURL)
	}
	base.Path = strings.TrimRight(base.Path, "/")

	if cookie == "" {
		return nil, fmt.Errorf("%s is not set (value of the %s cookie)", EnvCookie, SessionCookieName)
	}
	if !validCookieValue(cookie) {
		return nil, fmt.Errorf("%s contains characters that are not allowed in a cookie value", EnvCookie)
	}

	insecure := false
	switch v := strings.TrimSpace(getenv(EnvInsecure)); v {
	case "", "0":
	case "1":
		insecure = true
	default:
		return nil, fmt.Errorf("%s must be 1 or unset", EnvInsecure)
	}

	if *upload != (*scratch != "") {
		return nil, fmt.Errorf("-upload and -scratch-dir must be given together")
	}
	// Normalize before validating, so "/" cannot turn into an empty destDir.
	scratchDir := strings.TrimRight(*scratch, "/")
	if *scratch != "" {
		if !strings.HasPrefix(*scratch, "/") || strings.Contains(scratchDir, "..") || strings.ContainsAny(scratchDir, "\x00\r\n") {
			return nil, fmt.Errorf("-scratch-dir must be an absolute remote path without '..', NUL, CR or LF")
		}
		if scratchDir == "" {
			// The server would accept "/", but the probe leaves a file behind
			// and must not write it into the root of the storage.
			return nil, fmt.Errorf("-scratch-dir must not be the root directory; give a dedicated scratch directory")
		}
	}
	if *timeout <= 0 {
		return nil, fmt.Errorf("-timeout must be positive")
	}
	return &Options{
		BaseURL:    base,
		Cookie:     cookie,
		Insecure:   insecure,
		SpecPath:   *specPath,
		Upload:     *upload,
		ScratchDir: scratchDir,
		Timeout:    *timeout,
	}, nil
}

// normalizeCookie accepts either the bare value or "dataharbor-session=value".
func normalizeCookie(v string) string {
	v = strings.TrimSpace(v)
	return strings.TrimPrefix(v, SessionCookieName+"=")
}

// validCookieValue applies the RFC 6265 cookie-octet rule, which also keeps
// the value from injecting headers.
func validCookieValue(v string) bool {
	for i := 0; i < len(v); i++ {
		c := v[i]
		if c <= 0x20 || c >= 0x7f || c == '"' || c == ',' || c == ';' || c == '\\' {
			return false
		}
	}
	return true
}

func newHTTPClient(opts *Options) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if opts.Insecure {
		// #nosec G402 -- explicit opt-in (API_PROBE_INSECURE=1) for the development
		// stack, which serves self-signed certificates; off by default.
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}
	}
	return &http.Client{
		Transport: transport,
		Timeout:   opts.Timeout,
		// Never follow redirects: the probe validates the response it got, and
		// following one could carry the session to another origin.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		// No cookie jar: a refreshed Set-Cookie from the server is ignored.
	}
}
