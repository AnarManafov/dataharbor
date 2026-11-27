package main

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/AnarManafov/dataharbor/app/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// blockingHandler holds every request to /slow until release is closed and
// signals entered once per request that has reached it.
type blockingHandler struct {
	entered chan struct{}
	release chan struct{}
	served  atomic.Int64
}

func newBlockingHandler() *blockingHandler {
	return &blockingHandler{entered: make(chan struct{}, 16), release: make(chan struct{})}
}

func (h *blockingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/slow" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	h.entered <- struct{}{}
	<-h.release
	h.served.Add(1)
	_, _ = io.WriteString(w, "done")
}

// startTestServer runs serve on a loopback port and returns the address and
// the channel serve's result arrives on.
func startTestServer(t *testing.T, ctx context.Context, h http.Handler, timeout time.Duration) (string, <-chan error) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	result := make(chan error, 1)
	go func() {
		result <- serve(ctx, ln, h, serverOptions{shutdownTimeout: timeout})
	}()
	return ln.Addr().String(), result
}

type slowResult struct {
	body string
	err  error
}

// getSlow requests /slow on its own connection and delivers the outcome.
func getSlow(addr string) <-chan slowResult {
	out := make(chan slowResult, 1)
	go func() {
		client := &http.Client{Transport: &http.Transport{DisableKeepAlives: true}}
		resp, err := client.Get("http://" + addr + "/slow")
		if err != nil {
			out <- slowResult{err: err}
			return
		}
		defer func() { _ = resp.Body.Close() }()
		body, err := io.ReadAll(resp.Body)
		out <- slowResult{body: string(body), err: err}
	}()
	return out
}

func waitFor(t *testing.T, what string, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

// requireRefused polls until a new connection to addr is refused. Shutdown
// closes the listener first, but asynchronously to the caller that cancelled.
func requireRefused(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err != nil {
			return
		}
		_ = conn.Close()
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("new connections to %s are still accepted after shutdown started", addr)
}

func TestServe_DrainsInFlightRequest(t *testing.T) {
	h := newBlockingHandler()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	addr, result := startTestServer(t, ctx, h, 10*time.Second)

	slow := getSlow(addr)
	waitFor(t, "the request to reach the handler", h.entered)

	cancel()
	requireRefused(t, addr)

	// The shutdown must still be waiting for the request it has in flight.
	select {
	case err := <-result:
		t.Fatalf("serve returned (%v) while a request was still in flight", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(h.release)
	select {
	case res := <-slow:
		require.NoError(t, res.err)
		assert.Equal(t, "done", res.body, "the in-flight request must complete with its full response")
	case <-time.After(5 * time.Second):
		t.Fatal("the in-flight request never completed")
	}

	select {
	case err := <-result:
		assert.NoError(t, err, "a clean drain must return nil so the process exits 0")
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not return after the drain")
	}
}

func TestServe_ShutdownTimeoutCutsRequest(t *testing.T) {
	h := newBlockingHandler()
	// Release the handler goroutine at the end so it does not outlive the test.
	defer close(h.release)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const timeout = 200 * time.Millisecond
	addr, result := startTestServer(t, ctx, h, timeout)

	slow := getSlow(addr)
	waitFor(t, "the request to reach the handler", h.entered)

	started := time.Now()
	cancel()

	select {
	case err := <-result:
		elapsed := time.Since(started)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "1 request(s) still in flight")
		assert.GreaterOrEqual(t, elapsed, timeout, "the timeout must be waited out in full")
		assert.Less(t, elapsed, 5*time.Second, "the timeout must bound the shutdown")
	case <-time.After(10 * time.Second):
		t.Fatal("serve did not return after the shutdown timeout")
	}

	select {
	case res := <-slow:
		assert.Error(t, res.err, "a request still running at the deadline is cut")
	case <-time.After(5 * time.Second):
		t.Fatal("the cut request never returned to the client")
	}
	requireRefused(t, addr)
}

func TestServe_IdleShutdownIsClean(t *testing.T) {
	h := newBlockingHandler()
	ctx, cancel := context.WithCancel(context.Background())
	addr, result := startTestServer(t, ctx, h, 5*time.Second)

	resp, err := http.Get("http://" + addr + "/")
	require.NoError(t, err)
	_ = resp.Body.Close()

	cancel()
	select {
	case err := <-result:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not return with nothing in flight")
	}
	requireRefused(t, addr)
}

// TestServe_SIGTERMDrainsAndReturnsNil sends a real SIGTERM to the test process.
// The signal context is installed first, so the signal is handled rather than
// fatal.
func TestServe_SIGTERMDrainsAndReturnsNil(t *testing.T) {
	ctx, stop := shutdownSignalContext()
	defer stop()

	h := newBlockingHandler()
	addr, result := startTestServer(t, ctx, h, 10*time.Second)

	slow := getSlow(addr)
	waitFor(t, "the request to reach the handler", h.entered)

	require.NoError(t, syscall.Kill(os.Getpid(), syscall.SIGTERM))
	waitFor(t, "SIGTERM to cancel the context", ctx.Done())
	requireRefused(t, addr)

	close(h.release)
	select {
	case res := <-slow:
		require.NoError(t, res.err)
		assert.Equal(t, "done", res.body)
	case <-time.After(5 * time.Second):
		t.Fatal("the in-flight request never completed")
	}
	select {
	case err := <-result:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not return after SIGTERM")
	}
}

func TestServe_ListenerFailureIsReturned(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	require.NoError(t, ln.Close()) // Serve fails at once on a closed listener

	done := make(chan error, 1)
	go func() {
		done <- serve(context.Background(), ln, http.NotFoundHandler(), serverOptions{shutdownTimeout: time.Second})
	}()
	select {
	case err := <-done:
		require.Error(t, err)
		assert.Contains(t, err.Error(), "server failed")
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not report the listener failure")
	}
}

func TestParseShutdownTimeout(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
		ok   bool
	}{
		{"45s", 45 * time.Second, true},
		{"2m", 2 * time.Minute, true},
		{"250ms", 250 * time.Millisecond, true},
		{"", defaultShutdownTimeout, false},
		{"30", defaultShutdownTimeout, false},
		{"soon", defaultShutdownTimeout, false},
		{"0s", defaultShutdownTimeout, false},
		{"-5s", defaultShutdownTimeout, false},
	}
	for _, tc := range cases {
		got, ok := parseShutdownTimeout(tc.in)
		assert.Equal(t, tc.want, got, "value %q", tc.in)
		assert.Equal(t, tc.ok, ok, "value %q", tc.in)
	}
}

func TestStartServer_ListenErrorIsReturned(t *testing.T) {
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = busy.Close() }()

	config.SetConfig(&config.Config{Server: config.ServerConfig{Address: busy.Addr().String(), Debug: true}})
	gin.SetMode(gin.TestMode)

	err = startServer(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to listen")
}

// TestStartServer_ShutsDownCleanly covers the real router: startServer returns
// nil once its context is cancelled, having applied server.shutdown_timeout.
func TestStartServer_ShutsDownCleanly(t *testing.T) {
	config.SetConfig(&config.Config{Server: config.ServerConfig{
		Address:         "127.0.0.1:0",
		Debug:           true,
		ShutdownTimeout: "5s",
	}})
	gin.SetMode(gin.TestMode)

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- startServer(ctx) }()

	time.Sleep(100 * time.Millisecond)
	cancel()
	select {
	case err := <-result:
		assert.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("startServer did not return after cancellation")
	}
}

// bulkHandler writes size bytes and returns at once: the response then sits in
// the socket's send buffer until the client reads it.
func bulkHandler(size int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(size))
		_, _ = w.Write(make([]byte, size))
	})
}

// openUnreadResponse sends a request on a raw connection and does not read the
// response, so that it stays queued on the server side.
func openUnreadResponse(t *testing.T, addr string) net.Conn {
	t.Helper()
	conn := dialSmallWindow(t, addr)
	_, err := io.WriteString(conn, "GET / HTTP/1.1\r\nHost: test\r\nConnection: close\r\n\r\n")
	require.NoError(t, err)
	return conn
}

// dialSmallWindow connects with a receive buffer too small to absorb a response.
func dialSmallWindow(t *testing.T, addr string) net.Conn {
	t.Helper()
	// A small receive buffer, set before the handshake so that it bounds the
	// advertised window: the response must not fit into the client's kernel.
	dialer := net.Dialer{Control: func(_, _ string, c syscall.RawConn) error {
		var sockErr error
		if err := c.Control(func(fd uintptr) {
			sockErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, 16<<10)
		}); err != nil {
			return err
		}
		return sockErr
	}}
	conn, err := dialer.Dial("tcp", addr)
	require.NoError(t, err)
	return conn
}

// A response is not delivered when the handler has written it: a slow client
// leaves it queued, in the handler's Write or in the socket's send buffer.
// Either way the shutdown must not finish before the client has all of it,
// because a container's network is torn down the moment the process exits.
func TestServe_WaitsForResponseToReachSlowClient(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the send queue can only be inspected on Linux")
	}
	const size = 512 << 10
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	addr, result := startTestServer(t, ctx, bulkHandler(size), 10*time.Second)

	conn := openUnreadResponse(t, addr)
	defer func() { _ = conn.Close() }()
	time.Sleep(100 * time.Millisecond) // let the server write what it can

	cancel()
	select {
	case err := <-result:
		t.Fatalf("serve returned (%v) while the response was still undelivered", err)
	case <-time.After(300 * time.Millisecond):
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Len(t, body, size, "the client must receive the whole response")

	select {
	case err := <-result:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not return once the response was delivered")
	}
}

func TestServe_UndeliveredResponseAtDeadlineIsAnError(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the send queue can only be inspected on Linux")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const timeout = 300 * time.Millisecond
	addr, result := startTestServer(t, ctx, bulkHandler(512<<10), timeout)

	conn := openUnreadResponse(t, addr) // never read
	defer func() { _ = conn.Close() }()
	time.Sleep(100 * time.Millisecond)

	started := time.Now()
	cancel()
	select {
	case err := <-result:
		require.Error(t, err)
		assert.Contains(t, err.Error(), "shutdown timed out")
		assert.GreaterOrEqual(t, time.Since(started), timeout)
		assert.Less(t, time.Since(started), 5*time.Second)
	case <-time.After(10 * time.Second):
		t.Fatal("serve did not return after the shutdown timeout")
	}
}

// queuedPair returns a server-side drainConn with `size` bytes written to it
// that the client has not read, and the client connection. The server's send
// buffer is made large enough to take the write and the client's receive
// buffer too small to absorb it, so the bytes are in the send queue whatever
// the kernel's defaults are.
func queuedPair(t *testing.T, size int) (*drainListener, *drainConn, net.Conn) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	dl := &drainListener{Listener: ln}
	t.Cleanup(func() { _ = ln.Close() })

	client := dialSmallWindow(t, ln.Addr().String())
	t.Cleanup(func() { _ = client.Close() })
	accepted, err := dl.Accept()
	require.NoError(t, err)
	server, ok := accepted.(*drainConn)
	require.True(t, ok, "TCP connections must be wrapped")
	require.NoError(t, server.SetWriteBuffer(4*size))

	require.NoError(t, server.SetWriteDeadline(time.Now().Add(5*time.Second)))
	_, err = server.Write(make([]byte, size))
	require.NoError(t, err, "the write must fit the send buffer")
	queued, ok := sendQueueBytes(server.TCPConn)
	require.True(t, ok)
	require.Positive(t, queued, "the client must not have taken everything")
	return dl, server, client
}

func TestDrainConn_CloseWaitsForSendQueue(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the send queue can only be inspected on Linux")
	}
	const size = 256 << 10
	dl, server, client := queuedPair(t, size)
	dl.startDrain(time.Now().Add(10 * time.Second))

	closed := make(chan struct{})
	go func() {
		_ = server.Close()
		close(closed)
	}()
	select {
	case <-closed:
		t.Fatal("Close returned with data still queued")
	case <-time.After(200 * time.Millisecond):
	}
	assert.EqualValues(t, 1, dl.closing.Load())

	got, err := io.Copy(io.Discard, client)
	require.NoError(t, err)
	assert.EqualValues(t, size, got, "everything written must reach the client")
	waitFor(t, "Close to return once the queue is empty", closed)
	assert.Zero(t, dl.unsent.Load())
	dl.waitClosed()
}

func TestDrainConn_CloseGivesUpAtDeadlineAndCountsOnce(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the send queue can only be inspected on Linux")
	}
	dl, server, client := queuedPair(t, 256<<10)
	const timeout = 200 * time.Millisecond
	started := time.Now()
	dl.startDrain(started.Add(timeout))

	closed := make(chan struct{})
	go func() {
		_ = server.Close()
		close(closed)
	}()
	_ = server.Close() // a second close, as Server.Close does, must not cut the wait short
	waitFor(t, "Close to give up at the deadline", closed)
	assert.GreaterOrEqual(t, time.Since(started), timeout)
	assert.EqualValues(t, 1, dl.unsent.Load(), "one connection, counted once")

	// The peer must learn at once that the rest is not coming: a reset, not a
	// silent end that a proxy would sit out until its read timeout.
	require.NoError(t, client.SetReadDeadline(time.Now().Add(5*time.Second)))
	_, err := io.Copy(io.Discard, client)
	require.Error(t, err, "a cut connection must not look like a complete response")
	assert.ErrorIs(t, err, syscall.ECONNRESET)
}

// selfSignedCert writes a throwaway certificate and key for 127.0.0.1.
func selfSignedCert(t *testing.T) (certFile, keyFile string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)

	dir := t.TempDir()
	certFile = filepath.Join(dir, "cert.pem")
	keyFile = filepath.Join(dir, "key.pem")
	require.NoError(t, os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))
	require.NoError(t, os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600))
	return certFile, keyFile
}

// The backend serves HTTPS in development; the drain must work through TLS too.
func TestServe_TLSDrainsInFlightRequest(t *testing.T) {
	certFile, keyFile := selfSignedCert(t)
	h := newBlockingHandler()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	result := make(chan error, 1)
	go func() {
		result <- serve(ctx, ln, h, serverOptions{certFile: certFile, keyFile: keyFile, shutdownTimeout: 10 * time.Second})
	}()

	client := &http.Client{Transport: &http.Transport{
		DisableKeepAlives: true,
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // throwaway test certificate
	}}
	type outcome struct {
		body string
		err  error
	}
	slow := make(chan outcome, 1)
	go func() {
		resp, err := client.Get("https://" + ln.Addr().String() + "/slow")
		if err != nil {
			slow <- outcome{err: err}
			return
		}
		defer func() { _ = resp.Body.Close() }()
		body, err := io.ReadAll(resp.Body)
		slow <- outcome{body: string(body), err: err}
	}()
	waitFor(t, "the request to reach the handler", h.entered)

	cancel()
	requireRefused(t, ln.Addr().String())
	close(h.release)

	select {
	case res := <-slow:
		require.NoError(t, res.err)
		assert.Equal(t, "done", res.body)
	case <-time.After(5 * time.Second):
		t.Fatal("the in-flight HTTPS request never completed")
	}
	select {
	case err := <-result:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not return after the drain")
	}
}
