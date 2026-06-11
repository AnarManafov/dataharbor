# File Upload

DataHarbor supports uploading files from the browser directly into the
XRootD storage system. Uploads are multi-file, chunked, resumable and
verified end-to-end with SHA-256.

## User workflow

1. Navigate to the target directory in the File Browser.
2. Either:
   - click **Upload** in the toolbar to open the system file picker, or
   - drag files (or folders, where the browser supports it) onto the File
     Browser view.
3. Review the confirmation dialog:
   - Per-file size, total size and conflict status are displayed.
   - Conflicting files (already present in the destination) expose a
     per-file **On conflict** selector: `fail`, `skip`, `overwrite`
     (only when the server permits it), `rename`.
   - If any file or the batch exceeds the server-configured limits, the
     **Upload** button is disabled and the violations are listed.
4. Click **Upload**. A progress panel docks to the bottom-right of the
   screen and reports per-file and overall progress. You can pause,
   resume or cancel at any time.

## Server limits

Limits are configured in `application.yaml` under `xrd.upload`:

| Field                  | Default      | Description                                                              |
| ---------------------- | ------------ | ------------------------------------------------------------------------ |
| `enabled`              | `true`       | Master switch for the upload feature.                                    |
| `maxFileSize`          | `50 GiB`     | Upper bound per file.                                                    |
| `maxBatchSize`         | `100 GiB`    | Upper bound for the total bytes of a batch.                              |
| `maxFilesPerBatch`     | `100`        | Maximum number of files in one batch.                                    |
| `maxConcurrentPerUser` | `2`          | Concurrent **upload sessions** per user.                                 |
| `chunkSize`            | `8 MiB`      | Maximum chunk size accepted by the server.                               |
| `sessionTTL`           | `2h`         | Idle sessions are reaped after this period.                              |
| `tempSuffix`           | `.dh-upload` | Suffix for the server-side temp file.                                    |
| `allowOverwrite`       | `false`      | When `false`, the `overwrite` conflict action is rejected with HTTP 403. |
| `checksumAlgo`         | `sha256`     | Checksum algorithm expected from the client.                             |

Clients fetch these values via `GET /api/v1/xrd/upload/limits` and surface
them in the toolbar **Transfer limits** popover.

## Protocol

```
POST /api/v1/xrd/upload/session   -> { uploadId, destPath, conflict, chunkSize } per file
PUT  /api/v1/xrd/upload/{id}/chunk?offset=N   (body: raw bytes)
POST /api/v1/xrd/upload/{id}/complete  (body: { sha256 }) -> server verifies and publishes
DELETE /api/v1/xrd/upload/{id}   -> cancel and clean up temp file
GET  /api/v1/xrd/upload/{id}/status  -> bytesReceived, state
GET  /api/v1/xrd/upload/limits   -> full limits object
```

All endpoints require a valid session cookie. Chunks must be written at
strictly monotonic offsets (the server rejects out-of-order writes with
HTTP 409). The final `complete` request publishes the verified temp file
(`<file>.dh-upload.<id>`, unique per session) onto `<file>` via rename.

For `overwrite`, the existing file is first moved aside to a backup path,
the new file is renamed into place, and only then is the backup removed.
If publishing fails midway the original is restored from the backup, so an
interrupted overwrite never destroys the user's existing data.

A chunk interrupted by a client pause/cancel or a transient network error
does **not** fail the session: the last committed offset is preserved and
the client resumes from there (the upload is genuinely resumable). Sessions
that are abandoned are reclaimed by the idle/lifetime reaper (see below).

## Authentication and XRootD scopes

Upload requires a SciToken carrying the following scopes on the target
path:

- `storage.create` — always required for `CreateSessionRequest`.
- `storage.modify` — required only when `allowOverwrite=true` **and** the
  user selects `overwrite` for a conflicting file.

See [AUTHENTICATION.md](AUTHENTICATION.md) for the full SciTokens setup.

## Integrity

The client computes a streaming SHA-256 of each file in a Web Worker
(hash-wasm, with a pure-JS fallback) **while the chunks are uploading**, and
submits the hex digest with the `complete` request — there is no separate
hashing phase. The server hashes chunks as they arrive and compares its
digest against the client's before renaming the temp file to its final
destination. A mismatch aborts the upload with HTTP 400 and removes the temp
file. A `complete` request without a checksum is rejected with HTTP 400 and
leaves the session resumable.

## Orphaned temp files

The session store is in-memory, so a backend restart orphans the temp file
of any in-flight upload — the janitor no longer knows about it. To self-heal,
session creation sweeps the destination directory for stale
`<file>.dh-upload.<id>` entries whose `<id>` does not belong to a live
session and removes them. Re-uploading a file therefore cleans up the litter
of previous interrupted attempts at the same destination.

## Concurrency and fairness

A per-user `SlotManager` caps the number of concurrent upload **sessions**
a single user can hold (`xrd.upload.max_concurrent_per_user`). One slot is
taken per session and shared by every file in that batch, so a batch may
contain up to `max_files_per_batch` files regardless of the concurrency
cap. Acquisitions beyond the cap are rejected with HTTP 429. Downloads use
the same `SlotManager` mechanism with their own cap
(`xrd.download.max_concurrent_per_user`).

Abandoned sessions are reclaimed by a janitor goroutine: a session is
reaped once it exceeds `xrd.upload.session_ttl` (absolute lifetime) or
`xrd.upload.idle_ttl` (time since the last committed chunk), whichever comes
first. Reaping releases the slot, closes the XRootD handle, and deletes the
temp file.

## Implementation notes

- Backend handlers live in [app/controller/upload.go](../app/controller/upload.go).
- Slot manager is in [app/common/slots.go](../app/common/slots.go).
- Frontend service in [web/src/services/uploadService.js](../web/src/services/uploadService.js).
- Components in [web/src/components/upload/](../web/src/components/upload/).

Refer to GH-56 for the original design discussion.
