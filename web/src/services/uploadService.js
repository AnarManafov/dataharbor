import { reactive, ref, computed } from 'vue'
import { sha256 } from '@noble/hashes/sha2.js'
import {
  createUploadSession,
  uploadChunk,
  completeUpload,
  abortUpload,
  getUploadStatus,
} from '@/api/api.js'

/**
 * UploadService orchestrates multi-file, chunked, resumable uploads against
 * the DataHarbor backend (see docs/UPLOAD.md).
 *
 * Key responsibilities:
 *   - Expand drag-and-drop DataTransferItems into a flat list of {file,relPath}.
 *   - Compute a streaming SHA-256 per file without loading the whole file
 *     into memory (WebCrypto update is not standard; we feed 8 MiB slices).
 *   - Open a session with the backend and honor the per-file conflict
 *     decisions the user made in the confirmation dialog.
 *   - Drive parallelism: up to `concurrentFiles` files in parallel, sequential
 *     chunks per file.
 *   - Expose reactive progress state to Vue components.
 *   - Support pause/resume/cancel for both individual files and the whole batch.
 */

/** @typedef {'queued'|'hashing'|'uploading'|'verifying'|'done'|'error'|'skipped'|'paused'|'aborted'} FileState */

/**
 * Create an upload service instance. Kept as a factory (not a singleton) so
 * multiple batches can exist concurrently if needed, each with its own state.
 */
export function createUploadService(opts = {}) {
  const concurrentFiles = opts.concurrentFiles ?? 2
  const hashSliceSize = opts.hashSliceSize ?? 8 * 1024 * 1024 // 8 MiB

  // Reactive state the components bind to.
  const state = reactive({
    files: /** @type {Array<UploadItem>} */ ([]),
    overall: {
      totalBytes: 0,
      transferredBytes: 0,
      startedAt: 0,
      speedBps: 0,
      chunkSize: 0,
    },
    status: /** @type {'idle'|'preparing'|'uploading'|'paused'|'completed'|'skipped'|'failed'|'aborted'} */ ('idle'),
    errorMessage: '',
  })

  /** @typedef {{
   *   id: string,
   *   file: File,
   *   relPath: string,
   *   size: number,
   *   sha256: string,
   *   onConflict: 'fail'|'skip'|'overwrite'|'rename',
   *   state: FileState,
   *   uploadId: string,
   *   destPath: string,
   *   conflict: 'none'|'exists'|'',
   *   bytesSent: number,
   *   error: string,
   *   _controller: AbortController|null,
   *   _paused: boolean,
   * }} UploadItem */

  /**
   * Enumerate files from the dialog/drop event. Supports:
   *   - HTMLInputElement.files (flat)
   *   - DataTransferItemList (drag-drop, possibly with folders via
   *     webkitGetAsEntry).
   *
   * The returned items preserve a `relPath` that includes any subdirectory
   * structure, which the server will recreate under destDir.
   *
   * @param {FileList|DataTransferItemList|File[]} source
   * @returns {Promise<Array<{file: File, relPath: string}>>}
   */
  async function enumerate(source) {
    if (!source) return []
    // Case 1: plain FileList / File[].
    if (source instanceof FileList || Array.isArray(source)) {
      return Array.from(source).map((f) => ({ file: f, relPath: f.name }))
    }
    // Case 2: DataTransferItemList.
    const out = []
    const entries = []
    for (const item of source) {
      // item.kind === 'file' required
      if (item.kind !== 'file') continue
      const entry = item.webkitGetAsEntry?.()
      if (entry) {
        entries.push(entry)
      } else {
        // Fallback: no entry API; use the File directly.
        const f = item.getAsFile()
        if (f) out.push({ file: f, relPath: f.name })
      }
    }
    for (const e of entries) {
      await walkEntry(e, '', out)
    }
    return out
  }

  /**
   * Recursively walk a FileSystemEntry, appending files to `out` with their
   * relative path preserved.
   */
  function walkEntry(entry, prefix, out) {
    return new Promise((resolve, reject) => {
      if (entry.isFile) {
        entry.file(
          (f) => {
            out.push({ file: f, relPath: prefix + f.name })
            resolve()
          },
          (err) => reject(err)
        )
      } else if (entry.isDirectory) {
        const reader = entry.createReader()
        const readBatch = () => {
          reader.readEntries(async (children) => {
            if (children.length === 0) {
              resolve()
              return
            }
            try {
              for (const child of children) {
                await walkEntry(child, prefix + entry.name + '/', out)
              }
              // readEntries returns in chunks; call again until empty.
              readBatch()
            } catch (e) {
              reject(e)
            }
          }, reject)
        }
        readBatch()
      } else {
        resolve()
      }
    })
  }

  /**
   * Abort (fire-and-forget) every server-side session from the current batch
   * that is not in a terminal state. Paused or errored files keep their
   * session — and the batch concurrency slot it pins — alive on the server
   * until the idle janitor fires, so dropping them locally without this call
   * leaks slots and leaves ".dh-upload" temp files behind.
   */
  function releaseStaleSessions() {
    for (const f of state.files) {
      if (f.uploadId && !['done', 'skipped', 'aborted'].includes(f.state)) {
        abortUpload(f.uploadId).catch(() => { })
      }
    }
  }

  /**
   * Populate state.files from an enumeration result, in `queued` state, with
   * default conflict resolution `fail`.
   */
  function prepare(items) {
    releaseStaleSessions()
    state.files = items.map((it, idx) => ({
      id: `f_${idx}_${it.file.name}`,
      file: it.file,
      relPath: it.relPath,
      size: it.file.size,
      sha256: '',
      onConflict: 'fail',
      state: 'queued',
      uploadId: '',
      destPath: '',
      conflict: '',
      bytesSent: 0,
      error: '',
      _controller: null,
      _paused: false,
    }))
    state.overall.totalBytes = state.files.reduce((a, f) => a + f.size, 0)
    state.overall.transferredBytes = 0
    state.status = 'idle'
    state.errorMessage = ''
  }

  /**
   * Start the full flow: hash everything, create the session, run uploads,
   * complete them. Returns when the batch reaches a terminal status.
   *
   * @param {string} destDir absolute XRootD directory
   * @param {Record<string,'fail'|'skip'|'overwrite'|'rename'>} conflictByRelPath
   *        Per-file decisions collected from the confirmation dialog.
   */
  async function start(destDir, conflictByRelPath = {}) {
    if (!state.files.length) {
      state.status = 'idle'
      return
    }
    state.status = 'preparing'
    state.errorMessage = ''
    state.overall.startedAt = Date.now()

    // Apply per-file conflict decisions.
    for (const f of state.files) {
      const choice = conflictByRelPath[f.relPath]
      if (choice) f.onConflict = choice
    }

    // --- Hash each file (sequential to keep memory predictable).
    for (const f of state.files) {
      f.state = 'hashing'
      try {
        f.sha256 = await hashFileSha256(f.file, hashSliceSize)
      } catch (err) {
        f.state = 'error'
        f.error = 'hash failed: ' + errMsg(err)
      }
    }
    if (state.files.every((f) => f.state === 'error')) {
      state.status = 'failed'
      state.errorMessage = 'All files failed to hash'
      return
    }

    // --- Open a session with the backend.
    let sessionResp
    try {
      const resp = await createUploadSession({
        destDir,
        files: state.files
          .filter((f) => f.state !== 'error')
          .map((f) => ({
            relPath: f.relPath,
            size: f.size,
            sha256: f.sha256,
            onConflict: f.onConflict,
          })),
      })
      sessionResp = resp.data?.data ?? resp.data
    } catch (err) {
      // A session-creation failure (e.g. 403 when the user's token lacks write
      // permission) is reported once at the batch level, but the per-file rows
      // also need the reason — otherwise they show a bare "error" with no text.
      const msg = errMsg(err)
      state.status = 'failed'
      state.errorMessage = msg
      for (const f of state.files) {
        if (f.state !== 'error') {
          f.state = 'error'
          f.error = msg
        }
      }
      return
    }

    // Match server results back to local items by relPath.
    const byRel = new Map()
    for (const sf of sessionResp.files) byRel.set(sf.relPath, sf)
    for (const f of state.files) {
      if (f.state === 'error') continue
      const sf = byRel.get(f.relPath)
      if (!sf) {
        f.state = 'error'
        f.error = 'server did not return an entry for this file'
        continue
      }
      f.destPath = sf.destPath
      f.conflict = sf.conflict
      if (sf.status === 'skipped') {
        f.state = 'skipped'
        f.error = sf.reason || 'skipped'
        continue
      }
      f.uploadId = sf.uploadId
    }

    // --- Drive parallel uploads.
    state.status = 'uploading'
    const chunkSize = sessionResp.chunkSize || 8 * 1024 * 1024
    state.overall.chunkSize = chunkSize
    const queue = state.files.filter((f) => f.state === 'queued' || f.uploadId)
    queue.forEach((f) => { if (f.state !== 'skipped') f.state = 'queued' })

    await runWithConcurrency(queue, concurrentFiles, (f) =>
      uploadOne(f, chunkSize)
    )

    finalizeBatchStatus()
  }

  /**
   * Compute the terminal batch status once the upload workers have drained.
   * A paused batch must stay `paused` — the workers also exit when the user
   * pauses, and stamping `completed` here would hide the Resume control.
   */
  function finalizeBatchStatus() {
    if (state.status === 'aborted') {
      return // leave as-is
    }
    const anyPaused = state.files.some(
      (f) => f.state === 'paused' || (f.state === 'queued' && f.uploadId)
    )
    if (anyPaused) {
      state.status = 'paused'
      return
    }
    const anyErr = state.files.some((f) => f.state === 'error')
    const anyDone = state.files.some((f) => f.state === 'done')
    const anySkipped = state.files.some((f) => f.state === 'skipped')
    if (anyErr && !anyDone) {
      state.status = 'failed'
    } else if (!anyDone && !anyErr && anySkipped) {
      // Nothing was actually uploaded — every file was skipped (e.g. conflicts
      // with onConflict=fail/skip). Surface this distinctly so the UI does not
      // report a green "success" for a no-op batch.
      state.status = 'skipped'
    } else {
      state.status = 'completed'
    }
  }

  /**
   * Upload one file as a sequence of chunks, resuming from the server-known
   * offset if the item already had an uploadId from a prior session.
   */
  async function uploadOne(f, chunkSize) {
    if (f.state === 'skipped' || f.state === 'error' || f.state === 'done') return

    f.state = 'uploading'

    // Resume support: ask the server where it thinks we are.
    try {
      const st = await getUploadStatus(f.uploadId)
      const data = st.data?.data ?? st.data
      if (typeof data?.bytesReceived === 'number') {
        f.bytesSent = data.bytesReceived
      }
    } catch {
      // Non-fatal: start from zero
    }

    while (f.bytesSent < f.size) {
      if (f._paused) {
        f.state = 'paused'
        return
      }
      if (state.status === 'aborted') {
        f.state = 'aborted'
        return
      }

      const end = Math.min(f.bytesSent + chunkSize, f.size)
      const slice = f.file.slice(f.bytesSent, end)
      const ctrl = new AbortController()
      f._controller = ctrl

      try {
        await uploadChunk(f.uploadId, f.bytesSent, slice, {
          signal: ctrl.signal,
          onUploadProgress: (evt) => {
            const loaded = evt.loaded ?? 0
            // Reflect partial-chunk progress in overall bytes without touching
            // the authoritative f.bytesSent (that advances only after success).
            state.overall.transferredBytes = sumTransferred() + (loaded)
            updateSpeed()
          },
        })
      } catch (err) {
        if (ctrl.signal.aborted) {
          f.state = 'paused'
          return
        }
        f.state = 'error'
        f.error = 'chunk failed: ' + errMsg(err)
        // The client gives up on this file; free its server-side session (and
        // the batch slot it pins) instead of waiting for the idle janitor.
        abortUpload(f.uploadId).catch(() => { })
        return
      } finally {
        f._controller = null
      }

      f.bytesSent = end
      state.overall.transferredBytes = sumTransferred()
      updateSpeed()
    }

    f.state = 'verifying'
    try {
      await completeUpload(f.uploadId)
      f.state = 'done'
    } catch (err) {
      f.state = 'error'
      f.error = 'verify failed: ' + errMsg(err)
      // Most complete-failures are already terminated server-side (the abort
      // then 404s harmlessly), but resumable ones would otherwise pin the slot.
      abortUpload(f.uploadId).catch(() => { })
    }
  }

  /** Pause all in-flight uploads; can be resumed with resumeAll(). */
  function pauseAll() {
    state.status = 'paused'
    for (const f of state.files) {
      f._paused = true
      if (f._controller) f._controller.abort()
    }
    // Drop partial-chunk progress from the display; only committed bytes
    // survive a pause (the server resumes from the last committed offset).
    state.overall.transferredBytes = sumTransferred()
  }

  /** Resume a previously paused batch (reuses existing uploadIds). */
  async function resumeAll() {
    for (const f of state.files) {
      f._paused = false
      if (f.state === 'paused') f.state = 'queued'
    }
    state.status = 'uploading'
    const queue = state.files.filter((f) => f.state === 'queued' && f.uploadId)
    const chunkSize = state.overall.chunkSize || 8 * 1024 * 1024
    await runWithConcurrency(queue, concurrentFiles, (f) =>
      uploadOne(f, chunkSize)
    )
    finalizeBatchStatus()
  }

  /** Abort the whole batch: cancel in-flight chunks and ask the server to
   * clean up temp files. */
  async function abortAll() {
    state.status = 'aborted'
    for (const f of state.files) {
      if (f._controller) f._controller.abort()
      f._controller = null
    }
    await Promise.allSettled(
      state.files
        .filter((f) => f.uploadId && f.state !== 'done')
        .map((f) => abortUpload(f.uploadId).catch(() => { }))
    )
    for (const f of state.files) {
      if (f.state !== 'done' && f.state !== 'skipped') f.state = 'aborted'
    }
  }

  /** Clear the batch from the panel, aborting any sessions still alive on
   * the server so their slots and temp files are released immediately. */
  function reset() {
    releaseStaleSessions()
    state.files = []
    state.status = 'idle'
    state.errorMessage = ''
    state.overall.totalBytes = 0
    state.overall.transferredBytes = 0
    state.overall.speedBps = 0
  }

  function sumTransferred() {
    let total = 0
    for (const f of state.files) total += f.bytesSent
    return total
  }

  function updateSpeed() {
    const dt = (Date.now() - state.overall.startedAt) / 1000
    if (dt > 0) {
      state.overall.speedBps = state.overall.transferredBytes / dt
    }
  }

  return {
    state,
    enumerate,
    prepare,
    start,
    pauseAll,
    resumeAll,
    abortAll,
    reset,
    // Derived getters for convenience in templates.
    progress: computed(() =>
      state.overall.totalBytes > 0
        ? state.overall.transferredBytes / state.overall.totalBytes
        : 0
    ),
    isActive: computed(() => ['preparing', 'uploading'].includes(state.status)),
  }
}

/**
 * Compute a file's SHA-256 incrementally, reading in slices to keep heap
 * usage constant regardless of file size.
 *
 * @param {File|Blob} file
 * @param {number} sliceSize bytes per slice
 * @returns {Promise<string>} lowercase hex SHA-256
 */
async function hashFileSha256(file, sliceSize = 8 * 1024 * 1024) {
  const hasher = sha256.create()
  let offset = 0
  while (offset < file.size) {
    const slice = file.slice(offset, offset + sliceSize)
    const buf = await slice.arrayBuffer()
    hasher.update(new Uint8Array(buf))
    offset += sliceSize
  }
  return bytesToHex(hasher.digest())
}

function bytesToHex(bytes) {
  const hex = []
  for (let i = 0; i < bytes.length; i++) {
    hex.push(bytes[i].toString(16).padStart(2, '0'))
  }
  return hex.join('')
}

/** Run `worker` over items with at most `limit` in flight at a time. */
async function runWithConcurrency(items, limit, worker) {
  let idx = 0
  const runners = Array.from({ length: Math.min(limit, items.length) }, async () => {
    while (idx < items.length) {
      const i = idx++
      await worker(items[i])
    }
  })
  await Promise.all(runners)
}

function errMsg(err) {
  if (!err) return 'unknown error'
  if (typeof err === 'string') return err
  // Prefer the backend's descriptive message (e.g. the permission text from the
  // upload controller) over a generic axios string like "Request failed with
  // status code 403". handleApiError normalizes 4xx into { message, data }, but
  // we also dig into a raw axios error.response just in case.
  const backend =
    err.response?.data?.error || err.response?.data?.message ||
    err.data?.error || err.data?.message
  if (backend) return backend
  if (err.message) return err.message
  try {
    return JSON.stringify(err)
  } catch {
    return String(err)
  }
}

// Expose a single shared instance for convenience. Components that want an
// isolated session can call createUploadService() directly.
export const uploadService = createUploadService()

// Also expose ref so Vue dev tools pick it up.
export const uploadServiceRef = ref(uploadService)
