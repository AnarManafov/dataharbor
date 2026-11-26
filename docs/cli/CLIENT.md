# CLI: Client Design

[← CLI documentation](./README.md)

> **Status: proposed.** The binary name `dataharbor` and the environment prefix
> `DH_` are provisional (open question Q6).

## Target environment

- Shell scripts, cron jobs and batch systems on headless Linux hosts, including older
  enterprise distributions and shared login nodes without root.
- Restricted outbound network, HTTP proxies and sites with private certificate
  authorities.
- macOS and Windows laptops as secondary targets.
- Transfers that run for hours to days.

## Commands

| Command                                   | Purpose                                                  | Tier   |
| ----------------------------------------- | -------------------------------------------------------- | ------ |
| `dataharbor auth login`                   | Redeem a pairing code and store the device token         | MVP    |
| `dataharbor auth status`                  | Show user, server, scope, expiry and grant state         | MVP    |
| `dataharbor auth logout`                  | Revoke this device and delete local credentials          | MVP    |
| `dataharbor ls <remote>`                  | List a directory                                         | MVP    |
| `dataharbor stat <remote>`                | Size, modification time, checksum                        | MVP    |
| `dataharbor get <remote>... <local>`      | Download files or trees (`-r`), resumable                | MVP    |
| `dataharbor put <local>... <remote>`      | Upload files or trees (`-r`), resumable                  | MVP    |
| `dataharbor version`                      | Client version, server version, compatibility            | MVP    |
| `dataharbor completion <shell>`           | Shell completion scripts                                 | MVP    |
| `dataharbor config view / use-context`    | Show effective configuration, switch server profile      | MVP    |
| `dataharbor sync <src> <dst>`             | One-way sync with filters, `--dry-run`, guarded delete   | Full   |
| `dataharbor mkdir`, `dataharbor rm`       | Remote directory and file management                     | Full   |

`-` stands for standard input or output where a file is expected, so
`dataharbor get /data/x.tar - | tar x` works.

## Behaviour that scripts rely on

### Output

- Data goes to standard output; progress, logs and prompts go to standard error.
- `--json` on every command, with a versioned schema that only grows.
- Progress: bars when standard error is a terminal; otherwise a periodic one-line
  summary, or one JSON event per line with `--progress=ndjson`.
- Colour is off when `NO_COLOR` is set, the terminal is dumb, or output is not a
  terminal.

### Exit codes

| Code | Meaning                                         |
| ---- | ----------------------------------------------- |
| 0    | Success                                         |
| 1    | General error                                   |
| 2    | Usage error                                     |
| 3    | Not found                                       |
| 4    | Authentication required or re-approval needed   |
| 5    | Temporary failure; retrying later may succeed   |
| 6    | Permission or scope denied                      |
| 7    | Integrity check failed                          |
| 8    | Partial success; some files failed              |
| 9    | Client version not supported by the server      |
| 130  | Interrupted                                     |

### Non-interactive use

- The CLI prompts only when standard input is a terminal. `--no-input` disables
  prompts entirely and fails with a message naming the flag that was needed.
- Overwrite policy is explicit: the default refuses to overwrite a differing file;
  `--no-clobber` skips, `--force` overwrites.

### Interruption and re-runs

- The first SIGINT or SIGTERM stops cleanly and saves resume state; a second one exits
  immediately. Batch schedulers send SIGTERM before killing a job.
- Re-running the same command continues an interrupted transfer and skips files that
  are already complete.
- A lock per transfer prevents two processes from working on the same destination.

### Safety on download

- Names received from the server are untrusted: absolute paths and `..` are rejected,
  and writes are confined to the destination directory.
- Symbolic links are neither created from server data nor followed at the destination.
- Files are written under a temporary name and renamed after verification.
- Free disk space is checked before a large download.

## Configuration

Precedence: flags, then environment, then the configuration file, then defaults.

| Item             | Location                                                              |
| ---------------- | --------------------------------------------------------------------- |
| Configuration    | `$XDG_CONFIG_HOME/dataharbor/config.yaml`                             |
| Credentials      | `$XDG_CONFIG_HOME/dataharbor/credentials` (`0600`)                    |
| Resume state     | `$XDG_STATE_HOME/dataharbor/transfers/`                               |
| Override         | `DH_CONFIG_DIR` moves all of the above (small home quotas on HPC)     |

- **Contexts.** A context groups a server URL, a credential and defaults.
  `--context` or `DH_CONTEXT` selects one per invocation.
- **Environment variables** use one prefix that differs from the backend's
  `DATAHARBOR_*`, so running the CLI on the server host does not pick up backend
  settings.
- **Tokens for batch jobs** come from `DH_TOKEN_FILE` (preferred) or `DH_TOKEN`.
- **Network.** `HTTPS_PROXY` and `NO_PROXY` are honoured. `--ca-file` or the context
  adds a certificate authority. There is no silent "insecure" mode.

## Implementation

### Repository layout

```
cli/                 separate Go module, tagged cli/vX.Y.Z
  cmd/dataharbor/    main package
  internal/api/      client generated from the OpenAPI specification
  internal/...       transfer engine, config, credentials, output
api/openapi.yaml     shared contract (backend and client)
go.work              local development only; CI also builds with GOWORK=off
```

### Libraries

| Concern          | Choice                                         | Reason                                                              |
| ---------------- | ---------------------------------------------- | ------------------------------------------------------------------- |
| Commands         | cobra                                          | Deep command trees, completions, man-page generation                |
| Configuration    | koanf                                          | Explicit precedence; does not lowercase keys                        |
| Logging          | `log/slog`                                     | Standard library; redaction hooks                                   |
| Progress         | mpb                                            | Several concurrent bars with rate and estimate                      |
| HTTP             | `net/http` with a tuned transport              | Full control of timeouts, connections and buffers                   |
| API client       | Generated (oapi-codegen) for control calls     | One contract for both sides                                         |
| Keyring          | zalando/go-keyring, with file fallback         | Works in static builds; headless hosts usually have no keyring      |

Notes for implementers:

- `http.Client.Timeout` covers the whole response body and must stay unset for data
  requests.
- Without cgo, user lookups read only `/etc/passwd`; use `os.UserHomeDir`, never a
  lookup by name.

### Distribution

| Channel                                                | Tier   |
| ------------------------------------------------------ | ------ |
| Static binaries for Linux, macOS, Windows (amd64, arm64) with checksums | MVP |
| RPM and DEB packages                                   | MVP    |
| Signed checksums and build provenance attestations     | MVP    |
| Man pages and completions in the packages              | MVP    |
| Homebrew, container image, HPC channels (CVMFS, conda, Spack) | Full |

There is no self-update command. The CLI learns the recommended version from the
server's capabilities endpoint and prints a notice.

There is no telemetry.

### Testing

| Level                 | Tool                                                        | Tier |
| --------------------- | ----------------------------------------------------------- | ---- |
| Command behaviour     | Script-driven tests (exit codes, output streams, JSON)      | MVP  |
| HTTP behaviour        | Fake servers with injected errors, deterministic clocks     | MVP  |
| End to end            | CLI against the compose stack in CI                         | MVP  |
| Resume under faults   | Proxy that cuts, throttles and delays connections           | Full |
| Soak                  | Large-file run on demand, with kills and restarts           | Full |

## Documentation to ship

- Command reference (generated) and man pages.
- A scripting guide: exit codes, JSON schemas, environment variables, batch tokens.
- Recipes for cron, systemd timers and batch systems.
- Troubleshooting: proxies, private certificate authorities, expired devices.
