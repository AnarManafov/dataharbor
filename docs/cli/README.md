# DataHarbor CLI

[← Back to Documentation](../README.md)

Design documents for the DataHarbor command-line client: a scriptable tool for
uploading and downloading large data sets from hosts that cannot run a web browser.

> **Status: planning.** Nothing described here is implemented yet. These documents
> record what was known on 2026-11-26 and are extended as the feature is built.
> Where a document describes existing behaviour, it refers to backend v1.2.1.

Work is tracked in [issue #125](https://github.com/AnarManafov/dataharbor/issues/125)
and the `CLI …` milestones.

## Documents

| Document                         | Content                                                              |
| -------------------------------- | -------------------------------------------------------------------- |
| [Plan](./PLAN.md)                | Scope tiers (MVP, Full, Future), milestones, work items, conventions |
| [Decisions](./DECISIONS.md)      | Decision log, open questions, and how well each fact is verified     |
| [Authentication](./AUTH.md)      | Pairing flow, device tokens, server-side grants, revocation          |
| [Transfers](./TRANSFER.md)       | Upload and download protocol changes for terabyte-scale transfers    |
| [Backend](./BACKEND.md)          | Persistence, API contract, version negotiation, hardening            |
| [Client](./CLIENT.md)            | Command tree, configuration, output, distribution, testing           |

## Summary

- **Why.** Users need to move terabytes from data-acquisition and HPC hosts that are
  reachable only over SSH. The web UI is the only client today.
- **Authentication.** The user logs in on the web as usual and generates a short-lived
  pairing code. The CLI exchanges the code for a device token issued by the DataHarbor
  backend. Identity-provider tokens never leave the backend. See [AUTH.md](./AUTH.md).
- **Transport.** The CLI uses the same REST API as the web UI, with plain HTTP byte
  streams for data. No gRPC. See [DECISIONS.md](./DECISIONS.md#d2-rest-not-grpc).
- **Persistence.** The backend gains a PostgreSQL store for sessions, device tokens and
  identity-provider grants. See [BACKEND.md](./BACKEND.md#persistence).
- **Most of the work is in the backend.** The current API accepts only the browser
  session cookie, caps uploads at 50 GiB per file, and cannot resume downloads. The
  client is the smaller half of the project.

## The MVP in one sentence

A user on a headless host can pair the CLI through the web UI and then, from a script,
upload and download files and directory trees of hundreds of gigabytes, with
interrupted transfers resuming where they stopped and integrity verified, against a
single backend instance.

What the MVP deliberately leaves out is listed in [PLAN.md](./PLAN.md#known-mvp-limitations).
