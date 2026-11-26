package apiprobe

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/AnarManafov/dataharbor/app/internal/apicontract"
)

// jsonBodyLimit caps how much of a JSON response is read. A var so tests can
// lower it.
var jsonBodyLimit int64 = 16 << 20

// busyRetries and busyDelay bound how often, and how long apart, a download
// check is repeated while the server answers 429 for the per-user download
// limit. Vars so tests can shorten them.
var (
	busyRetries = 5
	busyDelay   = 100 * time.Millisecond
)

const (
	xrdPrefix = "/api/v1/xrd"

	// downloadReadCap caps how much of a binary download is read before the
	// connection is closed.
	downloadReadCap = 4096
	// excerptLen caps body excerpts in FAIL lines.
	excerptLen = 200
	// reasonLen caps the validation message in FAIL lines.
	reasonLen = 600

	bogusUploadID = "api-probe-nonexistent-upload"
)

// skipReasons explains why an operation is never called by default. Keys are
// concrete sample paths, resolved against the spec, so the table does not
// depend on how the spec names path parameters.
var skipReasons = []struct {
	method, sample, reason string
}{
	{"POST", "/api/auth/logout", "never called: it would end the session"},
	{"GET", "/api/auth/callback", "never called: it needs an authorization code from the identity provider"},
}

// uploadOps are the write operations exercised only with -upload.
var uploadOps = []struct{ method, sample string }{
	{"POST", xrdPrefix + "/upload/session"},
	{"PUT", xrdPrefix + "/upload/" + bogusUploadID + "/chunk"},
	{"POST", xrdPrefix + "/upload/" + bogusUploadID + "/complete"},
	{"DELETE", xrdPrefix + "/upload/" + bogusUploadID},
}

type verdict string

const (
	pass verdict = "PASS"
	fail verdict = "FAIL"
	skip verdict = "SKIP"
)

type opKey struct{ method, path string }

type prober struct {
	opts   *Options
	spec   *apicontract.Spec
	client *http.Client
	out    io.Writer
	red    *redactor

	hit                 map[opKey]bool
	nPass, nFail, nSkip int
	authEnabled         bool
	uploadEnabled       bool
	uploadLimitsKnown   bool
	initialDir, file    string
	fileSize            int64
	uploadChunkSize     int
}

// call describes one HTTP check.
type call struct {
	method  string
	path    string     // concrete path, without the base URL path prefix
	query   url.Values // optional
	body    []byte
	ctype   string // request Content-Type
	cookie  bool   // send the session cookie
	expect  []int  // acceptable statuses; PASS requires one of them
	readCap int64  // cap for non-JSON bodies; 0 means read fully (up to jsonBodyLimit)
	label   string // printed after the status
	// after inspects a contract-valid response with an expected status; a
	// non-nil error turns the check into a FAIL. It may also record state.
	after func(*result) error
	// skipIf runs before validation; a non-empty reason turns the check into
	// a SKIP. It is for responses that did not come from the backend at all.
	skipIf func(*result) string
	// retryBusy repeats the request with a short backoff while the server
	// answers 429 (per-user download slot still held, e.g. by the previous
	// download whose connection the probe closed early). A 429 that persists
	// and matches the spec is a SKIP, not a FAIL.
	retryBusy bool
}

type result struct {
	status int
	ctype  string
	body   []byte
}

// Run executes every check against the backend and writes one line per
// check plus a summary to out. It returns ExitOK or ExitMismatch.
func Run(opts *Options, spec *apicontract.Spec, client *http.Client, out io.Writer) int {
	p := &prober{
		opts:        opts,
		spec:        spec,
		client:      client,
		red:         newRedactor(opts.Cookie),
		hit:         map[opKey]bool{},
		authEnabled: true,
	}
	p.out = redactingWriter{w: out, r: p.red}

	p.readOnlyChecks()
	if opts.Upload {
		p.uploadChecks()
	}
	p.skipUncovered()

	p.printf("SUMMARY  pass=%d fail=%d skip=%d\n", p.nPass, p.nFail, p.nSkip)
	if p.nFail > 0 {
		return ExitMismatch
	}
	return ExitOK
}

func (p *prober) readOnlyChecks() {
	// In the docker deployments the nginx gateway answers /health itself with
	// a plain-text body and never forwards it. That response says nothing
	// about the backend; /api/health below reaches the same handler.
	p.do(call{method: "GET", path: "/health", expect: []int{200}, skipIf: gatewayHealth})
	p.do(call{method: "GET", path: "/api/health", expect: []int{200}})

	// LoginInit answers 200 JSON (no redirect). Called without the cookie, it
	// only mints a fresh, discarded pre-login session, so the operator's
	// session is untouched. Its body also tells whether auth is enabled.
	p.do(call{method: "GET", path: "/api/auth/login", expect: []int{200}, label: "without session cookie",
		after: func(r *result) error {
			var b struct {
				AuthURL string `json:"auth_url"`
			}
			if json.Unmarshal(r.body, &b) == nil && b.AuthURL == "/auth/disabled" {
				p.authEnabled = false
			}
			return nil
		}})

	if p.authEnabled {
		p.do(call{method: "GET", path: "/api/auth/user", cookie: true, expect: []int{200}})
	} else {
		p.do(call{method: "GET", path: "/api/auth/user", cookie: true, expect: []int{401},
			label: "auth disabled on server: no user session expected"})
	}
	p.do(call{method: "GET", path: "/api/auth/user", expect: []int{401}, label: "without session cookie"})

	if p.authEnabled {
		p.do(call{method: "GET", path: xrdPrefix + "/initialDir", expect: []int{401}, label: "without session cookie"})
	} else {
		p.skipLine("GET", xrdPrefix+"/initialDir", "unauthenticated check: auth is disabled on the server")
	}

	p.do(call{method: "GET", path: xrdPrefix + "/initialDir", cookie: true, expect: []int{200},
		after: func(r *result) error {
			var b struct {
				Data string `json:"data"`
			}
			if err := json.Unmarshal(r.body, &b); err != nil || !strings.HasPrefix(b.Data, "/") {
				return fmt.Errorf("data is not an absolute directory path")
			}
			p.initialDir = b.Data
			return nil
		}})
	p.do(call{method: "GET", path: xrdPrefix + "/hostname", cookie: true, expect: []int{200}})
	p.do(call{method: "GET", path: xrdPrefix + "/ping", cookie: true, expect: []int{200}})

	if p.initialDir == "" {
		const why = "initial directory unknown (initialDir check failed)"
		p.skipLine("GET", xrdPrefix+"/vstat", why)
		p.skipLine("GET", xrdPrefix+"/ls", why)
		p.skipLine("POST", xrdPrefix+"/ls/paged", why)
	} else {
		p.do(call{method: "GET", path: xrdPrefix + "/vstat", query: url.Values{"path": {p.initialDir}}, cookie: true, expect: []int{200}})
		p.do(call{method: "GET", path: xrdPrefix + "/ls", query: url.Values{"dir": {p.initialDir}}, cookie: true, expect: []int{200},
			after: func(r *result) error { p.pickFileFromLs(r.body); return nil }})
		body, _ := json.Marshal(map[string]any{"path": p.initialDir, "page": 1, "pageSize": 100})
		p.do(call{method: "POST", path: xrdPrefix + "/ls/paged", body: body, ctype: "application/json", cookie: true, expect: []int{200},
			after: func(r *result) error {
				if p.file == "" {
					p.pickFileFromPaged(r.body)
				}
				return nil
			}})
	}

	p.do(call{method: "GET", path: xrdPrefix + "/upload/limits", cookie: true, expect: []int{200},
		after: func(r *result) error {
			var b struct {
				Data struct {
					Upload struct {
						Enabled   bool `json:"enabled"`
						ChunkSize int  `json:"chunkSize"`
					} `json:"upload"`
				} `json:"data"`
			}
			if json.Unmarshal(r.body, &b) == nil {
				p.uploadLimitsKnown = true
				p.uploadEnabled = b.Data.Upload.Enabled
				p.uploadChunkSize = b.Data.Upload.ChunkSize
			}
			return nil
		}})
	p.do(call{method: "GET", path: xrdPrefix + "/upload/" + bogusUploadID + "/status", cookie: true, expect: []int{404},
		label: "unknown uploadId"})

	if p.file == "" {
		why := "no regular file found in the initial directory listing"
		p.skipLine("GET", xrdPrefix+"/download", why)
		p.skipLine("POST", xrdPrefix+"/download/batch", why)
		return
	}
	full := path.Join(p.initialDir, p.file)
	label := fmt.Sprintf("first %d bytes of the smallest file (%d bytes)", downloadReadCap, p.fileSize)
	p.do(call{method: "GET", path: xrdPrefix + "/download", query: url.Values{"path": {full}}, cookie: true,
		expect: []int{200}, readCap: downloadReadCap, label: label, retryBusy: true})
	// The server frees the download slot of the previous check only once its
	// copy notices the closed connection, so with max_concurrent_per_user: 1
	// this may briefly answer 429; retryBusy covers that.
	body, _ := json.Marshal(map[string]any{"basePath": p.initialDir, "files": []string{p.file}})
	p.do(call{method: "POST", path: xrdPrefix + "/download/batch", body: body, ctype: "application/json", cookie: true,
		expect: []int{200}, readCap: downloadReadCap, label: "archive of the same file, first bytes only", retryBusy: true})
}

// gatewayHealth recognises a /health answer produced by a reverse proxy in
// front of the backend: a 200 that is not JSON. Anything else (a JSON body of
// the wrong shape, another status) is still validated and can fail.
func gatewayHealth(r *result) string {
	if r.status != http.StatusOK {
		return ""
	}
	mt, _, _ := mime.ParseMediaType(r.ctype)
	if mt == "application/json" || strings.HasSuffix(mt, "+json") {
		return ""
	}
	return "answered by a gateway in front of the backend (non-JSON 200), not by the backend; see GET /api/health"
}

// pickFileFromLs remembers the smallest regular file of a GET /xrd/ls body.
func (p *prober) pickFileFromLs(body []byte) {
	var b struct {
		Entries []struct {
			Name  string `json:"name"`
			IsDir bool   `json:"isDir"`
			Size  int64  `json:"size"`
		} `json:"entries"`
	}
	if json.Unmarshal(body, &b) != nil {
		return
	}
	for _, e := range b.Entries {
		if !e.IsDir && usableName(e.Name) && (p.file == "" || e.Size < p.fileSize) {
			p.file, p.fileSize = e.Name, e.Size
		}
	}
}

// pickFileFromPaged is the fallback for a POST /xrd/ls/paged body.
func (p *prober) pickFileFromPaged(body []byte) {
	var b struct {
		Items []struct {
			Name string `json:"name"`
			Type string `json:"type"`
			Size int64  `json:"size"`
		} `json:"items"`
	}
	if json.Unmarshal(body, &b) != nil {
		return
	}
	for _, it := range b.Items {
		if it.Type == "file" && usableName(it.Name) && (p.file == "" || it.Size < p.fileSize) {
			p.file, p.fileSize = it.Name, it.Size
		}
	}
}

// usableName reports whether the server would accept n as a file name for
// both GET /xrd/download (validateFilePath on the joined path) and POST
// /xrd/download/batch (validateBatchFileName in controller/xrd.go): no path
// separator, no "..", no NUL/CR/LF, valid UTF-8. "." is not a file name.
func usableName(n string) bool {
	return n != "" && n != "." &&
		!strings.ContainsAny(n, "/\\\x00\r\n") &&
		!strings.Contains(n, "..") &&
		utf8.ValidString(n)
}

// uploadChecks runs the write flow in the scratch directory: one complete
// upload (the file stays, there is no delete endpoint) and one aborted one.
func (p *prober) uploadChecks() {
	if p.uploadLimitsKnown && !p.uploadEnabled {
		for _, op := range uploadOps {
			p.skipLine(op.method, op.sample, "upload is disabled on the server (limits: upload.enabled=false)")
		}
		return
	}

	suffix := randomSuffix()
	name := fmt.Sprintf("api-probe-%s-%s.txt", time.Now().UTC().Format("20060102T150405Z"), suffix)
	content := []byte("DataHarbor api-probe contract check " + suffix + "\n")
	sum := sha256.Sum256(content)

	id, chunk := p.createSession(name, int64(len(content)), "")
	if id == "" {
		for _, op := range uploadOps[1:] {
			p.skipLine(op.method, op.sample, "no upload session (session check failed)")
		}
		return
	}
	if chunk <= 0 || chunk > len(content) {
		chunk = len(content)
	}
	ok := true
	for off := 0; off < len(content) && ok; off += chunk {
		end := min(off+chunk, len(content))
		ok = p.do(call{method: "PUT", path: xrdPrefix + "/upload/" + id + "/chunk",
			query: url.Values{"offset": {strconv.Itoa(off)}}, body: content[off:end], ctype: "application/octet-stream",
			cookie: true, expect: []int{200}, label: fmt.Sprintf("offset %d, %d bytes", off, end-off)})
	}
	p.do(call{method: "GET", path: xrdPrefix + "/upload/" + id + "/status", cookie: true, expect: []int{200}, label: "live session"})
	completed := false
	if ok {
		body, _ := json.Marshal(map[string]string{"sha256": hex.EncodeToString(sum[:])})
		dest := path.Join(p.opts.ScratchDir, name)
		completed = p.do(call{method: "POST", path: xrdPrefix + "/upload/" + id + "/complete", body: body, ctype: "application/json",
			cookie: true, expect: []int{200}, label: "file remains in the scratch directory (no delete endpoint): " + dest})
	} else {
		p.skipLine("POST", xrdPrefix+"/upload/"+id+"/complete", "chunk upload failed")
	}
	if !completed {
		// Release the server-side slot and temp file of the failed session.
		p.do(call{method: "DELETE", path: xrdPrefix + "/upload/" + id, cookie: true, expect: []int{200}, label: "cleanup of the failed session"})
	}

	// Second session, aborted before any data is sent.
	id2, _ := p.createSession(fmt.Sprintf("api-probe-abort-%s.txt", suffix), 1, "second session, to be aborted")
	if id2 == "" {
		p.skipLine("DELETE", xrdPrefix+"/upload/"+bogusUploadID, "no second upload session")
		return
	}
	p.do(call{method: "DELETE", path: xrdPrefix + "/upload/" + id2, cookie: true, expect: []int{200}, label: "abort"})
}

// createSession returns the uploadId and chunk size of a one-file session,
// or "" when the check failed.
func (p *prober) createSession(name string, size int64, label string) (string, int) {
	body, _ := json.Marshal(map[string]any{
		"destDir": p.opts.ScratchDir,
		"files":   []map[string]any{{"relPath": name, "size": size, "onConflict": "fail"}},
	})
	var id string
	var chunk int
	p.do(call{method: "POST", path: xrdPrefix + "/upload/session", body: body, ctype: "application/json",
		cookie: true, expect: []int{200}, label: label,
		after: func(r *result) error {
			var b struct {
				Data struct {
					ChunkSize int `json:"chunkSize"`
					Files     []struct {
						UploadID string `json:"uploadId"`
						Status   string `json:"status"`
						Reason   string `json:"reason"`
					} `json:"files"`
				} `json:"data"`
			}
			if err := json.Unmarshal(r.body, &b); err != nil || len(b.Data.Files) != 1 {
				return fmt.Errorf("expected exactly one file in the session response")
			}
			f := b.Data.Files[0]
			if f.Status != "accepted" || f.UploadID == "" {
				return fmt.Errorf("file not accepted: status=%q reason=%q", f.Status, f.Reason)
			}
			id, chunk = f.UploadID, b.Data.ChunkSize
			return nil
		}})
	if chunk <= 0 {
		chunk = p.uploadChunkSize
	}
	return id, chunk
}

// skipUncovered prints a SKIP line for every documented operation the run
// did not call, so the output accounts for the whole specification.
func (p *prober) skipUncovered() {
	reasons := map[opKey]string{}
	for _, s := range skipReasons {
		if op, ok := p.spec.FindOperation(s.method, s.sample); ok {
			reasons[opKey{op.Method, op.Path}] = s.reason
		}
	}
	for _, u := range uploadOps {
		if op, ok := p.spec.FindOperation(u.method, u.sample); ok {
			reasons[opKey{op.Method, op.Path}] = "upload checks run only with -upload and -scratch-dir"
		}
	}
	for _, op := range p.spec.Operations() {
		k := opKey{op.Method, op.Path}
		if p.hit[k] {
			continue
		}
		reason, ok := reasons[k]
		if !ok {
			reason = "not exercised by the probe"
		}
		p.line(skip, op.Method, op.Path, "-", reason)
	}
}

// skipLine reports a check that was not run. The operation counts as
// accounted for, so it is not listed again by skipUncovered.
func (p *prober) skipLine(method, concrete, reason string) {
	shown := concrete
	if op, ok := p.spec.FindOperation(method, concrete); ok {
		shown = op.Path
		p.hit[opKey{op.Method, op.Path}] = true
	}
	p.line(skip, method, shown, "-", reason)
}

// do performs one check and prints its line. It reports whether it passed.
func (p *prober) do(c call) bool {
	shown := c.path
	op, documented := p.spec.FindOperation(c.method, c.path)
	if documented {
		shown = op.Path
		p.hit[opKey{op.Method, op.Path}] = true
	}

	res, err := p.send(c)
	for attempt, wait := 0, time.Duration(0); err == nil && c.retryBusy && res.status == http.StatusTooManyRequests && attempt < busyRetries; attempt++ {
		d := busyDelay << attempt
		if wait+d > p.opts.Timeout {
			break
		}
		wait += d
		time.Sleep(d)
		res, err = p.send(c)
	}
	if err != nil {
		p.line(fail, c.method, shown, "-", joinReason(c.label, "request failed: "+p.clip(err.Error())))
		return false
	}
	status := strconv.Itoa(res.status)
	if !documented {
		p.line(fail, c.method, shown, status, joinReason(c.label, "operation is not documented in the specification"))
		return false
	}
	if c.skipIf != nil {
		if why := c.skipIf(res); why != "" {
			p.line(skip, c.method, shown, "-", joinReason(c.label, why))
			return false
		}
	}
	if err := p.spec.Validate(c.method, c.path, apicontract.Response{Status: res.status, ContentType: res.ctype, Body: res.body}); err != nil {
		p.line(fail, c.method, shown, status, joinReason(c.label, p.clip(err.Error())+p.excerpt(res)))
		return false
	}
	if c.retryBusy && res.status == http.StatusTooManyRequests && !slices.Contains(c.expect, res.status) {
		p.line(skip, c.method, shown, "-", joinReason(c.label,
			"server still answers 429 (per-user download concurrency limit) after retries; the response matches the spec, "+
				"but the intended response could not be checked. Another download of this user may be running, "+
				"or the slot of the previous check was not released yet"))
		return false
	}
	if !slices.Contains(c.expect, res.status) {
		p.line(fail, c.method, shown, status, joinReason(c.label,
			fmt.Sprintf("expected status %s (response matches the spec, but the probe could not check the intended response)", joinInts(c.expect))+p.excerpt(res)))
		return false
	}
	if c.after != nil {
		if err := c.after(res); err != nil {
			p.line(fail, c.method, shown, status, joinReason(c.label, p.clip(err.Error())+p.excerpt(res)))
			return false
		}
	}
	p.line(pass, c.method, shown, status, c.label)
	return true
}

func (p *prober) send(c call) (*result, error) {
	u := *p.opts.BaseURL
	u.Path = p.opts.BaseURL.Path + c.path
	u.RawPath = ""
	if c.query != nil {
		u.RawQuery = c.query.Encode()
	}
	var body io.Reader
	if c.body != nil {
		body = bytes.NewReader(c.body)
	}
	req, err := http.NewRequest(c.method, u.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, */*")
	req.Header.Set("User-Agent", "dataharbor-api-probe")
	if c.ctype != "" {
		req.Header.Set("Content-Type", c.ctype)
	}
	if c.cookie {
		// A request cookie carries no attributes; the value was checked against
		// the RFC 6265 cookie-octet set when the options were parsed.
		req.Header.Set("Cookie", SessionCookieName+"="+p.opts.Cookie)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	ct := resp.Header.Get("Content-Type")
	limit := jsonBodyLimit
	mt, _, _ := mime.ParseMediaType(ct)
	jsonish := mt == "application/json" || strings.HasSuffix(mt, "+json")
	if c.readCap > 0 && !jsonish {
		// Binary download: read a little, then close without draining.
		limit = c.readCap
	}
	readMax := limit
	if jsonish {
		// One extra byte tells a body of exactly limit bytes from a larger one.
		readMax = limit + 1
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, readMax))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if jsonish && int64(len(data)) > limit {
		return nil, fmt.Errorf("JSON body exceeds %d bytes", limit)
	}
	return &result{status: resp.StatusCode, ctype: ct, body: data}, nil
}

// excerpt returns a capped, redacted body excerpt for FAIL lines. Redaction
// happens before the cap so a cut can never expose part of the cookie.
func (p *prober) excerpt(r *result) string {
	if len(r.body) == 0 {
		return ""
	}
	mt, _, _ := mime.ParseMediaType(r.ctype)
	if !strings.HasPrefix(mt, "text/") && mt != "application/json" && !strings.HasSuffix(mt, "+json") {
		return fmt.Sprintf(" [body: %d bytes of %s]", len(r.body), mt)
	}
	return " [body: " + oneLine(p.red.String(string(r.body)), excerptLen) + "]"
}

// line prints one check line: VERDICT  METHOD path  status  [reason].
func (p *prober) line(v verdict, method, pathShown, status, reason string) {
	switch v {
	case pass:
		p.nPass++
	case fail:
		p.nFail++
	case skip:
		p.nSkip++
	}
	s := fmt.Sprintf("%s  %s %s  %s", v, method, pathShown, status)
	if reason != "" {
		s += "  " + oneLine(p.red.String(reason), 0)
	}
	p.printf("%s\n", s)
}

func (p *prober) printf(format string, args ...any) {
	_, _ = fmt.Fprint(p.out, p.red.String(fmt.Sprintf(format, args...)))
}

// clip redacts, flattens and caps a message (redaction first, see excerpt).
func (p *prober) clip(msg string) string {
	return oneLine(p.red.String(msg), reasonLen)
}

func joinReason(label, msg string) string {
	if label == "" {
		return msg
	}
	return label + ": " + msg
}

func joinInts(v []int) string {
	s := make([]string, len(v))
	for i, n := range v {
		s[i] = strconv.Itoa(n)
	}
	return strings.Join(s, " or ")
}

func randomSuffix() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
