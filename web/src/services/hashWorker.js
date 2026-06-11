import { createSHA256 } from 'hash-wasm'
import { sha256 } from '@noble/hashes/sha2.js'
import { bytesToHex } from '@noble/hashes/utils.js'

/**
 * Dedicated worker that computes a streaming SHA-256 over a File without
 * blocking the main thread. Hashing uses hash-wasm (WASM, ~1 GB/s) with a
 * pure-JS @noble fallback (~200 MB/s) if WASM is unavailable.
 *
 * Protocol (messages are matched by the caller-chosen numeric `id`):
 *   in:  { id, file, sliceSize }   start hashing `file` in `sliceSize` slices
 *   in:  { type: 'pause', id }     hold a running job after its current slice
 *   in:  { type: 'resume', id }    let a paused job continue
 *   in:  { type: 'cancel', id }    abandon a job (also wakes a paused one)
 *   out: { id, hashedBytes }       progress, posted after every slice
 *   out: { id, digest }            lowercase hex SHA-256 on completion
 *   out: { id, error }             failure (or 'cancelled')
 *
 * Jobs run sequentially in arrival order, but control messages interleave
 * because the hash loop yields to the event loop on every slice read.
 */

const jobs = new Map() // id -> { paused, cancelled, wake }

let wasmBroken = false

/** New streaming hasher: WASM-backed, falling back to pure JS once WASM
 * proves unavailable in this environment. */
async function newHasher() {
  if (!wasmBroken) {
    try {
      const h = await createSHA256()
      h.init()
      return { update: (b) => h.update(b), digestHex: () => h.digest('hex') }
    } catch {
      wasmBroken = true
    }
  }
  const h = sha256.create()
  return { update: (b) => h.update(b), digestHex: () => bytesToHex(h.digest()) }
}

self.onmessage = async (e) => {
  const msg = e.data || {}
  if (msg.type === 'pause' || msg.type === 'resume' || msg.type === 'cancel') {
    const g = jobs.get(msg.id)
    if (!g) return // job already finished; nothing to control
    if (msg.type === 'pause') {
      g.paused = true
    } else {
      if (msg.type === 'cancel') g.cancelled = true
      else g.paused = false
      g.wake?.()
    }
    return
  }

  const { id, file, sliceSize } = msg
  const g = { paused: false, cancelled: false, wake: null }
  jobs.set(id, g)
  try {
    const hasher = await newHasher()
    let offset = 0
    while (offset < file.size) {
      while (g.paused && !g.cancelled) {
        await new Promise((resolve) => { g.wake = resolve })
        g.wake = null
      }
      if (g.cancelled) {
        self.postMessage({ id, error: 'cancelled' })
        return
      }
      const buf = await file.slice(offset, offset + sliceSize).arrayBuffer()
      hasher.update(new Uint8Array(buf))
      offset += buf.byteLength
      self.postMessage({ id, hashedBytes: Math.min(offset, file.size) })
    }
    self.postMessage({ id, digest: hasher.digestHex() })
  } catch (err) {
    self.postMessage({ id, error: String((err && err.message) || err) })
  } finally {
    jobs.delete(id)
  }
}
