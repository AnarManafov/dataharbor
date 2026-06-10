package common

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go-hep.org/x/hep/xrootd"
	"go-hep.org/x/hep/xrootd/xrdfs"
)

// UploadHandle bundles the XRootD client and open file of an in-progress upload.
// The caller is responsible for invoking Close exactly once when the session
// completes, aborts, or is reaped.
//
// Unlike the other XRDClient operations (which use a fresh client per request),
// uploads keep the connection open for the full duration of the session so that
// successive chunk writes do not pay the protocol-handshake cost on every
// request.
type UploadHandle struct {
	client *xrootd.Client
	file   xrdfs.File
	path   string // the target path as opened (typically the ".dh-upload" temp path)
}

// File returns the underlying open XRootD file for WriteAt calls.
func (h *UploadHandle) File() xrdfs.File { return h.file }

// Path returns the path the handle is writing to (usually the temp path).
func (h *UploadHandle) Path() string { return h.path }

// Close closes the file and the underlying client. Safe to call more than once;
// the second call is a no-op.
func (h *UploadHandle) Close(ctx context.Context) error {
	if h == nil {
		return nil
	}
	var firstErr error
	if h.file != nil {
		if err := h.file.Close(ctx); err != nil {
			firstErr = err
		}
		h.file = nil
	}
	if h.client != nil {
		if err := h.client.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		h.client = nil
	}
	return firstErr
}

// OpenUploadFile opens path on the XRootD server for writing. If overwrite is
// false, the call fails when the file already exists. If true, any existing
// file is deleted first.
//
// The returned UploadHandle owns both the XRootD client and the open file; the
// caller must Close it when the upload session ends.
func (xc *XRDClient) OpenUploadFile(ctx context.Context, authToken, path string, overwrite bool) (*UploadHandle, error) {
	xc.logger.Infow("Opening file for upload",
		"path", path, "overwrite", overwrite, "server", xc.address)

	// The handle (and its XRootD connection) outlives this HTTP request: chunk
	// writes and the final complete/close arrive as separate requests over the
	// session's multi-hour lifetime. The xrootd client wires the context passed
	// at creation into its response-reader goroutine, so inheriting the request
	// context would stop response delivery as soon as this request finishes —
	// later chunk acks and the kXR_close reply would then never be received and
	// those calls would block. Detach the client's lifetime from the request.
	clientCtx := context.WithoutCancel(ctx)
	client, err := xc.createClient(clientCtx, authToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	fs := client.FS()
	if fs == nil {
		_ = client.Close()
		return nil, fmt.Errorf("failed to get filesystem interface")
	}

	// OpenModeOwnerRead|OpenModeOwnerWrite = 0600 equivalent. The server-side
	// umask (set in xrootd.cfg) controls the final permissions.
	mode := xrdfs.OpenModeOwnerRead | xrdfs.OpenModeOwnerWrite |
		xrdfs.OpenModeGroupRead
	opts := xrdfs.OpenOptionsOpenUpdate | xrdfs.OpenOptionsMkPath
	if overwrite {
		opts |= xrdfs.OpenOptionsDelete
	} else {
		opts |= xrdfs.OpenOptionsNew
	}

	file, err := fs.Open(ctx, path, mode, opts)
	if err != nil {
		_ = client.Close()
		return nil, classifyXRDError(err, fmt.Sprintf("open %s for write", path))
	}

	return &UploadHandle{client: client, file: file, path: path}, nil
}

// StatPath fetches stat info for a path.
//   - If the path exists, returns (stat, true, nil).
//   - If the path does not exist, returns (zero, false, nil).
//   - On any other error, returns (zero, false, err).
func (xc *XRDClient) StatPath(ctx context.Context, authToken, path string) (xrdfs.EntryStat, bool, error) {
	start := time.Now()
	client, err := xc.createClient(ctx, authToken)
	if err != nil {
		return xrdfs.EntryStat{}, false, fmt.Errorf("failed to create client: %w", err)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			xc.logger.Warn("Error closing client", "error", closeErr)
		}
	}()

	fs := client.FS()
	if fs == nil {
		return xrdfs.EntryStat{}, false, fmt.Errorf("failed to get filesystem interface")
	}

	stat, err := fs.Stat(ctx, path)
	if err != nil {
		if isNotExistError(err) {
			return xrdfs.EntryStat{}, false, nil
		}
		xc.logger.Debugw("Stat failed", "path", path, "duration", time.Since(start), "error", err)
		return xrdfs.EntryStat{}, false, classifyXRDError(err, fmt.Sprintf("stat %s", path))
	}
	return stat, true, nil
}

// RenameFile renames oldpath to newpath on the XRootD server. Used to
// atomically publish a completed upload from its ".dh-upload" temp path onto
// the final destination.
func (xc *XRDClient) RenameFile(ctx context.Context, authToken, oldpath, newpath string) error {
	xc.logger.Infow("Renaming file", "from", oldpath, "to", newpath)
	client, err := xc.createClient(ctx, authToken)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			xc.logger.Warn("Error closing client", "error", closeErr)
		}
	}()

	fs := client.FS()
	if fs == nil {
		return fmt.Errorf("failed to get filesystem interface")
	}

	if err := fs.Rename(ctx, oldpath, newpath); err != nil {
		return classifyXRDError(err, fmt.Sprintf("rename %s to %s", oldpath, newpath))
	}
	return nil
}

// RemoveFile deletes path from the XRootD server. Missing paths are treated as
// success (idempotent cleanup).
func (xc *XRDClient) RemoveFile(ctx context.Context, authToken, path string) error {
	xc.logger.Infow("Removing file", "path", path)
	client, err := xc.createClient(ctx, authToken)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			xc.logger.Warn("Error closing client", "error", closeErr)
		}
	}()

	fs := client.FS()
	if fs == nil {
		return fmt.Errorf("failed to get filesystem interface")
	}

	if err := fs.RemoveFile(ctx, path); err != nil {
		if isNotExistError(err) {
			return nil
		}
		return classifyXRDError(err, fmt.Sprintf("remove %s", path))
	}
	return nil
}

// isNotExistError detects "path does not exist" style errors from XRootD.
// The server does not expose a typed error so we fall back to string matching.
func isNotExistError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "no such file") ||
		strings.Contains(s, "does not exist") ||
		strings.Contains(s, "not found")
}

// classifyXRDError wraps low-level XRootD errors into either an XRootDAuthError
// (so the HTTP layer can respond 403) or a plain annotated error.
func classifyXRDError(err error, ctx string) error {
	if err == nil {
		return nil
	}
	if isAuthorizationError(err) {
		return &XRootDAuthError{
			message: fmt.Sprintf("XRootD denied %s: not authorized", ctx),
			cause:   err,
		}
	}
	return fmt.Errorf("%s: %w", ctx, err)
}

// ErrAlreadyExists indicates an overwrite-disallowed attempt. Controllers can
// use errors.Is to detect this condition.
var ErrAlreadyExists = errors.New("file already exists")
