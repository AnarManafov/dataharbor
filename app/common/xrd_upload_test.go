package common

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go-hep.org/x/hep/xrootd"
)

// startTestXRDServer starts the go-hep file-backed XRootD test server on a
// random local port, serving files under a fresh temp dir.
func startTestXRDServer(t *testing.T) (addr string, baseDir string) {
	t.Helper()
	baseDir = t.TempDir()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := xrootd.NewServer(xrootd.NewFSHandler(baseDir), func(err error) {})
	go func() { _ = srv.Serve(listener) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})
	return listener.Addr().String(), baseDir
}

// Regression test for the upload-verification hang: the upload handle is
// created during the session-creation HTTP request, but chunk writes and the
// final close arrive as separate requests long after that first request's
// context has been canceled. The handle's connection must keep delivering
// responses after the cancellation; previously the client's response reader
// shut down with the creating request's context, so the second write and the
// kXR_close reply were never received and CompleteUpload blocked forever.
func TestOpenUploadFileSurvivesRequestContextCancel(t *testing.T) {
	addr, baseDir := startTestXRDServer(t)
	xc := &XRDClient{address: addr, username: "test", logger: GetLogger()}

	reqCtx, cancelReq := context.WithCancel(context.Background())
	handle, err := xc.OpenUploadFile(reqCtx, "", "/data.bin", true)
	require.NoError(t, err)
	cancelReq() // the session-creation HTTP request finishes here

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	chunk1 := []byte("hello ")
	chunk2 := []byte("world")
	require.NoError(t, handle.File().WriteAtContext(ctx, chunk1, 0))
	require.NoError(t, handle.File().WriteAtContext(ctx, chunk2, int64(len(chunk1))))
	require.NoError(t, handle.Close(ctx))

	got, err := os.ReadFile(filepath.Join(baseDir, "data.bin"))
	require.NoError(t, err)
	require.Equal(t, "hello world", string(got))
}
