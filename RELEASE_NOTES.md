# Release Notes

## v1.1.0 (2026-06-17)

Changelog:
### Added
- multi-file chunked resumable upload to XRootD (upload) [GH-56] (a46d549)
- add GitHub issue reporting for users (7d3bdc2)
- hash while uploading, sweep stale temps (upload) (1ed6ce5)
- refresh listing when upload completes (upload) (97e2dbc)

### Fixed
- run under v3-capable QEMU on Apple Silicon (xrootd) (a382d0c)
- use browser-native downloads (download) [GH-61] (ba143d0)
- detach XRootD client from request ctx (upload) (af08c3b)
- detach XRootD client from request ctx (upload) (df1bde0)
- detach chunk write from request ctx (upload) (e9b3643)
- land on initial dir after re-login (browse) (5fb650a)
- clarify the rename conflict label (upload) (67a73fd)

### Maintenance
- add opencode-ai setup and streamline AGENTS.md (devcontainer) [GH-56] (2666938)
- Updated coverage badge. (99ce4aa)
- add CLAUDE.md and fix theme font-size (b161670)
- Updated coverage badge. (a0d684a)

### Style
- flat themed surfaces for home, login, about and docs (web) (6eb606e)
- adopt typography tokens and GitHub-density root (web) (770166a)

### Build
- upgrade to XRootD 6.0.3 on Rocky Linux 10 (xrootd) [GH-56] (2b3ae7d)
- ignore the go `app` binary without ignoring app/ dirs (8166835)
- upgrade deps, fix shell-quote vuln (deps) (1d47eb9)

### CI
- push coverage badge via deploy key to satisfy master ruleset (8dad38f)

### Other
- ops(docker): configurable xrootd network alias [GH-63] (113eabf)

