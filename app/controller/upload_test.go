package controller

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go-hep.org/x/hep/xrootd/xrdfs"

	"github.com/AnarManafov/dataharbor/app/common"
	"github.com/AnarManafov/dataharbor/app/config"
)

// ---------------------------------------------------------------------------
// Pure-helper tests: these do not need any XRootD backend.
// ---------------------------------------------------------------------------

func TestValidateRelPath(t *testing.T) {
	ok := []string{"a.txt", "dir/a.txt", "a/b/c.bin", "weird name.txt"}
	bad := []string{
		"",
		"/abs.txt",
		"../escape",
		"a/../b",
		"./a",
		"a//b",
		"a/\x00/b",
		"a\nb",
	}
	for _, p := range ok {
		if err := validateRelPath(p); err != nil {
			t.Errorf("expected %q to be valid, got: %v", p, err)
		}
	}
	for _, p := range bad {
		if err := validateRelPath(p); err == nil {
			t.Errorf("expected %q to be invalid", p)
		}
	}
}

func TestIsValidSha256Hex(t *testing.T) {
	good := hex.EncodeToString(sha256.New().Sum(nil))
	require.Len(t, good, 64)
	assert.True(t, isValidSha256Hex(good))
	assert.True(t, isValidSha256Hex(strings.ToUpper(good)))

	assert.False(t, isValidSha256Hex(""))
	assert.False(t, isValidSha256Hex(good[:63]))
	assert.False(t, isValidSha256Hex(good+"0"))
	assert.False(t, isValidSha256Hex(strings.Repeat("Z", 64)))
}

func TestJoinXRDPath(t *testing.T) {
	assert.Equal(t, "/data/a.txt", joinXRDPath("/data", "a.txt"))
	assert.Equal(t, "/data/sub/a.txt", joinXRDPath("/data/", "sub/a.txt"))
	assert.Equal(t, "/a.txt", joinXRDPath("/", "a.txt"))
}

func TestNewOpaqueID_UniqueAndPrefixed(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := newOpaqueID("up_")
		require.True(t, strings.HasPrefix(id, "up_"))
		require.Len(t, id, 3+32) // prefix + 16 bytes hex
		assert.False(t, seen[id], "collision on %s", id)
		seen[id] = true
	}
}

func TestParseDurationOrDefault(t *testing.T) {
	assert.Equal(t, time.Second, parseDurationOrDefault("1s", time.Hour))
	assert.Equal(t, time.Hour, parseDurationOrDefault("", time.Hour))
	assert.Equal(t, time.Hour, parseDurationOrDefault("not a duration", time.Hour))
}

func TestStaleTempUploadID(t *testing.T) {
	const suffix = ".dh-upload"
	id := strings.Repeat("ab", 16) // 32 hex chars

	got, ok := staleTempUploadID("video.vmdk"+suffix+"."+id, "video.vmdk", suffix)
	require.True(t, ok)
	assert.Equal(t, id, got)

	cases := []struct {
		name  string
		entry string
		base  string
	}{
		{"different destination", "other.bin" + suffix + "." + id, "video.vmdk"},
		{"the destination itself", "video.vmdk", "video.vmdk"},
		{"missing id", "video.vmdk" + suffix + ".", "video.vmdk"},
		{"id too short", "video.vmdk" + suffix + "." + id[:31], "video.vmdk"},
		{"id too long", "video.vmdk" + suffix + "." + id + "0", "video.vmdk"},
		{"id not hex", "video.vmdk" + suffix + "." + strings.Repeat("z", 32), "video.vmdk"},
		{"suffix only as infix", "video.vmdk.backup" + suffix + "." + id, "video.vmdk"},
		{"no suffix", "video.vmdk." + id, "video.vmdk"},
	}
	for _, tc := range cases {
		if _, ok := staleTempUploadID(tc.entry, tc.base, suffix); ok {
			t.Errorf("%s: expected %q to NOT match base %q", tc.name, tc.entry, tc.base)
		}
	}

	// A temp file for "video.vmdk.backup" must not match base "video.vmdk"...
	_, ok = staleTempUploadID("video.vmdk.backup"+suffix+"."+id, "video.vmdk", suffix)
	assert.False(t, ok)
	// ...but must match its own base.
	_, ok = staleTempUploadID("video.vmdk.backup"+suffix+"."+id, "video.vmdk.backup", suffix)
	assert.True(t, ok)
}

// TestCollectStaleTemps verifies the orphan-sweep decision logic: temp files
// of dead sessions are selected for removal, temps of live sessions and
// unrelated entries are left alone.
func TestCollectStaleTemps(t *testing.T) {
	const suffix = ".dh-upload"
	orphanID := strings.Repeat("ab", 16)
	liveID := strings.Repeat("cd", 16)

	entries := []xrdfs.EntryStat{
		{EntryName: "big.vmdk" + suffix + "." + orphanID},  // orphan -> sweep
		{EntryName: "big.vmdk" + suffix + "." + liveID},    // live session -> keep
		{EntryName: "big.vmdk"},                            // the destination itself -> keep
		{EntryName: "other.bin" + suffix + "." + orphanID}, // different destination -> keep
		{EntryName: "notes.txt"},                           // unrelated -> keep
	}
	isLive := func(id string) bool { return id == liveID }

	stale := collectStaleTemps(entries, "/data/user", "big.vmdk", suffix, isLive)
	assert.Equal(t, []string{"/data/user/big.vmdk" + suffix + "." + orphanID}, stale)
}

// ---------------------------------------------------------------------------
// Handler tests: CreateUploadSession validation paths (short-circuit before
// any XRootD call is made).
// ---------------------------------------------------------------------------

func newUploadTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/xrd/upload/limits", GetTransferLimits)
	r.POST("/api/v1/xrd/upload/session", CreateUploadSession)
	r.PUT("/api/v1/xrd/upload/:uploadId/chunk", UploadChunk)
	r.POST("/api/v1/xrd/upload/:uploadId/complete", CompleteUpload)
	r.DELETE("/api/v1/xrd/upload/:uploadId", AbortUpload)
	r.GET("/api/v1/xrd/upload/:uploadId/status", GetUploadStatus)
	return r
}

// ensureUploadDefaults injects sane upload limits into the already-installed
// test config. This plays nicely with the TestMain setup which only configures
// base XRD/server fields.
func ensureUploadDefaults(t *testing.T) {
	t.Helper()
	cfg := config.GetConfig()
	cfg.XRD.Upload = config.UploadConfig{
		Enabled:              true,
		MaxFileSize:          int64(10) * 1024 * 1024 * 1024,
		MaxBatchSize:         int64(50) * 1024 * 1024 * 1024,
		MaxFilesPerBatch:     100,
		MaxConcurrentPerUser: 2,
		ChunkSize:            8 * 1024 * 1024,
		SessionTTL:           "2h",
		TempSuffix:           ".dh-upload",
		AllowOverwrite:       false,
		ChecksumAlgo:         "sha256",
	}
	config.SetConfig(cfg)
}

func postJSON(r http.Handler, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCreateUploadSession_FeatureDisabled(t *testing.T) {
	ensureUploadDefaults(t)
	cfg := config.GetConfig()
	cfg.XRD.Upload.Enabled = false
	config.SetConfig(cfg)
	t.Cleanup(ensureDefaultsCleanup(t))

	w := postJSON(newUploadTestRouter(), "/api/v1/xrd/upload/session",
		CreateSessionRequest{
			DestDir: "/data",
			Files: []UploadFileRequest{
				{RelPath: "a.txt", Size: 10},
			},
		})
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestCreateUploadSession_MalformedJSON(t *testing.T) {
	ensureUploadDefaults(t)
	r := newUploadTestRouter()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/xrd/upload/session",
		strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateUploadSession_InvalidDestDir(t *testing.T) {
	ensureUploadDefaults(t)
	w := postJSON(newUploadTestRouter(), "/api/v1/xrd/upload/session",
		CreateSessionRequest{
			DestDir: "../escape",
			Files:   []UploadFileRequest{{RelPath: "a.txt", Size: 1}},
		})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "destDir")
}

func TestCreateUploadSession_EmptyFiles(t *testing.T) {
	ensureUploadDefaults(t)
	w := postJSON(newUploadTestRouter(), "/api/v1/xrd/upload/session",
		map[string]any{"destDir": "/data", "files": []any{}})
	// gin binding catches "required" before our len() check, so either 400
	// shape is acceptable as long as it's 4xx.
	assert.GreaterOrEqual(t, w.Code, 400)
	assert.Less(t, w.Code, 500)
}

func TestCreateUploadSession_TooManyFiles(t *testing.T) {
	ensureUploadDefaults(t)
	cfg := config.GetConfig()
	cfg.XRD.Upload.MaxFilesPerBatch = 2
	config.SetConfig(cfg)
	t.Cleanup(ensureDefaultsCleanup(t))

	req := CreateSessionRequest{
		DestDir: "/data",
		Files: []UploadFileRequest{
			{RelPath: "a.txt", Size: 1},
			{RelPath: "b.txt", Size: 1},
			{RelPath: "c.txt", Size: 1},
		},
	}
	w := postJSON(newUploadTestRouter(), "/api/v1/xrd/upload/session", req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, strings.ToLower(w.Body.String()), "too many files")
}

func TestCreateUploadSession_BadRelPath(t *testing.T) {
	ensureUploadDefaults(t)
	w := postJSON(newUploadTestRouter(), "/api/v1/xrd/upload/session",
		CreateSessionRequest{
			DestDir: "/data",
			Files:   []UploadFileRequest{{RelPath: "../x", Size: 1}},
		})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "relPath")
}

func TestCreateUploadSession_FileTooLarge(t *testing.T) {
	ensureUploadDefaults(t)
	cfg := config.GetConfig()
	cfg.XRD.Upload.MaxFileSize = 100
	config.SetConfig(cfg)
	t.Cleanup(ensureDefaultsCleanup(t))

	w := postJSON(newUploadTestRouter(), "/api/v1/xrd/upload/session",
		CreateSessionRequest{
			DestDir: "/data",
			Files:   []UploadFileRequest{{RelPath: "a.txt", Size: 1000}},
		})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, strings.ToLower(w.Body.String()), "max file size")
}

func TestCreateUploadSession_BatchTooLarge(t *testing.T) {
	ensureUploadDefaults(t)
	cfg := config.GetConfig()
	cfg.XRD.Upload.MaxBatchSize = 150
	cfg.XRD.Upload.MaxFileSize = 1000
	config.SetConfig(cfg)
	t.Cleanup(ensureDefaultsCleanup(t))

	w := postJSON(newUploadTestRouter(), "/api/v1/xrd/upload/session",
		CreateSessionRequest{
			DestDir: "/data",
			Files: []UploadFileRequest{
				{RelPath: "a.txt", Size: 100},
				{RelPath: "b.txt", Size: 100},
			},
		})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, strings.ToLower(w.Body.String()), "batch size")
}

func TestCreateUploadSession_UnknownConflictAction(t *testing.T) {
	ensureUploadDefaults(t)
	w := postJSON(newUploadTestRouter(), "/api/v1/xrd/upload/session",
		CreateSessionRequest{
			DestDir: "/data",
			Files: []UploadFileRequest{
				{RelPath: "a.txt", Size: 10, OnConflict: "panic"},
			},
		})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, strings.ToLower(w.Body.String()), "onconflict")
}

func TestCreateUploadSession_OverwriteForbiddenByServer(t *testing.T) {
	ensureUploadDefaults(t)
	// AllowOverwrite is false by default
	w := postJSON(newUploadTestRouter(), "/api/v1/xrd/upload/session",
		CreateSessionRequest{
			DestDir: "/data",
			Files: []UploadFileRequest{
				{RelPath: "a.txt", Size: 10, OnConflict: ConflictActionOverwrite},
			},
		})
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCreateUploadSession_ReservedSuffix(t *testing.T) {
	ensureUploadDefaults(t)
	w := postJSON(newUploadTestRouter(), "/api/v1/xrd/upload/session",
		CreateSessionRequest{
			DestDir: "/data",
			Files:   []UploadFileRequest{{RelPath: "report.dh-upload", Size: 10}},
		})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, strings.ToLower(w.Body.String()), "reserved upload suffix")
}

// ---------------------------------------------------------------------------
// Handler tests: chunk / status / abort / complete without a real XRootD
// backend. These use a manually-primed session in uploadStore so that we
// exercise the state machine branches directly (offset mismatch, wrong user,
// bad sha, completed state, etc).
// ---------------------------------------------------------------------------

// primeSession inserts a session with a no-op handle so that UploadChunk's
// handle.File().WriteAtContext call is bypassed. Since we cannot fake the
// xrdfs.File interface without a lot of stubs, tests that need the write path
// are marked to short-circuit before that call.
func primeSession(t *testing.T, userKey string, size int64) *uploadSession {
	t.Helper()
	sess := &uploadSession{
		ID:             newOpaqueID("up_"),
		UserKey:        userKey,
		UserToken:      "",
		DestPath:       "/data/a.txt",
		TempPath:       "/data/a.txt.dh-upload",
		Size:           size,
		ChunkSize:      8 * 1024 * 1024,
		CreatedAt:      time.Now(),
		ExpiresAt:      time.Now().Add(time.Hour),
		batch:          &batchSlot{refs: 1, release: func() {}},
		hasher:         sha256.New(),
		state:          "uploading",
		lastActivityAt: time.Now(),
	}
	uploadStore.add(sess)
	t.Cleanup(func() { uploadStore.delete(sess.ID) })
	return sess
}

func TestUploadChunk_UnknownID(t *testing.T) {
	ensureUploadDefaults(t)
	r := newUploadTestRouter()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/xrd/upload/up_missing/chunk?offset=0",
		bytes.NewReader([]byte("hi")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUploadChunk_BadOffset(t *testing.T) {
	ensureUploadDefaults(t)
	sess := primeSession(t, "anonymous", 100)
	r := newUploadTestRouter()
	req := httptest.NewRequest(http.MethodPut,
		"/api/v1/xrd/upload/"+sess.ID+"/chunk?offset=abc",
		bytes.NewReader([]byte("x")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUploadChunk_OffsetMismatch(t *testing.T) {
	ensureUploadDefaults(t)
	sess := primeSession(t, "anonymous", 100)
	sess.BytesReceived = 10

	r := newUploadTestRouter()
	req := httptest.NewRequest(http.MethodPut,
		"/api/v1/xrd/upload/"+sess.ID+"/chunk?offset=0",
		bytes.NewReader([]byte("xx")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, strings.ToLower(w.Body.String()), "offset mismatch")
}

func TestUploadChunk_ChunkTooLarge(t *testing.T) {
	ensureUploadDefaults(t)
	cfg := config.GetConfig()
	cfg.XRD.Upload.ChunkSize = 4
	config.SetConfig(cfg)
	t.Cleanup(ensureDefaultsCleanup(t))

	sess := primeSession(t, "anonymous", 100)

	r := newUploadTestRouter()
	req := httptest.NewRequest(http.MethodPut,
		"/api/v1/xrd/upload/"+sess.ID+"/chunk?offset=0",
		bytes.NewReader([]byte("this-is-longer-than-4")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}

func TestUploadChunk_ExceedsDeclaredSize(t *testing.T) {
	ensureUploadDefaults(t)
	sess := primeSession(t, "anonymous", 5)
	r := newUploadTestRouter()
	req := httptest.NewRequest(http.MethodPut,
		"/api/v1/xrd/upload/"+sess.ID+"/chunk?offset=0",
		bytes.NewReader([]byte("toolong")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, strings.ToLower(w.Body.String()), "exceed declared size")
}

// TestUploadChunk_ReadErrorKeepsSessionResumable verifies that a truncated
// chunk body (client cancellation / network drop) does NOT destroy the session:
// BytesReceived stays put and the session remains "uploading" so the client can
// resume from the last committed offset.
func TestUploadChunk_ReadErrorKeepsSessionResumable(t *testing.T) {
	ensureUploadDefaults(t)
	sess := primeSession(t, "anonymous", 100)
	// A non-nil handle so we get past the finalize guard; the write path is
	// never reached because the body read fails first.
	sess.handle = &common.UploadHandle{}

	r := newUploadTestRouter()
	// Declare a Content-Length larger than the body we actually send so
	// io.ReadFull returns ErrUnexpectedEOF.
	req := httptest.NewRequest(http.MethodPut,
		"/api/v1/xrd/upload/"+sess.ID+"/chunk?offset=0",
		strings.NewReader("hi"))
	req.ContentLength = 10
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	// Session must still be alive and resumable.
	got, ok := uploadStore.get(sess.ID)
	require.True(t, ok, "session must not be deleted on a transient read error")
	got.mu.Lock()
	defer got.mu.Unlock()
	assert.Equal(t, "uploading", got.state)
	assert.Equal(t, int64(0), got.BytesReceived)
}

func TestGetUploadStatus_WrongUser(t *testing.T) {
	ensureUploadDefaults(t)
	sess := primeSession(t, "someone_else", 10)
	r := newUploadTestRouter()
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/xrd/upload/"+sess.ID+"/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetUploadStatus_OK(t *testing.T) {
	ensureUploadDefaults(t)
	sess := primeSession(t, "anonymous", 100)
	sess.BytesReceived = 42

	r := newUploadTestRouter()
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/xrd/upload/"+sess.ID+"/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var env struct {
		Data UploadStatusResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, sess.ID, env.Data.UploadID)
	assert.Equal(t, int64(100), env.Data.Size)
	assert.Equal(t, int64(42), env.Data.BytesReceived)
	assert.Equal(t, "uploading", env.Data.State)
}

func TestAbortUpload_UnknownID(t *testing.T) {
	ensureUploadDefaults(t)
	r := newUploadTestRouter()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/xrd/upload/up_nope", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestCompleteUpload_ShaMismatch: the checksum travels with the complete
// request (computed client-side while the chunks uploaded). A digest that
// does not match the server-side hash of the received bytes must fail.
func TestCompleteUpload_ShaMismatch(t *testing.T) {
	ensureUploadDefaults(t)
	sess := primeSession(t, "anonymous", 5)
	_, _ = sess.hasher.Write([]byte("hello"))
	sess.BytesReceived = 5

	w := postJSON(newUploadTestRouter(),
		"/api/v1/xrd/upload/"+sess.ID+"/complete",
		CompleteUploadRequest{Sha256: strings.Repeat("a", 64)})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, strings.ToLower(w.Body.String()), "sha256 mismatch")
}

// TestCompleteUpload_ChecksumRequired: a complete request without a sha256 is
// a client error that must NOT kill the session.
func TestCompleteUpload_ChecksumRequired(t *testing.T) {
	ensureUploadDefaults(t)
	sess := primeSession(t, "anonymous", 5)
	sess.BytesReceived = 5

	r := newUploadTestRouter()
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/xrd/upload/"+sess.ID+"/complete", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, strings.ToLower(w.Body.String()), "sha256 is required")

	got, ok := uploadStore.get(sess.ID)
	require.True(t, ok, "session must survive a missing-checksum complete attempt")
	got.mu.Lock()
	defer got.mu.Unlock()
	assert.Equal(t, "uploading", got.state)
}

func TestCompleteUpload_BadBodySha(t *testing.T) {
	ensureUploadDefaults(t)
	sess := primeSession(t, "anonymous", 5)
	sess.BytesReceived = 5

	w := postJSON(newUploadTestRouter(),
		"/api/v1/xrd/upload/"+sess.ID+"/complete",
		CompleteUploadRequest{Sha256: "deadbeef"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "64 hex")
}

func TestCompleteUpload_NotAllBytes(t *testing.T) {
	ensureUploadDefaults(t)
	sess := primeSession(t, "anonymous", 100)
	sess.BytesReceived = 50

	w := postJSON(newUploadTestRouter(),
		"/api/v1/xrd/upload/"+sess.ID+"/complete",
		CompleteUploadRequest{Sha256: sha64()})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, strings.ToLower(w.Body.String()), "not all bytes")
}

func TestGetTransferLimits(t *testing.T) {
	ensureUploadDefaults(t)
	r := newUploadTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/xrd/upload/limits", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var env struct {
		Data TransferLimitsResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.True(t, env.Data.Upload.Enabled)
	assert.Greater(t, env.Data.Upload.MaxFileSize, int64(0))
	assert.Equal(t, "sha256", env.Data.Upload.ChecksumAlgo)
}

// TestAbortUpload_CleansUpSession: a client cancel must tear the session down
// completely — removed from the store and the batch concurrency slot released
// (the temp-file removal is attempted too; against the unreachable test XRD
// server it fails and is logged, which must not block the rest).
func TestAbortUpload_CleansUpSession(t *testing.T) {
	ensureUploadDefaults(t)
	sess := primeSession(t, "anonymous", 100)
	released := false
	sess.batch = &batchSlot{refs: 1, release: func() { released = true }}

	r := newUploadTestRouter()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/xrd/upload/"+sess.ID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	_, ok := uploadStore.get(sess.ID)
	assert.False(t, ok, "aborted session must be removed from the store")
	assert.True(t, released, "aborting the last file of a batch must release its slot")

	sess.mu.Lock()
	defer sess.mu.Unlock()
	assert.Equal(t, "aborted", sess.state)
}

// wedgedUploadFile simulates an upload handle whose connection died: closing
// it never completes. Only Close is used; other xrdfs.File methods are never
// called by the teardown path.
type wedgedUploadFile struct{ xrdfs.File }

func (wedgedUploadFile) Close(context.Context) error {
	time.Sleep(10 * time.Second) // far beyond the test's teardown timeout
	return nil
}

// TestAbortUpload_WedgedHandleStillCleansUp is the regression test for the
// production incident where one dead XRootD connection poisoned everything:
// the handle close blocked forever inside the abort, so the temp file was
// never removed, the slot never released, and the session stayed registered —
// which also made the orphan sweep skip the temp ("live" session) and the
// janitor ignore it (no longer "uploading"). Teardown must abandon the wedged
// close at the deadline and finish the cleanup.
func TestAbortUpload_WedgedHandleStillCleansUp(t *testing.T) {
	ensureUploadDefaults(t)
	oldTimeout := teardownCloseTimeout
	teardownCloseTimeout = 100 * time.Millisecond
	t.Cleanup(func() { teardownCloseTimeout = oldTimeout })

	sess := primeSession(t, "anonymous", 100)
	sess.handle = common.NewUploadHandle(nil, wedgedUploadFile{}, sess.TempPath)
	released := false
	sess.batch = &batchSlot{refs: 1, release: func() { released = true }}

	start := time.Now()
	abortSession(sess, "test: client cancel with dead connection")
	require.Less(t, time.Since(start), 5*time.Second,
		"teardown must not wait for the wedged close")

	_, ok := uploadStore.get(sess.ID)
	assert.False(t, ok, "session must leave the store despite the wedged handle")
	assert.True(t, released, "slot must be released despite the wedged handle")

	sess.mu.Lock()
	defer sess.mu.Unlock()
	assert.Equal(t, "aborted", sess.state)
}

// ---------------------------------------------------------------------------
// Session-store tests
// ---------------------------------------------------------------------------

func TestUploadSessionStore_ReapExpired(t *testing.T) {
	store := &uploadSessionStore{sessions: make(map[string]*uploadSession)}
	live := &uploadSession{
		ID: "live", UserKey: "u", TempPath: "/tmp/x",
		state: "uploading", ExpiresAt: time.Now().Add(time.Hour),
		lastActivityAt: time.Now(),
	}
	store.add(live)
	// A fresh, recently-active session must not be reaped. (The reaping action
	// itself routes through the package-level abortSession + XRootD client, so
	// it is exercised by integration tests; the decision logic is covered by
	// TestReapReason below.)
	store.reapExpired(time.Hour)
	_, ok := store.get("live")
	assert.True(t, ok)
}

// TestReapReason verifies the pure decision logic that drives session reaping.
func TestReapReason(t *testing.T) {
	now := time.Now()
	idle := 10 * time.Minute

	cases := []struct {
		name    string
		sess    *uploadSession
		idleTTL time.Duration
		want    bool // true => expect a non-empty reap reason
	}{
		{
			name:    "active and recent => keep",
			sess:    &uploadSession{state: "uploading", ExpiresAt: now.Add(time.Hour), lastActivityAt: now},
			idleTTL: idle,
			want:    false,
		},
		{
			name:    "past absolute lifetime => reap",
			sess:    &uploadSession{state: "uploading", ExpiresAt: now.Add(-time.Minute), lastActivityAt: now},
			idleTTL: idle,
			want:    true,
		},
		{
			name:    "idle too long => reap",
			sess:    &uploadSession{state: "uploading", ExpiresAt: now.Add(time.Hour), lastActivityAt: now.Add(-time.Hour)},
			idleTTL: idle,
			want:    true,
		},
		{
			name:    "idle TTL disabled => keep despite inactivity",
			sess:    &uploadSession{state: "uploading", ExpiresAt: now.Add(time.Hour), lastActivityAt: now.Add(-time.Hour)},
			idleTTL: 0,
			want:    false,
		},
		{
			name:    "terminal session => never reaped",
			sess:    &uploadSession{state: "completed", ExpiresAt: now.Add(-time.Hour), lastActivityAt: now.Add(-time.Hour)},
			idleTTL: idle,
			want:    false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := reapReason(tc.sess, now, tc.idleTTL) != ""
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestBatchSlot_RefCountReleasesOnce verifies the per-batch slot is released
// exactly once, only after the last file in the batch finishes. This is the
// mechanism that lets a single session contain more files than the per-user
// concurrency cap without exhausting slots.
func TestBatchSlot_RefCountReleasesOnce(t *testing.T) {
	var released int
	b := &batchSlot{release: func() { released++ }}

	// Three files accepted into one batch.
	b.addRef()
	b.addRef()
	b.addRef()

	b.done()
	assert.Equal(t, 0, released, "slot must not release while files remain")
	b.done()
	assert.Equal(t, 0, released, "slot must not release while a file remains")
	b.done()
	assert.Equal(t, 1, released, "slot releases once the last file finishes")

	// Extra done() calls are safe no-ops.
	b.done()
	assert.Equal(t, 1, released, "slot must not release more than once")
}

// TestBatchSlot_ReleaseNow covers the rollback path: the whole batch is
// discarded and the slot freed immediately, and a later done() does not
// double-release.
func TestBatchSlot_ReleaseNow(t *testing.T) {
	var released int
	b := &batchSlot{release: func() { released++ }}
	b.addRef()
	b.releaseNow()
	assert.Equal(t, 1, released)
	b.done()
	assert.Equal(t, 1, released, "done after releaseNow must not double-release")
}

// TestBatchSlot_NilSafe ensures the helpers tolerate a nil receiver.
func TestBatchSlot_NilSafe(t *testing.T) {
	var b *batchSlot
	assert.NotPanics(t, func() {
		b.addRef()
		b.done()
		b.releaseNow()
	})
}

// ---------------------------------------------------------------------------
// Concurrency: two creators racing on slot acquire
// ---------------------------------------------------------------------------

func TestUploadSlots_AreShared(t *testing.T) {
	ensureUploadDefaults(t)
	// Reset the once and the slot manager so we get a fresh instance with the
	// test cap.
	uploadSlotsOnce = sync.Once{}
	uploadSlots = nil
	cfg := config.GetConfig()
	cfg.XRD.Upload.MaxConcurrentPerUser = 1
	config.SetConfig(cfg)
	t.Cleanup(func() {
		uploadSlotsOnce = sync.Once{}
		uploadSlots = nil
		ensureDefaultsCleanup(t)()
	})

	slots := getUploadSlots()
	rel, err := slots.Acquire("alice")
	require.NoError(t, err)
	_, err = slots.Acquire("alice")
	assert.Error(t, err, "second concurrent acquire must fail")
	rel()
	rel2, err := slots.Acquire("alice")
	require.NoError(t, err)
	rel2()
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// sha64 returns a deterministic valid-looking all-zero 64-char hex string. It
// is only syntactically a SHA-256; callers that need semantic correctness
// compute their own.
func sha64() string { return strings.Repeat("0", 64) }

// ensureDefaultsCleanup returns a func that restores the baseline upload
// defaults used by ensureUploadDefaults. Useful when a test has mutated the
// config.
func ensureDefaultsCleanup(t *testing.T) func() {
	t.Helper()
	return func() { ensureUploadDefaults(t) }
}

// Silence an unused-import warning when this file's helpers evolve.
var (
	_ = context.Background
	_ = common.GetXRDClient
)
