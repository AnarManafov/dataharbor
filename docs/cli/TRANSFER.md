# CLI: Transfers

[← CLI documentation](./README.md)

> **Status: proposed.** Describes how the upload and download API must change to carry
> terabyte-scale transfers. The existing upload protocol is documented in
> [UPLOAD.md](../UPLOAD.md).

## Current state (backend v1.2.1)

### Upload

| Property              | Today                                                                                   |
| --------------------- | --------------------------------------------------------------------------------------- |
| Protocol              | Create session → `PUT` chunks → complete with SHA-256                                   |
| Chunk order           | Strictly sequential per file; the server keeps a running SHA-256                        |
| Chunk size            | 8 MiB; each chunk is buffered in backend memory                                         |
| Limits                | 50 GiB per file, 100 GiB and 100 files per batch                                        |
| Session lifetime      | 2 hours absolute, shared by the whole batch; 10 minutes idle per file                   |
| Concurrency           | 2 sessions per user; files within a session may upload in parallel                      |
| Storage token         | Captured once at session creation and reused for the final rename                       |
| Lost acknowledgement  | A re-sent chunk is rejected with 409; the client must query the status                  |
| Storage connection    | One per file for the whole session; no reconnect if it drops                            |
| State                 | In backend memory; a restart loses every session                                        |
| Data location         | Written directly to an XRootD temporary file next to the destination                    |

### Download

| Property        | Today                                                                                  |
| --------------- | -------------------------------------------------------------------------------------- |
| Single file     | Streamed from offset 0; `Accept-Ranges` is sent but `Range` is not honoured            |
| Timeout         | Derived from file size; assumes roughly 17 MB/s or better and cuts slower transfers    |
| Integrity       | No checksum, no ETag                                                                   |
| Concurrency     | 2 downloads per user                                                                   |
| Batch           | tar stream, flat names from one directory, 100 files and 10 GiB by default, no resume  |

### Reverse proxy

- The Docker nginx configuration rate-limits `/api/` to 10 requests per second per
  address, which bounds chunked upload at roughly 80 MiB/s with 8 MiB chunks.
- The RPM nginx configuration sets no body-size limit, so nginx's 1 MiB default would
  reject 8 MiB chunks.

## Target behaviour

Work is ordered by what it buys: first allow large transfers, then make them survive
failures, then make single large files faster.

### Layer 1: limits (MVP)

- Raise the per-file, per-batch and file-count limits; make "no limit" expressible.
- Default chunk size 32–64 MiB, advertised by the server and chosen so that one chunk
  completes well inside proxy timeouts.
- Raise per-user concurrency caps. The values are set after the baseline measurement
  (CLI-F1).
- nginx data routes: no request-rate limit, body size above the chunk size, request and
  response buffering off, timeouts suited to large bodies.

### Layer 2: reliability (MVP unless noted)

| Change                                                                                         | Tier |
| ---------------------------------------------------------------------------------------------- | ---- |
| Download honours `Range`, returns a strong `ETag`, and supports `If-Range`                     | MVP  |
| Download timeout replaced by a stall detector (no bytes for N seconds)                         | MVP  |
| Endpoint returning the storage's own checksum (adler32 or crc32c) for a file                   | MVP  |
| Upload session expiry slides with activity; queued files in a batch are not reaped as idle     | MVP  |
| Every storage call in an upload uses a current token, not the one captured at creation         | MVP  |
| Re-sending the last acknowledged chunk succeeds instead of returning 409                       | MVP  |
| Upload reopens the storage connection after a drop                                             | Full |
| Chunks are streamed to storage instead of buffered whole in memory                             | Full |
| Upload sessions are stored in PostgreSQL and survive a backend restart                         | Full |

Portable upload sessions persist the destination, the temporary file path, the byte
offset and the serialized hash state. Any backend instance can then reopen the
temporary file and continue.

### Layer 3: single-file throughput

- **Parallel range downloads (MVP).** With `Range` in place, the client fetches several
  ranges of one file at once. Each range is an independent read on the backend.
  Per-user download slots must count one transfer, not one range request.
- **Parallel chunk upload within a file (Future).** The running SHA-256 forces chunks to
  arrive in order. Lifting that needs a new integrity model: a digest per chunk plus a
  whole-file checksum that can be combined from parts (crc32c or adler32) and compared
  with the storage's own. This is a second protocol revision.

### Beyond the proxy (Future)

Every byte passes through the backend, with TLS on both legs, so backend network and
CPU cap throughput. Two ways past that:

- More backend instances. Requires portable upload sessions, database-backed
  concurrency slots and a lock for the upload janitor.
- Direct-to-storage mode. The backend authorizes the transfer and returns a short-lived
  token scoped to the path and operation; the client talks to XRootD directly and falls
  back to the proxy when storage is unreachable. The API should leave room for a
  transfer plan that returns either backend URLs or storage URLs.

Open question Q2 decides whether either of these moves earlier.

## Integrity

| Step                    | Check                                                                                  |
| ----------------------- | -------------------------------------------------------------------------------------- |
| Upload, per file        | Client SHA-256 compared with the server's running SHA-256 at completion (existing)     |
| Upload, after publish   | Optional: client compares its local checksum with the storage checksum                 |
| Download                | Client computes the checksum while writing and compares with the storage checksum      |
| Download resume         | `If-Range` with the `ETag`; if the file changed, the client restarts from zero         |

SHA-256 cannot be combined across parallel parts, which is why parallel upload needs a
different whole-file checksum. XRootD reports adler32, crc32c and md5 natively.

## Client behaviour

- **Connections.** One HTTP/1.1 connection per stream. Go's HTTP/2 client multiplexes
  all requests to a host over one TCP connection, which removes the benefit of parallel
  streams.
- **Timeouts.** No overall request deadline on data requests; connect, TLS and
  response-header timeouts plus a stall watchdog.
- **Retries.** Per chunk or range, exponential backoff with jitter, honouring
  `Retry-After`. A 429 from the concurrency cap makes the client reduce its parallelism.
- **Resume state.** A journal per transfer under the state directory: session id, chunk
  size, bytes acknowledged, and the source file's size and modification time. On
  resume the server's reported offset is authoritative.
- **Downloads.** Written to `<name>.dh-partial` in the destination directory, verified,
  then renamed.
- **Authentication errors mid-transfer.** The transfer pauses with a distinct exit code
  when the device needs re-approval; re-running the command resumes it.

## Many small files

Per-file overhead dominates: each file needs a session entry, a stat and a temporary
file on storage. The MVP relies on larger batches and file-level parallelism. A
single-request upload for small files and a tar-stream upload are Future items.

## Measurement

Before limits are chosen, CLI-F1 records a baseline: a large file copied with `xrdcp`
directly and through the backend from a representative source host. The same test is
repeated after the MVP transfer work to quantify the proxy overhead.
