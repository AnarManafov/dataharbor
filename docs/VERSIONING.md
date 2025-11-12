# Backend Versioning

[← Back to Documentation](./README.md)

## Overview

The DataHarbor backend uses **build-time version injection** following Go best practices. Version information is embedded directly into the binary at compile time using `-ldflags`.

## Version Information

Three pieces of information are tracked:

- **Version** - From `package.json` (e.g., `0.14.4`)
- **Git Commit** - Short hash (e.g., `596fe74`)
- **Build Time** - UTC timestamp (e.g., `2025-10-06T11:25:24Z`)

Variables are defined in `app/config/cmd.go`:

```go
var (
    Version   = "dev"       // Overridden at build time
    BuildTime = "unknown"   // Overridden at build time
    GitCommit = "unknown"   // Overridden at build time
)
```

## Building

### Using Build Script (Recommended)

```bash
./scripts/build-backend.sh dataharbor-backend
```

This automatically injects version info and builds with static linking (`CGO_ENABLED=0`).

### Using NPM

```bash
npm run build:backend
```

### Manual Build

```bash
cd app
VERSION=$(grep -o '"version": *"[^"]*"' ../package.json | head -1 | sed 's/.*"\([^"]*\)".*/\1/')
GIT_COMMIT=$(git rev-parse --short HEAD)
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS="-s -w -X github.com/AnarManafov/dataharbor/app/config.Version=$VERSION -X github.com/AnarManafov/dataharbor/app/config.GitCommit=$GIT_COMMIT -X github.com/AnarManafov/dataharbor/app/config.BuildTime=$BUILD_TIME"
CGO_ENABLED=0 go build -ldflags="$LDFLAGS" -o dataharbor-backend .
```

## Displaying Version

```bash
$ dataharbor-backend --version
dataharbor-backend version 0.14.4
Build time: 2025-10-06T11:25:24Z
Git commit: 596fe74
```

## Key Points

- **Static linking** (`CGO_ENABLED=0`) for maximum portability
- **No runtime dependencies** - version is embedded in binary
- **Single source of truth** - version from `package.json` injected at build time
- Development builds without injection show `"dev"` / `"unknown"` values
- Production builds must use build script, RPM packaging, or CI/CD workflows

---

## Related Documentation

- **[Development Guide](./DEVELOPMENT.md)** - Build and release workflow
- **[Deployment Guide](./DEPLOYMENT.md)** - Version deployment
- **[Backend Development](./BACKEND.md)** - Backend build process

---

[← Back to Documentation](./README.md) | [↑ Top](#backend-versioning)
