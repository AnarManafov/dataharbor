# AGENTS.md — DataHarbor

> Guidance for AI coding agents working in this repository.

## Project Overview

Monorepo: Go backend (`app/`) + Vue 3 frontend (`web/`). Root `package.json` uses **npm workspaces**. Both serve over HTTPS in dev.

| Component | Directory   | Port    |
| --------- | ----------- | ------- |
| Backend   | `app/`      | 22000   |
| Frontend  | `web/`      | 5173    |

Dev container: ports 5173 (frontend) and 8081 (backend). Use `forwardPorts` mapping.

---

## Commands

Run everything from the **repo root** unless noted.

```bash
make deps              # Install all dependencies (Go mod download + npm install)
make dev               # Both servers concurrently (via concurrently)
make dev-backend       # Go run app/. Supports CONFIG_FILE_PATH env var
make dev-frontend      # Vite dev server (uses cert-config.js for HTTPS)

make build             # Full build: backend binary + frontend dist/
make build-backend     # Static binary (CGO_ENABLED=0, ldflags inject Version/GitCommit/BuildTime)
                       # Output: bin/dataharbor-backend
make build-frontend    # web/dist/

make test              # All backend tests with coverage report
make test-race         # Race detection (requires CGO_ENABLED=1 internally)
make test-integration  # Tests in app/test/ only
make test-benchmark    # Benchmarks in app/test/

make lint vet fmt      # Code quality (golangci-lint, go vet, go fmt)
make clean-all         # Remove node_modules, build artifacts, coverage files
```

Single-file Go tests: `cd app && go test ./controller -run TestName -v`

Run single test package:
```bash
cd app && go test ./config/... -v
```

---

## Architecture

### Backend — Gin REST API (`app/`)

**Module path**: `github.com/AnarManafov/dataharbor/app`

**Middleware pipeline** (order matters):
```
Recovery → CORS → [Debug if cfg.Server.Debug] → TraceRequest → Routes
```

Protected routes add `SessionAuthMiddleware`.

**Entry point**: `main.go` → initializes logger → loads config → calls `route.SetupRouter()`.

**Config loading** (highest precedence first):
1. `DATAHARBOR_*` env vars (`_` separator for nested keys)
2. `--config` CLI flag
3. Auto-detected YAML: `application.yaml`, `application.development.yaml`, etc.
4. Defaults in `config/config.go`

Singletons: `config.GetConfig()`, `common.GetLogger()` (*zap.SugaredLogger), `common.GetXRDClient()`.

**Response pattern** — never call `c.JSON()` directly for API responses:
```go
response.Success(c, data)
response.Error(c, http.StatusBadRequest, "message")
```

**Writing logs** — use `common.GetLogger().Infof(...)`, not `fmt.Print` (allowed only before logger init during startup).

**Frontend serving**: `route/routes.go:SetupRouter` calls `setupStaticFiles()` which searches `web/dist/` via multiple fallback paths. On non-API unmatched routes, serves `index.html` for SPA routing. API 404s return JSON `{"message": "API endpoint not found"}`.

### Frontend — Vue 3 SPA (`web/`)

**Entry**: `src/main.js` creates app with Pinia + Vuex + Vue Router. Fetches `/config.json` at startup to configure runtime settings before mounting. The `VITE_CONFIG_FILE_PATH` env var overrides the config URL.

**Routing**: `src/router/index.js` — history mode. Uses `meta.isPublic` / `meta.requiresAuth`. OIDC callback routes (`/oidc-callback*`) bypass auth checks.

**State**: Vuex 4 (`src/store/index.js`) is active for auth. Pinia is registered but has only a scaffolded `counter.js` store. Don't introduce Pinia stores for new features without deciding architecture.

**API layer**: Two Axios instances exist (`src/api/api.js` primary, `src/api/request.js` secondary). Consolidation is pending. New calls should go in `api.js`.

**Element Plus**: Auto-imported via `unplugin-vue-components` + `unplugin-auto-import`. No manual imports needed.

**Styling**: CSS custom properties with `--dh-*` prefix in `src/styles/theme.css`. SCSS in SFC `<style>` blocks. No Tailwind/ESLint/Prettier. Uses tabs in Go, spaces elsewhere.

**Path alias**: `@` → `src/`

**Version constants**: `__APP_VERSION__`, `__GLOBAL_VERSION__`, `__BACKEND_VERSION__` injected by Vite `define`. Values come from git tags parsed at build time. Falls back to `package.json` version.

**Sass**: Uses `modern-compiler` API (configured in `vite.config.js`). Dart Sass legacy API warnings will fail if you use old import syntax.

**Dev proxy**: Vite proxies `/api` → `https://localhost:22000` with `secure: false`, cookie domain rewrites, and `SameSite=Lax` conversion via `proxyRes` middleware.

---

## API Routes

All under `/api/v1/xrd/`, except auth. Protected routes listed under "Protected".

### Public
```
GET  /health              Health check
GET  /api/health           Health check (alt)
GET  /api/auth/login       Initiate OIDC login
GET  /api/auth/callback    OIDC callback
POST /api/auth/logout      Logout
GET  /api/auth/user        Current user info
```

### Protected
```
GET  /xrd/ls               List directory
GET  /xrd/initialDir       Initial directory path
GET  /xrd/download         Stream file download
GET  /xrd/hostname         XRD server hostname
GET  /xrd/vstat            Virtual FS stats
GET  /xrd/ping             Ping XRD server
POST /xrd/ls/paged         Paginated directory listing
POST /xrd/download/batch   Multi-file tar/gzip archive download

# Upload endpoints (chunked, resumable, SHA-256 verified)
GET    /xrd/upload/limits                  Transfer limits
POST   /xrd/upload/session                 Create upload session
PUT    /xrd/upload/:uploadId/chunk         Upload chunk
POST   /xrd/upload/:uploadId/complete      Complete upload
DELETE /xrd/upload/:uploadId               Abort upload
GET    /xrd/upload/:uploadId/status        Upload status
```

Upload writes require SciToken scopes presented to XRootD server. See `docs/UPLOAD.md`.

---

## Adding New Endpoints

1. Create handler in `controller/`: `func HandlerName(c *gin.Context)`
2. Register in `route/routes.go` under appropriate group (`auth` = public, `v1` = protected)
3. Use `response.Success()` / `response.Error()` for all responses
4. Add co-located `*_test.go` tests — mandatory, not optional

### Adding Middleware

1. New file in `middleware/`: `func MyMiddleware() gin.HandlerFunc { ... }`
2. Wire into pipeline in `route/routes.go` (order relative to existing middleware matters)
3. Add tests in `middleware/my_middleware_test.go`

---

## Key Quirks & Gotchas

1. **XROOTD fork**: `app/go.mod` uses `replace go-hep.org/x/hep => github.com/AnarManafov/hep`. This is a custom fork with TLS/ZTN support. Import path in code is still `go-hep.org/x/hep` — don't change it. Any XHD protocol changes require updating the fork.

2. **In-memory token store**: `controller/auth.go` stores OIDC tokens in a Go map. Not suitable for multi-instance deployments. Tokens can't fit in cookies due to size. Documented limitation.

3. **Two Axios instances**: `src/api/api.js` (primary) and `src/api/request.js` (secondary). They're independent — interceptors on one don't affect the other.

4. **Vuex + Pinia coexist**: Both registered in `main.js`. Vuex handles real auth state. Pinia is only a scaffold. Don't mix them for the same concern.

5. **No frontend linting or tests**: ESLint/Prettier not configured. Vitest referenced but unconfigured. Be consistent with existing patterns.

6. **CGO disabled for production**: Build uses `CGO_ENABLED=0`. Don't add dependencies requiring CGO. Race tests re-enable it.

7. **SSL everywhere in dev**: Both servers use HTTPS. Dev certs managed via `web/cert-config.js` (frontend) and `app/config/application.development.yaml` (backend). Vite proxy sets `secure: false` to accept self-signed certs.

8. **Always install from root**: `npm install` or `make deps` must run from repo root. Workspaces lock file at root — running only in `web/` causes inconsistencies.

9. **Upload janitor**: `StartUploadJanitor()` fires a periodic goroutine. It's called during `SetupRouter()`, so starting the router initiates the cleanup loop.

10. **Backend version vs frontend version**: Different sources. Backend version comes from git tags injected via ldflags at build. Frontend versions come from Vite's `getVersion()` helper which parses `git describe` output for `v*`, `web/v*`, and `app/v*` tag patterns separately. If no tags exist, falls back to respective `package.json` versions.

11. **File path traversal protection**: `validateFilePath()` in `controller/xrd.go` guards all path inputs. Replicate this pattern for new endpoints accepting path parameters.

12. **Download rate limiting temporarily disabled**: Per-user slot logic in `controller/xrd.go` is commented out (slot release bug). Code present with TODO markers.
