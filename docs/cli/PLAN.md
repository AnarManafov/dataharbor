# CLI: Plan

[← CLI documentation](./README.md)

## Scope tiers

| Tier        | Goal                                                                                                   |
| ----------- | ------------------------------------------------------------------------------------------------------ |
| Foundations | Answer the open questions, harden what the CLI depends on, and put the shared contract and database in place |
| MVP         | Scripted, resumable, verified upload and download from a headless host against a single backend instance |
| Full        | Feature-complete first major release: survives backend restarts, sync, audit, fault-tested             |
| Future      | Larger design steps that are deliberately not scheduled yet                                            |

### MVP acceptance

A user on a headless host can pair the CLI through the web UI and then, from a script,
upload and download files and directory trees of hundreds of gigabytes, with
interrupted transfers resuming where they stopped and integrity verified, against a
single backend instance.

### Known MVP limitations

- A backend restart ends in-flight uploads; the affected file restarts from zero.
  Downloads resume.
- One backend instance only.
- Chunks of one file upload sequentially. Parallelism is across files for upload, and
  across files and ranges for download.
- No `sync`, filters, remote `mkdir` or `rm`.
- Device tokens are plain bearer tokens (no key binding).
- Every byte passes through the backend.

### Full adds

- Upload sessions stored in the database, storage reconnect, streamed chunks.
- `sync` with filters and dry run; remote file management.
- Audit records for CLI actions.
- Fault-injection and soak tests; more distribution channels.

### Future (not ticketed)

- DPoP key binding for device tokens.
- Parallel chunk upload within a file, with a new integrity model.
- Direct-to-storage mode with short-lived scoped tokens.
- Several backend instances: database-backed concurrency slots, janitor lock.
- Fast path for many small files (single-request upload, tar-stream upload).
- Bandwidth limiting and transfer scheduling.
- Accepting existing WLCG bearer tokens.
- Path-prefix scopes for devices.

## Milestones

| Milestone                          | Content                                                              |
| ---------------------------------- | -------------------------------------------------------------------- |
| CLI 0: Foundations                 | Spikes, hardening, OpenAPI specification, PostgreSQL plumbing        |
| CLI MVP 1: Auth and persistence    | Session store, token refresh, pairing, device tokens, web page       |
| CLI MVP 2: Transfers and API       | Resumable downloads, checksums, upload fixes, limits, error codes    |
| CLI MVP 3: Client and release      | The client itself, release pipeline, integration tests               |
| CLI Full: Feature complete         | Everything listed under "Full adds"                                  |

MVP 1 and MVP 2 can mostly run in parallel; the one cross-dependency is T5, which
waits for P4. MVP 3 can start with the client scaffold (L1) at any time; its other
tickets wait for the API and authentication work they build on.

## Conventions

- **Ticket ids.** Each work item has a stable id in its title (`CLI-T1: …`). The letter
  names the workstream: F foundations, S stability and security, P persistence,
  A API contract, C CLI authentication, T transfers, L client, R release and testing.
- **Tier.** Carried by the milestone, not by a label.
- **Labels.**

  | Label           | Meaning                                                        |
  | --------------- | -------------------------------------------------------------- |
  | `cli`           | Belongs to the CLI project                                     |
  | `backend`, `frontend`, `docker`, `testing`, `documentation` | Area (existing labels) |
  | `security`      | Security-relevant change                                       |
  | `spike`         | Investigation whose output is a written answer                 |
  | `owner-action`  | Needs the maintainer (infrastructure or identity-provider access) |
  | `needs-details` | Draft; scope and acceptance criteria to be completed before work starts |

- **Ready for implementation.** A ticket is ready when it has no `needs-details` label
  and everything it depends on is closed. Tickets are written so that an AI coding
  agent can implement one in isolation: files to touch, acceptance criteria, required
  tests and verification commands.
- **Manual verification.** Every ticket that changes behaviour states how the
  maintainer can check it by hand, and where practical ships a probe: a script or make
  target that exercises the change against a running stack and prints pass or fail.
  Probes are read-only unless a flag says otherwise.
- **Specifications first.** A change to behaviour described in these documents updates
  the document in the same commit.

## Work items

<!-- TICKETS:BEGIN -->
### CLI 0: Foundations

| ID | Issue | Work item | Depends on | Labels |
| -- | ----- | --------- | ---------- | ------ |
| F1 | [#78](https://github.com/AnarManafov/dataharbor/issues/78) | Baseline throughput: direct XRootD copy versus through the backend | – | `spike`, `owner-action` |
| F2 | [#79](https://github.com/AnarManafov/dataharbor/issues/79) | Confirm identity-provider offline access and token lifetimes | – | `spike`, `owner-action` |
| F3 | [#80](https://github.com/AnarManafov/dataharbor/issues/80) | Spike: XRootD behaviour when a token expires on an open connection | – | `spike`, `backend`, `owner-action` |
| F4 | [#81](https://github.com/AnarManafov/dataharbor/issues/81) | Finalize the CLI specifications after the spikes | F1, F2, F3 | `documentation` |
| S1 | [#82](https://github.com/AnarManafov/dataharbor/issues/82) | Redact sensitive values in request logging | – | `backend`, `security` |
| S2 | [#83](https://github.com/AnarManafov/dataharbor/issues/83) | Debug logging must not buffer transfer bodies | – | `backend` |
| S3 | [#84](https://github.com/AnarManafov/dataharbor/issues/84) | Graceful shutdown that drains in-flight transfers | – | `backend` |
| A1 | [#85](https://github.com/AnarManafov/dataharbor/issues/85) | OpenAPI 3.1 specification for the existing API | – | `backend`, `documentation` |
| P1 | [#86](https://github.com/AnarManafov/dataharbor/issues/86) | PostgreSQL plumbing and store interface | – | `backend`, `docker`, `owner-action` |

### CLI MVP 1: Auth and persistence

| ID | Issue | Work item | Depends on | Labels |
| -- | ----- | --------- | ---------- | ------ |
| P2 | [#87](https://github.com/AnarManafov/dataharbor/issues/87) | Encryption helper for secrets stored in the database | P1 | `backend`, `security`, `needs-details` |
| P3 | [#88](https://github.com/AnarManafov/dataharbor/issues/88) | Store web sessions and identity-provider grants in PostgreSQL | P1, P2 | `backend`, `needs-details` |
| P4 | [#89](https://github.com/AnarManafov/dataharbor/issues/89) | Serialized and background token refresh | P3 | `backend`, `needs-details` |
| C1 | [#90](https://github.com/AnarManafov/dataharbor/issues/90) | Pairing codes and device tokens: data model and endpoints | P1, P2 | `backend`, `security`, `needs-details` |
| C2 | [#91](https://github.com/AnarManafov/dataharbor/issues/91) | Device-token authentication middleware with scopes | C1, P4 | `backend`, `security`, `needs-details` |
| C3 | [#92](https://github.com/AnarManafov/dataharbor/issues/92) | Per-device offline grant obtained at pairing | C1, P4, F2 | `backend`, `security`, `needs-details` |
| C4 | [#93](https://github.com/AnarManafov/dataharbor/issues/93) | Device management endpoints: list, revoke, re-approve | C1, C2 | `backend`, `needs-details` |
| C5 | [#94](https://github.com/AnarManafov/dataharbor/issues/94) | Web UI: "CLI access" page | C1, C4 | `frontend`, `needs-details` |

### CLI MVP 2: Transfers and API

| ID | Issue | Work item | Depends on | Labels |
| -- | ----- | --------- | ---------- | ------ |
| A2 | [#95](https://github.com/AnarManafov/dataharbor/issues/95) | Machine-readable error codes | A1 | `backend`, `needs-details` |
| A3 | [#96](https://github.com/AnarManafov/dataharbor/issues/96) | Capabilities endpoint and minimum client version | A1 | `backend`, `needs-details` |
| T1 | [#97](https://github.com/AnarManafov/dataharbor/issues/97) | Download: honour Range, with ETag and If-Range | – | `backend`, `needs-details` |
| T2 | [#98](https://github.com/AnarManafov/dataharbor/issues/98) | Download: stall detector instead of size-based timeout | – | `backend`, `needs-details` |
| T3 | [#99](https://github.com/AnarManafov/dataharbor/issues/99) | Checksum endpoint backed by the storage checksum | – | `backend`, `needs-details` |
| T4 | [#100](https://github.com/AnarManafov/dataharbor/issues/100) | Upload: sliding session expiry | – | `backend`, `needs-details` |
| T5 | [#101](https://github.com/AnarManafov/dataharbor/issues/101) | Upload: use a current token for every storage call | P4, F3 | `backend`, `needs-details` |
| T6 | [#102](https://github.com/AnarManafov/dataharbor/issues/102) | Upload: accept a re-sent chunk as already done | – | `backend`, `needs-details` |
| T11 | [#103](https://github.com/AnarManafov/dataharbor/issues/103) | Raise transfer limits and defaults | F1 | `backend`, `documentation`, `needs-details` |
| T12 | [#104](https://github.com/AnarManafov/dataharbor/issues/104) | nginx settings for data routes | – | `docker`, `needs-details` |

### CLI MVP 3: Client and release

| ID | Issue | Work item | Depends on | Labels |
| -- | ----- | --------- | ---------- | ------ |
| L1 | [#105](https://github.com/AnarManafov/dataharbor/issues/105) | Client scaffold: module, commands, configuration, CI | – | `needs-details` |
| L2 | [#106](https://github.com/AnarManafov/dataharbor/issues/106) | Client: generated API client and HTTP transport | L1, A1, A3 | `needs-details` |
| L3 | [#107](https://github.com/AnarManafov/dataharbor/issues/107) | Client: auth login, status, logout and credential storage | L2, C2, C4 | `security`, `needs-details` |
| L4 | [#108](https://github.com/AnarManafov/dataharbor/issues/108) | Client: ls and stat | L2, T3 | `needs-details` |
| L5 | [#109](https://github.com/AnarManafov/dataharbor/issues/109) | Client: get (resumable, parallel, verified download) | L2, T1, T2, T3 | `needs-details` |
| L6 | [#110](https://github.com/AnarManafov/dataharbor/issues/110) | Client: put (resumable, verified upload) | L2, T4, T5, T6 | `needs-details` |
| L8 | [#111](https://github.com/AnarManafov/dataharbor/issues/111) | Client: output, exit codes and signal handling | L1, A2 | `needs-details` |
| L9 | [#112](https://github.com/AnarManafov/dataharbor/issues/112) | Client: completions, man pages and user documentation | L3, L5, L6 | `documentation`, `needs-details` |
| R1 | [#113](https://github.com/AnarManafov/dataharbor/issues/113) | Release pipeline for the client | L1 | `chore`, `needs-details` |
| R2 | [#114](https://github.com/AnarManafov/dataharbor/issues/114) | Integration tests: client against the compose stack | L3, L5, L6 | `testing`, `needs-details` |

### CLI Full: Feature complete

| ID | Issue | Work item | Depends on | Labels |
| -- | ----- | --------- | ---------- | ------ |
| S4 | [#115](https://github.com/AnarManafov/dataharbor/issues/115) | Audit records for CLI actions | P1, C2 | `backend`, `security`, `needs-details` |
| A4 | [#116](https://github.com/AnarManafov/dataharbor/issues/116) | Regenerate docs/API.md from the specification | A1 | `documentation`, `needs-details` |
| T7 | [#117](https://github.com/AnarManafov/dataharbor/issues/117) | Upload: reopen the storage connection after a drop | – | `backend`, `needs-details` |
| T8 | [#118](https://github.com/AnarManafov/dataharbor/issues/118) | Upload: stream chunks instead of buffering them | – | `backend`, `needs-details` |
| T9 | [#119](https://github.com/AnarManafov/dataharbor/issues/119) | Upload sessions stored in PostgreSQL and resumable after restart | P1, T7 | `backend`, `needs-details` |
| T10 | [#120](https://github.com/AnarManafov/dataharbor/issues/120) | File-management API: recursive listing, mkdir, delete | A1, C2 | `backend`, `needs-details` |
| L7 | [#121](https://github.com/AnarManafov/dataharbor/issues/121) | Client: sync, mkdir and rm | L5, L6, T10 | `needs-details` |
| R3 | [#122](https://github.com/AnarManafov/dataharbor/issues/122) | Fault-injection tests for resume | R2, T9 | `testing`, `needs-details` |
| R4 | [#123](https://github.com/AnarManafov/dataharbor/issues/123) | Large-file soak test | R2, F1 | `testing`, `needs-details` |
| R5 | [#124](https://github.com/AnarManafov/dataharbor/issues/124) | Additional distribution channels | R1 | `chore`, `needs-details` |
<!-- TICKETS:END -->
