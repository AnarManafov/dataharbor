# CLI: Backend Changes

[← CLI documentation](./README.md)

> **Status: proposed.** Covers backend work that is not specific to authentication
> ([AUTH.md](./AUTH.md)) or the transfer protocol ([TRANSFER.md](./TRANSFER.md)).

## Persistence

### Why

Unattended transfers need server-side state that survives a restart: identity-provider
grants, device tokens, and later upload sessions. Today all of it is in process memory.

### Choice

PostgreSQL. See [DECISIONS.md](./DECISIONS.md#d3-postgresql-for-persistence).

### Relation to issue #51

Issue #51 plans PostgreSQL for usage statistics and an admin panel. The CLI work shares
that infrastructure instead of building a second one:

| Topic                | Issue #51                         | CLI work                                                         |
| -------------------- | --------------------------------- | ---------------------------------------------------------------- |
| Driver               | pgx v5                            | Same                                                             |
| Compose service      | PostgreSQL in dev and deploy      | Same                                                             |
| Schema migrations    | Liquibase compose service         | Open question Q4: the RPM deployment has no compose              |
| Token cache          | Stays in memory                   | **Moves to the database** (decision D3)                          |
| Activity events      | `activity_events` table           | CLI audit records should use the same table, not a parallel one  |
| Session tracking     | `active_sessions` table           | Overlaps the session store; one model should serve both          |

Whichever of the two lands first provides the database package, configuration and
compose service; the other builds on it. Issue #50 (cleanup of abandoned in-memory
tokens) is superseded once sessions live in the database.

### What is stored

| Data                          | Tier | Notes                                                   |
| ----------------------------- | ---- | ------------------------------------------------------- |
| Web sessions and grants       | MVP  | Replaces the in-memory token map                        |
| Pairing codes, CLI devices    | MVP  | See [AUTH.md](./AUTH.md#provisional-data-model)         |
| Audit records                 | Full | Shared with #51's activity events                       |
| Upload sessions               | Full | See [TRANSFER.md](./TRANSFER.md)                        |
| Per-user concurrency counters | Future | Needed only with several backend instances            |

### Rules

- Secrets are encrypted by the backend (AES-GCM) before they are written. The key comes
  from configuration, never from the database.
- Device tokens and pairing codes are stored as hashes only.
- Database access goes through a store interface with an in-memory implementation, so
  unit tests run without PostgreSQL. Integration tests use the compose stack.
- The backend continues to build with `CGO_ENABLED=0`.

## API contract

### OpenAPI specification

There is no machine-readable description of the API, and [API.md](../API.md) has
drifted from the code. The CLI needs a contract that both sides are built against.

- A hand-written OpenAPI 3.1 file in the repository describes the existing endpoints
  first, then each new one before it is implemented.
- The Go client for the CLI is generated from it. Data endpoints (chunk upload,
  download) are hand-written in the client, because generated clients buffer bodies.
- CI fails when generated code is out of date or when routes and specification
  disagree.
- [API.md](../API.md) is regenerated from the specification.

### Errors

Responses use several shapes today, and clients must match on free text. Each error
gains a stable machine-readable code, added without changing fields the web UI reads.

| Code (provisional)       | Meaning                                         | CLI exit code |
| ------------------------ | ----------------------------------------------- | ------------- |
| `auth_required`          | No valid credential                             | 4             |
| `device_reapproval`      | The device's grant must be re-approved          | 4             |
| `forbidden_scope`        | Device scope does not allow the operation       | 6             |
| `not_found`              | Path does not exist                             | 3             |
| `conflict`               | Destination exists or offset mismatch           | 1             |
| `too_many_transfers`     | Concurrency cap reached; retry after a delay    | 5             |
| `checksum_mismatch`      | Integrity check failed                          | 7             |
| `client_too_old`         | Client below the supported minimum              | 9             |

### Capabilities and version skew

A public endpoint (provisionally `GET /api/v1/capabilities`) returns:

- server version and API version;
- minimum and recommended client version;
- feature flags, for example `download.range`, `checksum.adler32`, `cli.pairing`;
- transfer limits (chunk size, size caps, concurrency caps).

The CLI calls it once per invocation, adapts to the features offered, and sends its own
version in the `User-Agent` and a dedicated header. An administrator can raise the
minimum client version in configuration to block old or vulnerable clients.

Policy:

- Changes within `/api/v1` are additive.
- Deprecated endpoints are announced with the `Deprecation` and `Sunset` headers
  (RFC 9745, RFC 8594), which the CLI prints once.
- The supported skew between client and server versions is documented with the first
  release.

## Hardening that the CLI depends on

| Item                                                                                      | Tier        |
| ----------------------------------------------------------------------------------------- | ----------- |
| Request logging redacts credentials (`Authorization`, `Cookie`, tokens)                   | Foundations |
| Debug logging never buffers upload or download bodies                                     | Foundations |
| The server shuts down gracefully on SIGTERM and drains in-flight transfers                | Foundations |
| Audit records for CLI actions: user, device, action, path, bytes, result                  | Full        |

## Running several instances

Not part of the MVP. The pieces that tie the backend to one process today:

- upload sessions (open storage connection and hash state in memory);
- per-user concurrency slots (in-memory counters);
- the upload janitor (would run in every instance);
- the userinfo cache (per instance, harmless).

The database removes the first obstacle for sessions and tokens; the items above are
tracked under Full and Future in [PLAN.md](./PLAN.md).
