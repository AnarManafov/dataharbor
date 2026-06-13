package controller

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go-hep.org/x/hep/xrootd/xrdfs"

	"github.com/AnarManafov/dataharbor/app/common"
	"github.com/AnarManafov/dataharbor/app/config"
	"github.com/AnarManafov/dataharbor/app/middleware"
	"github.com/AnarManafov/dataharbor/app/response"
)

// ---------------------------------------------------------------------------
// Public types (request / response payloads)
// ---------------------------------------------------------------------------

// ConflictAction tells the server what to do when a destination file already
// exists on the XRootD server.
type ConflictAction string

const (
	ConflictActionFail      ConflictAction = "fail"      // (default) reject the upload for this file
	ConflictActionSkip      ConflictAction = "skip"      // silently skip the file
	ConflictActionOverwrite ConflictAction = "overwrite" // write to a temp path then atomically rename
	ConflictActionRename    ConflictAction = "rename"    // append a numeric suffix to the destination
)

// CreateSessionRequest is the body of POST /api/v1/xrd/upload/session.
type CreateSessionRequest struct {
	// DestDir is the absolute XRootD directory under which files will be placed.
	DestDir string `json:"destDir" binding:"required"`

	// Files lists every file the client intends to upload in this session.
	Files []UploadFileRequest `json:"files" binding:"required,dive"`
}

// UploadFileRequest describes one file the client wants to upload.
type UploadFileRequest struct {
	// RelPath is the destination path relative to DestDir. For a flat upload
	// this is just the filename. For folder uploads it preserves subdirectories
	// (e.g. "dataset/foo.txt").
	RelPath string `json:"relPath" binding:"required"`

	// Size is the total size in bytes of the file about to be uploaded.
	Size int64 `json:"size" binding:"required"`

	// OnConflict controls behavior when RelPath already exists at DestDir.
	// Defaults to "fail" when empty. See ConflictAction* constants.
	OnConflict ConflictAction `json:"onConflict"`
}

// UploadFileSession is returned for each file that the server accepted into
// the session. An accepted file has an UploadID the client uses for chunks.
type UploadFileSession struct {
	RelPath  string `json:"relPath"`
	UploadID string `json:"uploadId,omitempty"` // empty when Status is "skipped"
	DestPath string `json:"destPath"`           // final absolute path on XRootD
	Status   string `json:"status"`             // "accepted" | "skipped"
	Conflict string `json:"conflict"`           // "none" | "exists"
	Reason   string `json:"reason,omitempty"`   // populated when Status is "skipped"
}

// CreateSessionResponse is returned from POST /api/v1/xrd/upload/session.
type CreateSessionResponse struct {
	SessionID string              `json:"sessionId"`
	ChunkSize int                 `json:"chunkSize"`
	Files     []UploadFileSession `json:"files"`
}

// UploadStatusResponse is returned by GET /api/v1/xrd/upload/:uploadId/status
// and used by the client to resume after a disconnect.
type UploadStatusResponse struct {
	UploadID       string `json:"uploadId"`
	Size           int64  `json:"size"`
	BytesReceived  int64  `json:"bytesReceived"`
	ChunkSize      int    `json:"chunkSize"`
	State          string `json:"state"` // "uploading" | "completed" | "failed" | "aborted"
	FailureMessage string `json:"failureMessage,omitempty"`
}

// CompleteUploadRequest is the body of POST .../complete. The client hashes
// the file while its chunks upload and supplies the digest here, where the
// server needs it for verification.
type CompleteUploadRequest struct {
	Sha256 string `json:"sha256" binding:"required"`
}

// CompleteUploadResponse is returned by POST /api/v1/xrd/upload/:uploadId/complete.
type CompleteUploadResponse struct {
	UploadID     string `json:"uploadId"`
	DestPath     string `json:"destPath"`
	Size         int64  `json:"size"`
	Sha256       string `json:"sha256"`
	DurationMs   int64  `json:"durationMs"`
	BytesWritten int64  `json:"bytesWritten"`
}

// TransferLimitsResponse is returned by GET /api/v1/xrd/upload/limits. It
// bundles both upload and download limits so the UI can surface them in a
// single popover.
type TransferLimitsResponse struct {
	Upload   uploadLimitsPayload   `json:"upload"`
	Download downloadLimitsPayload `json:"download"`
}

type uploadLimitsPayload struct {
	Enabled              bool   `json:"enabled"`
	MaxFileSize          int64  `json:"maxFileSize"`
	MaxBatchSize         int64  `json:"maxBatchSize"`
	MaxFilesPerBatch     int    `json:"maxFilesPerBatch"`
	MaxConcurrentPerUser int    `json:"maxConcurrentPerUser"`
	ChunkSize            int    `json:"chunkSize"`
	AllowOverwrite       bool   `json:"allowOverwrite"`
	ChecksumAlgo         string `json:"checksumAlgo"`
}

type downloadLimitsPayload struct {
	MaxBatchFiles    int  `json:"maxBatchFiles"`
	MaxBatchSizeMB   int  `json:"maxBatchSizeMB"`
	BatchCompression bool `json:"batchCompression"`
}

// ---------------------------------------------------------------------------
// Internal session state
// ---------------------------------------------------------------------------

// uploadSession holds the server-side state for one in-progress file upload.
// It is owned by uploadSessionStore; its mutex serializes access to BytesReceived,
// handle, hasher, and state transitions.
type uploadSession struct {
	ID        string
	UserKey   string
	UserToken string // captured at creation; chunks run under this token
	DestPath  string // final absolute path on XRootD
	TempPath  string // in-progress path (DestPath + TempSuffix + per-session id)
	Size      int64
	ChunkSize int
	Overwrite bool // chosen conflict action required overwrite semantics
	CreatedAt time.Time
	ExpiresAt time.Time

	// batch is the per-upload-session-group slot owner. Every file accepted in
	// one CreateUploadSession call shares a single concurrency slot; the slot is
	// released only once the last file in the batch reaches a terminal state.
	batch *batchSlot

	// writeMu serializes chunk writes against each other and against handle
	// teardown (close). It is always acquired BEFORE mu when both are needed.
	// Holding it across the (potentially long) XRootD write guarantees the
	// handle cannot be closed mid-write and that two chunks cannot race on the
	// same offset.
	writeMu sync.Mutex

	mu            sync.Mutex
	BytesReceived int64
	handle        *common.UploadHandle
	hasher        interface {
		io.Writer
		Sum(b []byte) []byte
	}
	state          string // "uploading" | "completed" | "failed" | "aborted"
	failureMessage string
	lastActivityAt time.Time // updated on each committed chunk; drives idle reaping
}

// batchSlot owns exactly one concurrency slot shared by all file sessions
// created in a single CreateUploadSession call. The underlying SlotRelease is
// idempotent, but we ref-count child sessions so the slot is released only when
// the last file in the batch finishes (completes, aborts, fails, or is reaped).
type batchSlot struct {
	mu      sync.Mutex
	refs    int
	release common.SlotRelease
}

// addRef registers one child file session against the batch.
func (b *batchSlot) addRef() {
	if b == nil {
		return
	}
	b.mu.Lock()
	b.refs++
	b.mu.Unlock()
}

// done marks one child file session as terminal. When the count reaches zero
// the shared slot is released. Safe to call once per child session.
func (b *batchSlot) done() {
	if b == nil {
		return
	}
	b.mu.Lock()
	if b.refs > 0 {
		b.refs--
	}
	releaseNow := b.refs == 0 && b.release != nil
	rel := b.release
	if releaseNow {
		b.release = nil
	}
	b.mu.Unlock()
	if releaseNow {
		rel()
	}
}

// releaseNow unconditionally releases the slot regardless of ref count. Used on
// the session-creation rollback path where the whole batch is being discarded.
func (b *batchSlot) releaseNow() {
	if b == nil {
		return
	}
	b.mu.Lock()
	rel := b.release
	b.release = nil
	b.mu.Unlock()
	if rel != nil {
		rel()
	}
}

// uploadSessionStore is the in-memory registry of active upload sessions. It
// is safe for concurrent use. A single instance is created at package init.
type uploadSessionStore struct {
	mu       sync.Mutex
	sessions map[string]*uploadSession // key: upload id
}

var (
	uploadStore       = &uploadSessionStore{sessions: make(map[string]*uploadSession)}
	uploadSlotsOnce   sync.Once
	uploadSlots       *common.SlotManager
	uploadJanitorOnce sync.Once

	// teardownCloseTimeout bounds the handle close during session teardown. A
	// dead connection wedges the close; after this long it is abandoned so the
	// rest of the cleanup (temp removal, slot release) still runs. Variable so
	// tests can shorten it.
	teardownCloseTimeout = 15 * time.Second
)

// getUploadSlots returns the lazily-initialized concurrency slot manager for
// uploads. The per-user cap comes from config.
func getUploadSlots() *common.SlotManager {
	uploadSlotsOnce.Do(func() {
		cfg := config.GetConfig()
		uploadSlots = common.NewSlotManager("upload", cfg.XRD.Upload.MaxConcurrentPerUser)
	})
	return uploadSlots
}

// startUploadJanitor spins up (once) a goroutine that reaps expired sessions.
// Called from route setup.
func startUploadJanitor() {
	uploadJanitorOnce.Do(func() {
		go func() {
			cfg := config.GetConfig()
			idleTTL := parseDurationOrDefault(cfg.XRD.Upload.IdleTTL, 10*time.Minute)
			ticker := time.NewTicker(1 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				uploadStore.reapExpired(idleTTL)
			}
		}()
	})
}

// StartUploadJanitor is the exported entry point used by the router package.
func StartUploadJanitor() { startUploadJanitor() }

func (s *uploadSessionStore) add(sess *uploadSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sess.ID] = sess
}

func (s *uploadSessionStore) get(id string) (*uploadSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	return sess, ok
}

func (s *uploadSessionStore) delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}

// reapExpired iterates the store and aborts any in-progress session that has
// either reached its absolute lifetime (ExpiresAt) or been idle (no committed
// chunk) for longer than idleTTL. The idle timeout bounds the resources an
// abandoned session can pin — slot, open XRootD handle, and temp file — to
// minutes rather than the full (multi-hour) session TTL. Called on a timer.
func (s *uploadSessionStore) reapExpired(idleTTL time.Duration) {
	now := time.Now()
	s.mu.Lock()
	candidates := make([]*uploadSession, 0, len(s.sessions))
	for _, sess := range s.sessions {
		candidates = append(candidates, sess)
	}
	s.mu.Unlock()

	for _, sess := range candidates {
		if reason := reapReason(sess, now, idleTTL); reason != "" {
			common.GetLogger().Warnw("reaping upload session",
				"uploadId", sess.ID, "userKey", sess.UserKey, "dest", sess.DestPath, "reason", reason)
			abortSession(sess, reason)
		}
	}
}

// reapReason reports why an in-progress session should be reaped at time now, or
// "" if it should be left alone. A session is reaped once it passes its absolute
// lifetime (ExpiresAt) or has been idle (no committed chunk) for longer than
// idleTTL. Terminal sessions are never reaped (they are deleted inline).
func reapReason(sess *uploadSession, now time.Time, idleTTL time.Duration) string {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	if sess.state != "uploading" {
		return ""
	}
	if now.After(sess.ExpiresAt) {
		return "session expired (max lifetime reached)"
	}
	if idleTTL > 0 && now.Sub(sess.lastActivityAt) > idleTTL {
		return "session idle timeout (no chunk activity)"
	}
	return ""
}

// ---------------------------------------------------------------------------
// Public handlers
// ---------------------------------------------------------------------------

// GetTransferLimits returns the configured upload and download limits.
// Handler for GET /api/v1/xrd/upload/limits.
func GetTransferLimits(c *gin.Context) {
	cfg := config.GetConfig()
	resp := TransferLimitsResponse{
		Upload: uploadLimitsPayload{
			Enabled:              cfg.XRD.Upload.Enabled,
			MaxFileSize:          cfg.XRD.Upload.MaxFileSize,
			MaxBatchSize:         cfg.XRD.Upload.MaxBatchSize,
			MaxFilesPerBatch:     cfg.XRD.Upload.MaxFilesPerBatch,
			MaxConcurrentPerUser: cfg.XRD.Upload.MaxConcurrentPerUser,
			ChunkSize:            cfg.XRD.Upload.ChunkSize,
			AllowOverwrite:       cfg.XRD.Upload.AllowOverwrite,
			ChecksumAlgo:         cfg.XRD.Upload.ChecksumAlgo,
		},
		Download: downloadLimitsPayload{
			MaxBatchFiles:    cfg.XRD.Download.MaxBatchFiles,
			MaxBatchSizeMB:   cfg.XRD.Download.MaxBatchSizeMB,
			BatchCompression: cfg.XRD.Download.BatchCompression,
		},
	}
	response.Success(c, resp)
}

// CreateUploadSession handles POST /api/v1/xrd/upload/session.
//
// It validates the request against server-side limits, detects destination
// conflicts, allocates a per-file session, and acquires one concurrency slot
// per accepted file. On any mid-flight failure it rolls back all partial state.
func CreateUploadSession(c *gin.Context) {
	cfg := config.GetConfig()
	if !cfg.XRD.Upload.Enabled {
		response.Error(c, http.StatusServiceUnavailable, "Upload is disabled on this server")
		return
	}

	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// --- Input validation --------------------------------------------------
	if err := validateFilePath(req.DestDir); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid destDir: "+err.Error())
		return
	}
	if len(req.Files) == 0 {
		response.Error(c, http.StatusBadRequest, "files must not be empty")
		return
	}
	if len(req.Files) > cfg.XRD.Upload.MaxFilesPerBatch {
		response.Error(c, http.StatusBadRequest,
			fmt.Sprintf("too many files: %d (max %d)", len(req.Files), cfg.XRD.Upload.MaxFilesPerBatch))
		return
	}

	tempSuffix := cfg.XRD.Upload.TempSuffix
	if tempSuffix == "" {
		tempSuffix = ".dh-upload"
	}

	var totalSize int64
	for i, f := range req.Files {
		if err := validateRelPath(f.RelPath); err != nil {
			response.Error(c, http.StatusBadRequest,
				fmt.Sprintf("files[%d]: invalid relPath: %s", i, err.Error()))
			return
		}
		// Reserve the in-progress temp namespace: reject destinations that
		// themselves end in the temp suffix so user files cannot masquerade as
		// (or collide with) another upload's temporary file.
		if strings.HasSuffix(path.Base(f.RelPath), tempSuffix) {
			response.Error(c, http.StatusBadRequest,
				fmt.Sprintf("files[%d]: relPath must not end with the reserved upload suffix %q", i, tempSuffix))
			return
		}
		if f.Size <= 0 {
			response.Error(c, http.StatusBadRequest,
				fmt.Sprintf("files[%d]: size must be > 0", i))
			return
		}
		if f.Size > cfg.XRD.Upload.MaxFileSize {
			response.Error(c, http.StatusBadRequest,
				fmt.Sprintf("files[%d] (%s): size %d exceeds max file size %d",
					i, f.RelPath, f.Size, cfg.XRD.Upload.MaxFileSize))
			return
		}
		if f.OnConflict != "" &&
			f.OnConflict != ConflictActionFail &&
			f.OnConflict != ConflictActionSkip &&
			f.OnConflict != ConflictActionOverwrite &&
			f.OnConflict != ConflictActionRename {
			response.Error(c, http.StatusBadRequest,
				fmt.Sprintf("files[%d]: unknown onConflict=%q", i, f.OnConflict))
			return
		}
		if f.OnConflict == ConflictActionOverwrite && !cfg.XRD.Upload.AllowOverwrite {
			response.Error(c, http.StatusForbidden,
				"Overwrite is disabled on this server (xrd.upload.allow_overwrite=false)")
			return
		}
		totalSize += f.Size
		if totalSize > cfg.XRD.Upload.MaxBatchSize {
			response.Error(c, http.StatusBadRequest,
				fmt.Sprintf("total batch size %d exceeds max %d",
					totalSize, cfg.XRD.Upload.MaxBatchSize))
			return
		}
	}

	// --- Auth + user key ---------------------------------------------------
	userToken, _ := middleware.GetUserToken(c)
	userKey := getUserKey(c)
	ctx := c.Request.Context()

	xrdClient := common.GetXRDClient()

	// --- Per-file processing ----------------------------------------------
	sessionID := newOpaqueID("sess_")
	expiresAt := time.Now().Add(parseDurationOrDefault(cfg.XRD.Upload.SessionTTL, 2*time.Hour))

	// Acquire a single concurrency slot for the whole batch. The per-user cap
	// (xrd.upload.max_concurrent_per_user) limits how many upload *sessions* a
	// user can run at once, NOT how many files a session may contain — so we
	// take exactly one slot here and share it across every file via batchSlot.
	slotRelease, slotErr := getUploadSlots().Acquire(userKey)
	if slotErr != nil {
		response.Error(c, http.StatusTooManyRequests, slotErr.Error())
		return
	}
	batch := &batchSlot{release: slotRelease}

	created := make([]*uploadSession, 0, len(req.Files))
	results := make([]UploadFileSession, 0, len(req.Files))
	sweepCache := make(map[string][]xrdfs.EntryStat)

	rollback := func(fileInfo *UploadFileRequest, reason error) {
		for _, sess := range created {
			if sess.handle != nil {
				_ = sess.handle.Close(ctx)
			}
			uploadStore.delete(sess.ID)
			if sess.TempPath != "" {
				_ = xrdClient.RemoveFile(ctx, userToken, sess.TempPath)
			}
		}
		batch.releaseNow() // whole batch discarded; free the shared slot
		common.GetLogger().Warnf("CreateUploadSession rollback: removed %d partially-created sessions due to error on file %q: %v", len(created), fileInfo.RelPath, reason)
	}

	for _, f := range req.Files {
		destPath := joinXRDPath(req.DestDir, f.RelPath)

		// Detect conflict
		_, exists, statErr := xrdClient.StatPath(ctx, userToken, destPath)
		if statErr != nil {
			rollback(&f, statErr)
			if common.IsAuthError(statErr) {
				response.Error(c, http.StatusForbidden,
					"You don't have permission to access this location. Your access does not include "+
						"read permission for this path — contact your administrator if you believe this is an error.")
				return
			}
			response.Error(c, http.StatusInternalServerError,
				"Failed to check destination: "+statErr.Error())
			return
		}

		action := f.OnConflict
		if action == "" {
			action = ConflictActionFail
		}
		conflictLabel := "none"
		finalDest := destPath

		if exists {
			conflictLabel = "exists"
			switch action {
			case ConflictActionFail:
				results = append(results, UploadFileSession{
					RelPath:  f.RelPath,
					DestPath: destPath,
					Status:   "skipped",
					Conflict: conflictLabel,
					Reason:   "destination exists; onConflict=fail",
				})
				continue
			case ConflictActionSkip:
				results = append(results, UploadFileSession{
					RelPath:  f.RelPath,
					DestPath: destPath,
					Status:   "skipped",
					Conflict: conflictLabel,
					Reason:   "destination exists; onConflict=skip",
				})
				continue
			case ConflictActionRename:
				renamed, rerr := findAvailableRename(ctx, xrdClient, userToken, destPath)
				if rerr != nil {
					rollback(&f, rerr)
					response.Error(c, http.StatusInternalServerError,
						"Failed to compute renamed destination: "+rerr.Error())
					return
				}
				finalDest = renamed
			case ConflictActionOverwrite:
				// finalDest stays; temp-rename logic will handle atomic swap
			}
		}

		// Clean up orphaned temp files from earlier attempts at this destination
		// (and at its pre-rename name) before opening a fresh one.
		sweepStaleTemps(ctx, xrdClient, userToken, destPath, tempSuffix, sweepCache)
		if finalDest != destPath {
			sweepStaleTemps(ctx, xrdClient, userToken, finalDest, tempSuffix, sweepCache)
		}

		// Open the temp file for writing. The temp path is made unique per
		// session (suffix + opaque upload id) so two concurrent uploads to the
		// same destination — or a file literally named "<name><tempSuffix>" —
		// cannot collide on the in-progress file.
		uploadID := newOpaqueID("up_")
		tempPath := finalDest + tempSuffix + "." + strings.TrimPrefix(uploadID, "up_")
		handle, openErr := xrdClient.OpenUploadFile(ctx, userToken, tempPath, true /* overwrite stale temp */)
		if openErr != nil {
			rollback(&f, openErr)
			if common.IsAuthError(openErr) {
				response.Error(c, http.StatusForbidden,
					"You don't have permission to upload to this location. Your access does not include "+
						"write permission for this path — contact your administrator if you believe this is an error.")
				return
			}
			response.Error(c, http.StatusInternalServerError,
				"Failed to open destination for writing: "+openErr.Error())
			return
		}

		now := time.Now()
		sess := &uploadSession{
			ID:             uploadID,
			UserKey:        userKey,
			UserToken:      userToken,
			DestPath:       finalDest,
			TempPath:       tempPath,
			Size:           f.Size,
			ChunkSize:      cfg.XRD.Upload.ChunkSize,
			Overwrite:      action == ConflictActionOverwrite,
			CreatedAt:      now,
			ExpiresAt:      expiresAt,
			batch:          batch,
			handle:         handle,
			hasher:         sha256.New(),
			state:          "uploading",
			lastActivityAt: now,
		}
		batch.addRef()
		uploadStore.add(sess)
		created = append(created, sess)

		results = append(results, UploadFileSession{
			RelPath:  f.RelPath,
			UploadID: sess.ID,
			DestPath: finalDest,
			Status:   "accepted",
			Conflict: conflictLabel,
		})
	}

	// If every file was skipped (e.g. all conflicts with onConflict=fail/skip),
	// no child holds the shared slot; release it so the user is not charged a
	// concurrency slot for a no-op session.
	if len(created) == 0 {
		batch.releaseNow()
	}

	common.GetLogger().Infow("upload session created",
		"sessionId", sessionID, "userKey", userKey,
		"requested", len(req.Files), "accepted", len(created))

	response.Success(c, CreateSessionResponse{
		SessionID: sessionID,
		ChunkSize: cfg.XRD.Upload.ChunkSize,
		Files:     results,
	})
}

// uploadWriteContext returns the context that bounds a single XRootD chunk
// write. It is intentionally detached from the request context: a chunk write
// runs on the session's shared, long-lived XRootD connection, and cancelling it
// mid-protocol (when the client pauses/aborts the chunk request) abandons a
// stream whose response is never read, wedging the connection for every later
// chunk — including the resume. The timeout still bounds a stuck server.
func uploadWriteContext(reqCtx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(reqCtx), 5*time.Minute)
}

// UploadChunk handles PUT /api/v1/xrd/upload/:uploadId/chunk?offset=N.
//
// The request body is the raw bytes of one chunk (no multipart wrapper).
// Chunks must be written in order: the supplied offset must equal the session's
// current BytesReceived. Out-of-order chunks are rejected with 409.
func UploadChunk(c *gin.Context) {
	uploadID := c.Param("uploadId")
	sess, ok := uploadStore.get(uploadID)
	if !ok {
		response.Error(c, http.StatusNotFound, "Unknown uploadId")
		return
	}
	if sess.UserKey != getUserKey(c) {
		response.Error(c, http.StatusForbidden, "Upload session does not belong to this user")
		return
	}

	offsetStr := c.Query("offset")
	offset, err := strconv.ParseInt(offsetStr, 10, 64)
	if err != nil || offset < 0 {
		response.Error(c, http.StatusBadRequest, "Invalid or missing offset")
		return
	}

	cfg := config.GetConfig()
	maxChunk := int64(cfg.XRD.Upload.ChunkSize)
	if c.Request.ContentLength <= 0 {
		response.Error(c, http.StatusBadRequest, "Content-Length is required and must be > 0")
		return
	}
	if c.Request.ContentLength > maxChunk {
		response.Error(c, http.StatusRequestEntityTooLarge,
			fmt.Sprintf("chunk too large: %d > %d", c.Request.ContentLength, maxChunk))
		return
	}

	// Serialize this write against other chunk writes for the same session and
	// against handle teardown (Close). Holding writeMu across the XRootD write
	// guarantees the handle cannot be closed mid-write (no use-after-close) and
	// that two chunks cannot pass the offset check and write concurrently.
	// Lock ordering invariant: writeMu is always acquired BEFORE mu.
	// sess.handle is guarded by writeMu; the rest of the mutable state by mu.
	sess.writeMu.Lock()
	defer sess.writeMu.Unlock()

	sess.mu.Lock()
	if sess.state != "uploading" {
		state := sess.state
		sess.mu.Unlock()
		response.Error(c, http.StatusConflict, "Session is "+state+"; cannot accept more data")
		return
	}
	if offset != sess.BytesReceived {
		received := sess.BytesReceived
		sess.mu.Unlock()
		response.Error(c, http.StatusConflict,
			fmt.Sprintf("offset mismatch: got %d, expected %d", offset, received))
		return
	}
	if offset+c.Request.ContentLength > sess.Size {
		declared := sess.Size
		sess.mu.Unlock()
		response.Error(c, http.StatusBadRequest,
			fmt.Sprintf("chunk would exceed declared size: offset=%d len=%d size=%d",
				offset, c.Request.ContentLength, declared))
		return
	}
	sess.mu.Unlock()

	handle := sess.handle // safe: guarded by writeMu, which we hold
	if handle == nil {
		response.Error(c, http.StatusConflict, "Session is being finalized; cannot accept more data")
		return
	}

	// Read + forward the chunk.
	// We read into a single buffer rather than streaming to XRootD because
	// xrdfs.File.WriteAtContext takes []byte and we need to feed the hasher
	// with the exact same bytes.
	buf := make([]byte, c.Request.ContentLength)
	if _, err := io.ReadFull(c.Request.Body, buf); err != nil {
		// A read error here is almost always client-side cancellation
		// (pause/abort) or a transient network drop. Do NOT fail the session:
		// BytesReceived and the hasher have not advanced, so the session stays
		// consistent and the client can resume from the last committed offset.
		// Truly abandoned sessions are reaped by the janitor (idle timeout).
		response.Error(c, http.StatusBadRequest, "failed to read chunk body: "+err.Error())
		return
	}

	// Write to XRootD at the declared offset.
	//
	// The write context is DETACHED from the request context: once the body is
	// buffered, the write must run to completion on the session's long-lived
	// XRootD connection even if the client goes away. When the user pauses, the
	// browser aborts the in-flight chunk request and c.Request.Context() is
	// cancelled — but cancelling the write mid-protocol abandons an XRootD
	// stream whose (unbuffered) response channel is never drained. The server's
	// reply for that stream then blocks the connection's response reader for
	// good, so the *next* chunk (the resume) hangs forever trying to claim a
	// stream, and the wedged write keeps holding writeMu so even the abort path
	// stalls and the temp file is never cleaned up. Detaching keeps the shared
	// connection consistent; the 5-minute timeout still bounds a genuinely
	// stuck XRootD server. The write is positional and idempotent, so a chunk
	// the client re-sends on resume (because it never saw our ack) is harmless.
	writeCtx, cancel := uploadWriteContext(c.Request.Context())
	defer cancel()
	if err := handle.File().WriteAtContext(writeCtx, buf, offset); err != nil {
		// Transient too: leave the session resumable. WriteAt is positional and
		// idempotent, so re-sending the same offset on retry is safe.
		response.Error(c, http.StatusInternalServerError, "xrootd write failed: "+err.Error())
		return
	}

	sess.mu.Lock()
	// Re-check state in case another goroutine terminated it concurrently.
	if sess.state != "uploading" {
		state := sess.state
		sess.mu.Unlock()
		response.Error(c, http.StatusConflict, "Session is "+state)
		return
	}
	_, _ = sess.hasher.Write(buf)
	sess.BytesReceived += int64(len(buf))
	sess.lastActivityAt = time.Now()
	received := sess.BytesReceived
	sess.mu.Unlock()

	response.Success(c, gin.H{"bytesReceived": received})
}

// CompleteUpload handles POST /api/v1/xrd/upload/:uploadId/complete.
//
// Validates that all declared bytes have arrived, verifies the client-supplied
// SHA-256, atomically renames the temp file onto the destination, and releases
// the concurrency slot.
func CompleteUpload(c *gin.Context) {
	uploadID := c.Param("uploadId")
	sess, ok := uploadStore.get(uploadID)
	if !ok {
		response.Error(c, http.StatusNotFound, "Unknown uploadId")
		return
	}
	if sess.UserKey != getUserKey(c) {
		response.Error(c, http.StatusForbidden, "Upload session does not belong to this user")
		return
	}

	var req CompleteUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest,
			"sha256 is required in the complete request: "+err.Error())
		return
	}
	expectSHA := strings.ToLower(req.Sha256)
	if !isValidSha256Hex(expectSHA) {
		response.Error(c, http.StatusBadRequest, "sha256 must be 64 hex chars")
		return
	}

	sess.mu.Lock()
	if sess.state != "uploading" {
		state := sess.state
		msg := sess.failureMessage
		sess.mu.Unlock()
		response.Error(c, http.StatusConflict,
			fmt.Sprintf("session is %s: %s", state, msg))
		return
	}
	if sess.BytesReceived != sess.Size {
		received, size := sess.BytesReceived, sess.Size
		sess.mu.Unlock()
		response.Error(c, http.StatusBadRequest,
			fmt.Sprintf("not all bytes received: %d of %d", received, size))
		return
	}
	gotSHA := hex.EncodeToString(sess.hasher.Sum(nil))
	if gotSHA != expectSHA {
		sess.mu.Unlock()
		// Abort + cleanup.
		failSession(sess, fmt.Sprintf("sha256 mismatch: got %s expected %s", gotSHA, expectSHA))
		response.Error(c, http.StatusBadRequest,
			fmt.Sprintf("sha256 mismatch: got %s expected %s", gotSHA, expectSHA))
		return
	}
	sess.mu.Unlock()

	ctx := c.Request.Context()
	token := sess.UserToken
	start := time.Now()

	// Close the handle to flush the temp file. Guarded by writeMu so we never
	// close it out from under an in-flight chunk write. Bounded so a stalled
	// XRootD server fails the request instead of hanging it indefinitely (the
	// request context itself carries no deadline).
	sess.writeMu.Lock()
	handle := sess.handle
	sess.handle = nil // prevent double-close
	var closeErr error
	if handle != nil {
		closeCtx, cancelClose := context.WithTimeout(ctx, 30*time.Second)
		closeErr = handle.Close(closeCtx)
		cancelClose()
	}
	sess.writeMu.Unlock()
	if closeErr != nil {
		common.GetLogger().Errorw("failed to close upload handle", "uploadId", uploadID, "error", closeErr)
		failSessionNoHandle(sess, "close failed: "+closeErr.Error())
		_ = common.GetXRDClient().RemoveFile(ctx, token, sess.TempPath)
		response.Error(c, http.StatusInternalServerError, "Failed to finalize upload: "+closeErr.Error())
		return
	}

	xrdClient := common.GetXRDClient()

	// Publish the verified temp file onto the destination. For overwrite we move
	// the existing file aside to a backup path FIRST and delete it only after
	// the new file is safely in place — so a failure mid-publish never destroys
	// the user's original data (it is restored from, or preserved at, the backup).
	var backupPath string
	backedUp := false
	if sess.Overwrite {
		_, exists, statErr := xrdClient.StatPath(ctx, token, sess.DestPath)
		if statErr != nil {
			failSessionNoHandle(sess, "failed to stat existing destination: "+statErr.Error())
			_ = xrdClient.RemoveFile(ctx, token, sess.TempPath)
			response.Error(c, http.StatusInternalServerError,
				"Failed to replace existing file: "+statErr.Error())
			return
		}
		if exists {
			backupPath = sess.DestPath + ".dh-bak." + strings.TrimPrefix(sess.ID, "up_")
			if err := xrdClient.RenameFile(ctx, token, sess.DestPath, backupPath); err != nil {
				failSessionNoHandle(sess, "failed to move existing destination aside: "+err.Error())
				_ = xrdClient.RemoveFile(ctx, token, sess.TempPath)
				response.Error(c, http.StatusInternalServerError,
					"Failed to replace existing file: "+err.Error())
				return
			}
			backedUp = true
		}
	}

	if err := xrdClient.RenameFile(ctx, token, sess.TempPath, sess.DestPath); err != nil {
		// Restore the original so the user never loses their file.
		if backedUp {
			if rerr := xrdClient.RenameFile(ctx, token, backupPath, sess.DestPath); rerr != nil {
				common.GetLogger().Errorw("CRITICAL: failed to restore original after failed publish; original preserved at backup path",
					"uploadId", sess.ID, "dest", sess.DestPath, "backup", backupPath, "error", rerr)
			}
		}
		failSessionNoHandle(sess, "rename failed: "+err.Error())
		_ = xrdClient.RemoveFile(ctx, token, sess.TempPath)
		response.Error(c, http.StatusInternalServerError, "Failed to publish file: "+err.Error())
		return
	}

	// New file is in place; the backup (if any) is now safe to remove.
	if backedUp {
		if err := xrdClient.RemoveFile(ctx, token, backupPath); err != nil {
			common.GetLogger().Warnw("failed to remove backup of overwritten file (orphan left behind)",
				"uploadId", sess.ID, "backup", backupPath, "error", err)
		}
	}

	sess.mu.Lock()
	sess.state = "completed"
	size := sess.Size
	bytes := sess.BytesReceived
	sess.mu.Unlock()

	sess.batch.done()
	uploadStore.delete(sess.ID)

	common.GetLogger().Infow("upload completed",
		"uploadId", sess.ID, "userKey", sess.UserKey,
		"dest", sess.DestPath, "bytes", bytes, "sha256", gotSHA)

	response.Success(c, CompleteUploadResponse{
		UploadID:     sess.ID,
		DestPath:     sess.DestPath,
		Size:         size,
		Sha256:       gotSHA,
		DurationMs:   time.Since(start).Milliseconds(),
		BytesWritten: bytes,
	})
}

// GetUploadStatus handles GET /api/v1/xrd/upload/:uploadId/status. Clients use
// this on reconnect to resume from the last committed offset.
func GetUploadStatus(c *gin.Context) {
	uploadID := c.Param("uploadId")
	sess, ok := uploadStore.get(uploadID)
	if !ok {
		response.Error(c, http.StatusNotFound, "Unknown uploadId")
		return
	}
	if sess.UserKey != getUserKey(c) {
		response.Error(c, http.StatusForbidden, "Upload session does not belong to this user")
		return
	}
	sess.mu.Lock()
	defer sess.mu.Unlock()
	response.Success(c, UploadStatusResponse{
		UploadID:       sess.ID,
		Size:           sess.Size,
		BytesReceived:  sess.BytesReceived,
		ChunkSize:      sess.ChunkSize,
		State:          sess.state,
		FailureMessage: sess.failureMessage,
	})
}

// AbortUpload handles DELETE /api/v1/xrd/upload/:uploadId.
func AbortUpload(c *gin.Context) {
	uploadID := c.Param("uploadId")
	sess, ok := uploadStore.get(uploadID)
	if !ok {
		response.Error(c, http.StatusNotFound, "Unknown uploadId")
		return
	}
	if sess.UserKey != getUserKey(c) {
		response.Error(c, http.StatusForbidden, "Upload session does not belong to this user")
		return
	}
	abortSession(sess, "aborted by client")
	response.Success(c, gin.H{"uploadId": uploadID, "state": "aborted"})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// terminateSession transitions an in-progress session to a terminal state,
// optionally closes the handle (under writeMu, so it never closes mid-write),
// removes the temp file, releases the batch's shared concurrency slot, and
// removes the session from the store. It is a no-op (returns false) if the
// session is no longer "uploading", making it safe to call more than once and
// from racing goroutines.
func terminateSession(sess *uploadSession, newState, reason string, closeHandle bool) bool {
	sess.mu.Lock()
	if sess.state != "uploading" {
		sess.mu.Unlock()
		return false
	}
	sess.state = newState
	sess.failureMessage = reason
	sess.mu.Unlock()

	if closeHandle {
		// Guarded by writeMu: wait for any in-flight chunk write to finish
		// before closing, so the handle is never closed out from under a write.
		sess.writeMu.Lock()
		h := sess.handle
		sess.handle = nil
		sess.writeMu.Unlock()
		if h != nil {
			// Bounded: a dead connection wedges the close (see
			// UploadHandle.Close). The cleanup below must run regardless.
			closeCtx, cancelClose := context.WithTimeout(context.Background(), teardownCloseTimeout)
			if err := h.Close(closeCtx); err != nil {
				common.GetLogger().Warnw("failed to close upload handle during teardown",
					"uploadId", sess.ID, "error", err)
			}
			cancelClose()
		}
	}
	// Fresh context and a fresh XRootD connection: removal must not depend on
	// the (possibly dead) upload connection or a budget spent by the close.
	rmCtx, cancelRm := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelRm()
	if err := common.GetXRDClient().RemoveFile(rmCtx, sess.UserToken, sess.TempPath); err != nil {
		common.GetLogger().Warnw("failed to remove upload temp file (orphan left behind)",
			"uploadId", sess.ID, "temp", sess.TempPath, "error", err)
	}
	sess.batch.done()
	uploadStore.delete(sess.ID)
	return true
}

// abortSession marks the session aborted (handle still open — close it).
// Safe to call multiple times.
func abortSession(sess *uploadSession, reason string) {
	if terminateSession(sess, "aborted", reason, true) {
		common.GetLogger().Infow("upload aborted",
			"uploadId", sess.ID, "userKey", sess.UserKey,
			"dest", sess.DestPath, "reason", reason)
	}
}

// failSession marks the session failed (handle still open — close it).
func failSession(sess *uploadSession, reason string) {
	if terminateSession(sess, "failed", reason, true) {
		common.GetLogger().Warnw("upload failed",
			"uploadId", sess.ID, "userKey", sess.UserKey,
			"dest", sess.DestPath, "reason", reason)
	}
}

// failSessionNoHandle is used in the complete-path where we've already closed
// the handle ourselves. It does not attempt another close.
func failSessionNoHandle(sess *uploadSession, reason string) {
	terminateSession(sess, "failed", reason, false)
}

// staleTempUploadID reports whether entryName (a bare file name) is an upload
// temp file for destBase — i.e. "<destBase><tempSuffix>.<32-hex id>" — and
// returns the embedded upload id (without the "up_" prefix).
func staleTempUploadID(entryName, destBase, tempSuffix string) (string, bool) {
	prefix := destBase + tempSuffix + "."
	if !strings.HasPrefix(entryName, prefix) {
		return "", false
	}
	id := strings.TrimPrefix(entryName, prefix)
	if len(id) != 32 {
		return "", false
	}
	if _, err := hex.DecodeString(id); err != nil {
		return "", false
	}
	return id, true
}

// sweepStaleTemps removes temp files for destPath left behind by upload
// sessions this server no longer tracks. The session store is in-memory, so a
// backend restart orphans every in-flight temp file — the janitor cannot see
// them, and they would otherwise sit on the XRootD server forever. Sweeping at
// session creation makes re-uploading a file self-heal its own litter.
//
// Best-effort: list/remove failures are logged, never fatal. dirCache avoids
// re-listing the same directory for multi-file batches.
func sweepStaleTemps(ctx context.Context, xc *common.XRDClient, token, destPath, tempSuffix string, dirCache map[string][]xrdfs.EntryStat) {
	dir, base := path.Split(destPath)
	dir = path.Clean(dir)
	entries, listed := dirCache[dir]
	if !listed {
		var err error
		entries, err = xc.ListDirectory(ctx, dir, token)
		if err != nil {
			common.GetLogger().Debugw("orphan sweep: cannot list directory", "dir", dir, "error", err)
			entries = nil
		}
		dirCache[dir] = entries
	}
	stale := collectStaleTemps(entries, dir, base, tempSuffix, func(id string) bool {
		_, live := uploadStore.get("up_" + id)
		return live
	})
	for _, p := range stale {
		if err := xc.RemoveFile(ctx, token, p); err != nil {
			common.GetLogger().Warnw("orphan sweep: failed to remove stale temp file",
				"path", p, "error", err)
		} else {
			common.GetLogger().Infow("orphan sweep: removed stale upload temp file", "path", p)
		}
	}
}

// collectStaleTemps returns the full paths of directory entries that are
// orphaned upload temp files for the destination `base` in `dir`: entries that
// match the temp naming scheme and whose embedded upload id is NOT a live
// session (isLive). Pure decision logic, separated from XRootD I/O for tests.
func collectStaleTemps(entries []xrdfs.EntryStat, dir, base, tempSuffix string, isLive func(string) bool) []string {
	var stale []string
	for _, e := range entries {
		id, ok := staleTempUploadID(e.Name(), base, tempSuffix)
		if !ok {
			continue
		}
		if isLive(id) {
			continue // belongs to an active session; the janitor owns it
		}
		stale = append(stale, path.Join(dir, e.Name()))
	}
	return stale
}

// validateRelPath ensures a user-supplied relPath is safe to append to an
// already-validated destDir. It forbids absolute paths, empty segments, ".",
// and ".." segments, as well as control characters.
func validateRelPath(rel string) error {
	if rel == "" {
		return errors.New("relPath must not be empty")
	}
	if strings.HasPrefix(rel, "/") {
		return errors.New("relPath must be relative")
	}
	if strings.ContainsAny(rel, "\x00\r\n") {
		return errors.New("relPath contains invalid characters")
	}
	for _, seg := range strings.Split(rel, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return fmt.Errorf("relPath contains forbidden segment %q", seg)
		}
	}
	return nil
}

// joinXRDPath joins an already-validated destination directory with an
// already-validated relative path using POSIX path semantics.
func joinXRDPath(dir, rel string) string {
	return path.Clean(path.Join(dir, rel))
}

// findAvailableRename tries destPath, destPath (1), destPath (2), ... until it
// finds a name that does not exist. Gives up after 100 tries to avoid
// pathological loops.
func findAvailableRename(ctx context.Context, xc *common.XRDClient, token, destPath string) (string, error) {
	dir, base := path.Split(destPath)
	ext := path.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	for i := 1; i < 100; i++ {
		candidate := fmt.Sprintf("%s%s (%d)%s", dir, stem, i, ext)
		_, exists, err := xc.StatPath(ctx, token, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	return "", errors.New("could not find an available rename candidate")
}

// isValidSha256Hex checks for exactly 64 lower/upper hex characters.
func isValidSha256Hex(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

// newOpaqueID returns a cryptographically random id with the given prefix. The
// random portion is 128 bits encoded as hex (32 chars).
func newOpaqueID(prefix string) string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return prefix + hex.EncodeToString(b[:])
}

// parseDurationOrDefault parses a Go duration string; on error returns def.
func parseDurationOrDefault(s string, def time.Duration) time.Duration {
	if s == "" {
		return def
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return def
	}
	return d
}
