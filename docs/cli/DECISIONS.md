# CLI: Decisions and Open Questions

[← CLI documentation](./README.md)

## Decision log

### D1. Web-first pairing with a backend-issued device token

**Decided 2026-11-26.**

The user generates a short-lived pairing code in the authenticated web UI. The CLI
exchanges it for an opaque device token issued and verified by the DataHarbor backend.
The identity-provider refresh token for that device stays in the backend.

Reasons:

- Every XRootD operation is performed with the user's own identity-provider access
  token, so the backend must hold a valid one for the user at all times. Any design in
  which only the CLI holds credentials does not remove that requirement.
- Starting the flow from an authenticated session ("authenticate, then initiate") is
  the mitigation that cross-device security guidance recommends against the phishing
  weakness of the OAuth device grant.
- The backend controls lifetime, scope, revocation and audit per device.
- When the identity-provider grant expires, the user re-approves on the web. No access
  to the headless host is needed.

Alternatives considered:

| Alternative                                                        | Why not chosen                                                                                             |
| ------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------- |
| OAuth device grant run by the CLI against the identity provider    | Device-code phishing exposure; no backend-side revocation; grant lifetime fixed by the identity provider   |
| Custom per-request tokens signed with a client key pair            | Reinvents DPoP (RFC 9449); a home-grown format has to get replay, URL binding and algorithm pinning right   |
| Short-lived access token plus rotating refresh token for the CLI   | Rotation breaks when several processes share one credential file (overlapping cron jobs, NFS home)         |

### D2. REST, not gRPC

**Decided 2026-11-26.**

One HTTP API serves both the web UI and the CLI: JSON for control, plain HTTP byte
streams for data (`Range` downloads, offset-addressed chunk uploads). The API is
described by an OpenAPI specification from which the Go client is generated.

Reasons:

- The bulk-transfer tools surveyed (AWS CLI, `rclone`, `azcopy`, `gcloud storage`,
  XRootD's own HTTP interface) use HTTP on the WAN data path.
- nginx applies `client_max_body_size` to a whole gRPC stream and has 60-second read
  timeouts between messages; long client streams need special handling.
- Browsers cannot do gRPC client streaming, so the web UI would stay on REST and two
  API surfaces would have to be maintained.
- No like-for-like WAN benchmark of gRPC against HTTP/1.1 was found. The decision rests
  on operational fit, not on a measured throughput difference.

### D3. PostgreSQL for persistence

**Decided 2026-11-26.**

Sessions, device tokens, identity-provider grants and (later) upload sessions are
stored in PostgreSQL. SQLite was rejected because several backend instances cannot
share it and it offers no cross-instance lock for token refresh.

This **reverses one decision recorded in issue #51**, which keeps the token cache in
memory. Unattended multi-day transfers require tokens to survive a backend restart.
The PostgreSQL infrastructure itself (pgx v5, compose service) follows #51.

### D4. Key binding (DPoP) is a second phase

**Decided 2026-11-26.**

The first release uses a plain bearer device token. Binding the token to a client key
pair with DPoP (RFC 9449) is planned as an addition that does not change the pairing
flow. A key in a `0600` file is as exposed as a token in a `0600` file; the binding
protects against leaks in transit, logs and proxies, and becomes strong only when the
key is held in `ssh-agent` or hardware.

### D5. The CLI is a separate Go module

**Decided 2026-11-26.**

The client lives in `cli/` with its own `go.mod` and `cli/v*` release tags. It does
not import the backend module: the backend's `replace` directive for the XRootD fork
would make `go install` of the client impossible, and the backend's dependencies would
bloat the binary. The two sides share the OpenAPI specification, not Go packages.

### D6. Reliability before throughput

**Decided 2026-11-26.**

Raising limits alone is not enough for terabyte transfers. The order of work is:
configuration limits, then resumability and correctness, then single-file
parallelism. See [TRANSFER.md](./TRANSFER.md).

## Open questions

| #  | Question                                                                                                              | Blocks                          | Owner       |
| -- | --------------------------------------------------------------------------------------------------------------------- | ------------------------------- | ----------- |
| Q1 | Does the Keycloak client allow `offline_access`? What are the access-token and offline-session lifetimes?             | Per-device grants (CLI-C3)      | Maintainer  |
| Q2 | What throughput and data volume are expected, and can source hosts reach XRootD directly?                             | Whether direct-to-storage moves into the MVP | Maintainer |
| Q3 | Does XRootD re-check token expiry on an already-open connection?                                                      | Long downloads, upload connection handling | Spike CLI-F3 |
| Q4 | Schema migrations: Liquibase as a compose service (as in #51) or migrations embedded in the backend binary?           | PostgreSQL plumbing (CLI-P1)    | Maintainer  |
| Q5 | Which scope vocabulary does the storage accept (`read:/` and `write:/`, or `storage.read` and friends)? Existing docs disagree. | Read-only device scope  | Maintainer  |
| Q6 | Binary name. `dh` collides with Debian's debhelper command; these documents use `dataharbor` provisionally.           | Client scaffold (CLI-L1)        | Maintainer  |
| Q7 | Default and maximum device-token lifetime.                                                                            | Pairing endpoints (CLI-C1)      | Maintainer  |
| Q9 | Pairing-code format: a long paste-only code (128 bits), or a short typeable code protected by rate limiting and a short lifetime? | Pairing endpoints (CLI-C1) | Settled in CLI-F4 |
| Q8 | Does the XRootD client fork expose the checksum query, or does it need a change?                                      | Checksum endpoint (CLI-T3)      | Spike in CLI-T3 |

## Confidence in the facts used

| Fact                                                                                   | Basis                                              |
| -------------------------------------------------------------------------------------- | -------------------------------------------------- |
| Backend accepts only the session cookie; tokens are held in an in-memory map           | Read in code (v1.2.1)                              |
| Download never reads `Range`; upload token is captured once per session                | Read in code, re-checked                           |
| Upload limits and chunk ordering                                                       | Read in code                                       |
| RFC 8628, 9449, 9700, 10027 content                                                    | Read in the RFCs                                   |
| `gh`, `glab`, AWS CLI, `kubelogin` login behaviour                                     | Read in official documentation                     |
| Keycloak offline-session defaults (30 days idle)                                       | Vendor documentation, not the actual realm         |
| FTS and Rucio token handling                                                           | Conference slides only                             |
| DPoP support in identity providers other than Keycloak                                 | Not found; unverified                              |
| nginx and Go HTTP/2 flow-control limits                                                | Read in documentation and source                   |
