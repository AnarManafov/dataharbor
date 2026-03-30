# AGENTS.md — DataHarbor

> Guidance for AI coding agents working in this repository.

## Project Overview

DataHarbor is a full-stack web application providing secure web access to GSI Lustre cluster data via XROOTD protocol integration. It is a **monorepo** with two main components:

| Component | Directory | Language/Framework | Port        |
| --------- | --------- | ------------------ | ----------- |
| Backend   | `app/`    | Go 1.26+, Gin      | 22000 (dev) |
| Frontend  | `web/`    | Vue 3, Vite 7      | 5173 (dev)  |

The root `package.json` uses **npm workspaces** to orchestrate both components.

## Repository Layout

```text
├── app/                    # Go backend (REST API + XROOTD client)
│   ├── main.go             # Entry point
│   ├── go.mod / go.sum     # Go module
│   ├── config/             # Config structs + YAML loading (Viper)
│   ├── controller/         # HTTP handlers (auth, xrd, health, user)
│   ├── middleware/         # Gin middleware (auth, CORS, recovery, trace, debug)
│   ├── route/              # Route registration
│   ├── common/             # Logger (Zap), XRD client wrapper, sysconf
│   ├── request/            # Request DTOs
│   ├── response/           # Response DTOs & standardized error handling
│   ├── util/               # Helpers (snowflake IDs, etc.)
│   └── test/               # Integration & benchmark tests
├── web/                    # Vue 3 frontend (SPA)
│   ├── src/
│   │   ├── api/            # Axios HTTP client (api.js, request.js)
│   │   ├── components/     # Reusable Vue components
│   │   ├── composables/    # Vue 3 composables (useAuth.js)
│   │   ├── config/         # Runtime config loader
│   │   ├── router/         # Vue Router with auth guards
│   │   ├── services/       # Download streaming (StreamSaver.js)
│   │   ├── store/          # Vuex 4 store (auth state — active)
│   │   ├── stores/         # Pinia store (scaffolded, not primary)
│   │   ├── styles/         # CSS custom properties design system (theme.css)
│   │   ├── utils/          # Utility functions
│   │   └── views/          # Route-level page components
│   ├── public/             # Static assets + runtime config.json
│   ├── vite.config.js      # Vite config with Element Plus auto-import
│   └── cert-config.js      # HTTPS dev cert configuration
├── docker/                 # Docker Compose configs (dev, prod, deploy)
├── docs/                   # Comprehensive documentation
├── packaging/              # RPM spec files & build scripts
├── scripts/                # Build scripts (build-backend.sh)
├── tools/                  # Release tools (changelog, version sync)
├── .github/workflows/      # CI/CD (backend.yml, frontend.yml, release)
├── .devcontainer/          # Dev container config
└── package.json            # Root workspace orchestrator
```

## Quick Reference Commands

The project includes a `Makefile` for all common tasks. Run `make help` to see all available targets.

```bash
# Start full dev environment (frontend + backend concurrently)
make dev

# Start components independently
make dev-frontend             # https://localhost:5173
make dev-backend              # https://localhost:22000

# Build both components
make build

# Build components individually
make build-backend            # Static binary with version injection
make build-frontend           # Vite production build

# Testing
make test                     # All backend tests with coverage
make test-verbose             # Verbose test output
make test-race                # Tests with race detection
make test-coverage-html       # HTML coverage report
make test-integration         # Integration tests
make test-benchmark           # Benchmark tests

# Code quality
make fmt                      # Format Go code
make vet                      # Run go vet
make lint                     # Run golangci-lint

# Dependencies
make deps                     # Install all dependencies
make update                   # Update all dependencies
make tidy                     # Tidy go.mod

# Clean
make clean                    # Clean build artifacts
make clean-all                # Clean everything including node_modules
```

---

## Backend (Go) — `app/`

### Architecture

The backend is a **Gin-based REST API** with a middleware pipeline:

```tet
Request → Recovery → CORS → [Debug] → Trace → [Auth] → Controller → Response
```

- **Controllers** (`controller/`): Handle HTTP requests. Key files:
  - `auth.go` — OIDC authentication (login, callback, logout, user info, session middleware, token refresh)
  - `xrd.go` — XROOTD file operations (list, download, paged listing, hostname, initial dir)
  - `health.go` — Health check endpoint
  - `user.go` — User info endpoint
- **Middleware** (`middleware/`): Modular Gin middleware. Each middleware is in its own file.
- **Routes** (`route/routes.go`): All API route registration in `SetupRouter()`. Also serves the frontend SPA via `NoRoute` fallback.
- **Config** (`config/`): Typed config structs loaded from YAML via Viper. Supports env var overrides with `DATAHARBOR_` prefix.
- **Common** (`common/`): Shared singletons — logger (Zap + Lumberjack), XRD client wrapper.
- **Response** (`response/`): Standardized API response format via `response.Success()`, `response.Error()`, etc.

### Key Patterns & Conventions

1. **Module path**: `github.com/AnarManafov/dataharbor/app`
2. **Config**: Typed Go structs in `config/config.go`. Loaded once at startup, accessed via `config.GetConfig()` singleton. YAML files in `config/application.*.yaml`.
3. **Logging**: Use `common.GetLogger()` which returns a `*zap.SugaredLogger`. Never use `fmt.Print` in production code (only in early startup before logger init).
4. **Error responses**: Always use `response.Error(c, http.StatusXxx, "message")` or `response.Success(c, data)`. Never call `c.JSON()` directly for API responses (except in `response.JSON()` for special cases).
5. **XRD client**: Access via `common.GetXRDClient()` singleton. Creates fresh connections per request (no connection pooling). Auth tokens are passed per-request.
6. **Authentication**: BFF (Backend-For-Frontend) pattern with OIDC. Tokens stored server-side in an in-memory map (not in cookies due to size). Session management via Gorilla sessions with HTTP-only cookies.
7. **Testing**: Co-located `*_test.go` files using `testify` assertions. Each test package has a `main_test.go` for shared setup. Integration/benchmark tests live in `app/test/`.
8. **Build**: Static binary with `CGO_ENABLED=0`. Version info injected via ldflags (`config.Version`, `config.GitCommit`, `config.BuildTime`).
9. **File path security**: All user-provided file paths are validated against directory traversal (`validateFilePath()` in `controller/xrd.go`).

### API Routes

```text
Public:
  GET  /health              — Health check
  GET  /api/health           — Health check (alt path)
  GET  /api/auth/login       — Initiate OIDC login
  GET  /api/auth/callback    — OIDC callback
  POST /api/auth/logout      — Logout
  GET  /api/auth/user        — Current user info

Protected (require session):
  GET  /api/v1/xrd/ls        — List directory
  GET  /api/v1/xrd/initialDir — Initial directory path
  GET  /api/v1/xrd/download  — Stream file download
  GET  /api/v1/xrd/hostname  — XRD server hostname
  POST /api/v1/xrd/ls/paged  — Paginated directory listing
```

### Adding a New Endpoint

1. Create or extend a controller in `controller/` with a handler function: `func HandlerName(c *gin.Context) { ... }`
2. Register the route in `route/routes.go` under the appropriate group (`api` for protected, `auth` for public auth routes)
3. Use `response.Success(c, data)` / `response.Error(c, status, msg)` for responses
4. Add tests in the same package (`controller/handler_test.go`)

### Adding Middleware

1. Create a new file in `middleware/` (e.g., `rate_limit.go`)
2. Implement as `func MiddlewareName() gin.HandlerFunc { ... }`
3. Wire it into the pipeline in `route/routes.go`
4. Add tests in `middleware/rate_limit_test.go`

### Configuration

Config is loaded from YAML files with this precedence (highest first):
1. Environment variables (`DATAHARBOR_*`, using `_` as separator for nested keys)
2. Config file specified via `--config` CLI flag
3. Auto-detected config file (`config/application.yaml`, `application.yaml`, etc.)
4. Default values in `config/config.go`

Key config sections: `server`, `logging`, `xrd`, `auth`, `frontend`. See `config/config.go` for the full typed structure.

---

## Frontend (Vue.js) — `web/`

### Architecture

Vue 3 SPA with Element Plus UI components, built with Vite.

- **Entry**: `src/main.js` → `src/App.vue` (registers Vue Router, Pinia, Vuex, plugins)
- **Routing**: `src/router/index.js` — HTML5 history mode, `beforeEach` guard checks auth for protected routes
- **State**: Vuex 4 (`src/store/index.js`) is the **active** auth state store. Pinia (`src/stores/`) is scaffolded but not primary.
- **API**: `src/api/api.js` (endpoint definitions) + `src/api/request.js` (Axios interceptors). Two Axios instances exist — consolidation is a known TODO.
- **Auth**: `src/composables/useAuth.js` — singleton composable managing auth state. BFF pattern: backend handles OIDC, frontend just redirects.
- **Styling**: CSS custom properties design system in `src/styles/theme.css` using `--dh-*` prefix. Element Plus variables mapped to design tokens. SCSS in SFC `<style>` blocks. No Tailwind.
- **Downloads**: `src/services/downloadService.js` uses StreamSaver.js for large file streaming.

### Key Patterns & Conventions

1. **Component style**: Mix of Options API and Composition API. **Use Composition API for new components.**
2. **Element Plus auto-import**: Components and APIs are auto-imported via `unplugin-vue-components` + `unplugin-auto-import`. No need to explicitly import Element Plus components.
3. **Path alias**: `@` → `src/` (configured in `vite.config.js` and `jsconfig.json`)
4. **Auth flow**: Frontend redirects to `/api/auth/login` → backend handles OIDC → callback sets HTTP-only session cookies → frontend calls `/api/auth/user` to get user info.
5. **Runtime config**: App fetches `/config.json` at startup for runtime settings (API base URL, feature flags). Build-time config via `VITE_*` env vars.
6. **Version injection**: Three git-tag-derived constants injected at build time: `__APP_VERSION__`, `__GLOBAL_VERSION__`, `__BACKEND_VERSION__`.
7. **Dev proxy**: Vite proxies `/api` → `https://localhost:22000` (backend) with cookie rewriting.
8. **CSS tokens**: Use `--dh-*` CSS custom properties from `theme.css`. Don't hardcode colors or sizes.
9. **No linting configured**: No ESLint/Prettier in the frontend. Follow Vue.js Style Guide manually.
10. **No frontend tests**: Test infrastructure is not yet set up. Framework references (vitest) exist in docs but aren't configured.

### Views & Routes

| View                | Route                | Auth Required |
| ------------------- | -------------------- | :-----------: |
| `HomeView`          | `/`                  |      No       |
| `BrowseXrdView`     | `/browse/:path(.*)*` |      Yes      |
| `AboutView`         | `/about`             |      No       |
| `DocumentationView` | `/docs`              |      No       |
| `LoginView`         | `/login`             |      No       |
| `DownloadTestView`  | `/download-test`     |      Yes      |

### Adding a New View

1. Create `src/views/NewView.vue` using Composition API
2. Add route in `src/router/index.js` with `meta: { requiresAuth: true/false }`
3. If it needs API calls, add endpoint functions in `src/api/api.js`
4. If it needs shared state, extend Vuex store in `src/store/index.js`
5. Use Element Plus components (auto-imported) and `--dh-*` CSS tokens for styling

### Adding a Component

1. Create in `src/components/` (or `src/components/partials/` for small focused components)
2. Element Plus components are auto-imported — just use them in templates
3. Use Composition API with `<script setup>` for new components

---

## Cross-Cutting Concerns

### Authentication Flow

The app implements **Backend-For-Frontend (BFF)** OIDC authentication:
- Backend handles all OIDC communication (token exchange, refresh, storage)
- Frontend never sees raw tokens — only HTTP-only session cookies
- Token storage is in-memory on the backend (not suitable for multi-instance deployments without replacing with Redis/DB)

### Commit Convention

Follow [Conventional Commits](https://www.conventionalcommits.org/):
```
type(scope): description
```
Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`
Scopes: `backend`, `frontend`, `docker`, `ci`, `docs`

### Branch Strategy

- `master` — main branch, always deployable
- `feature/*` or `feature/issue-number` — feature branches
- `fix/*` — bug fix branches
- Release tags: `release-vX.Y.Z` triggers automated release pipeline

### CI/CD Pipelines

| Workflow                    | Trigger                                           | Purpose                                                                                                  |
| --------------------------- | ------------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| `backend.yml`               | Changes in `app/**`                               | Go build (amd64/arm64), test, coverage, lint (`golangci-lint`), integration tests, benchmarks, RPM build |
| `frontend.yml`              | Changes in `web/**`                               | npm build, security audit (`npm audit`), RPM build                                                       |
| `version-tag-processor.yml` | `release-v*` / `hotfix-v*` / `prerelease-v*` tags | Update versions, generate changelog, create final tags                                                   |
| `publish-release.yml`       | `v*` tags                                         | Build artifacts, create GitHub release                                                                   |
| `docker-publish.yml`        | Tags                                              | Build & publish Docker images                                                                            |

### Versioning

- Single version source: root `package.json` `version` field
- `tools/sync-versions.js` propagates version to `web/package.json`
- Backend version injected at build time via ldflags
- Frontend version from git tags via Vite `define`

### Docker

Docker Compose configurations in `docker/`:
- `docker-compose.yml` — development
- `docker-compose.prod.yml` — production
- `docker-compose.deploy.yml` — deployment

### Dev Container

Full development environment via `.devcontainer/` with Go, Node.js, Docker CLI, GitHub CLI. Opens ports 5173 (frontend) and 8081 (backend).

---

## Testing Guidelines

### Backend Tests

- **Framework**: `testify` (assertions + mocks)
- **Location**: Co-located `*_test.go` files in each package
- **Shared setup**: `main_test.go` per package for test suite initialization
- **Integration tests**: `app/test/config_integration_test.go` — run with `make test-integration`
- **Benchmarks**: `app/test/config_benchmark_test.go` — run with `make test-benchmark`
- **Coverage target**: 80% overall, 90% for critical paths (auth, file operations)
- **Race detection**: Use `make test-race` before submitting
- **Mandatory tests**: All code changes (new features, bug fixes, refactors) must include corresponding unit tests. Do not submit changes without test coverage for new or modified logic.

### Frontend Tests

- No test infrastructure is currently configured
- Vitest is referenced in docs but not set up in `package.json`
- When adding tests, use Vitest + Vue Test Utils

---

## Important Caveats for Agents

1. **Two Axios instances**: `src/api/api.js` and `src/api/request.js` each create their own Axios instance. The `api.js` instance is the primary one used by most code. This is a known issue (TODO in `request.js`).
2. **Vuex + Pinia coexist**: Both are registered. Vuex is the active store for auth state. Pinia's `counter.js` is a scaffold. Don't mix them for the same concern.
3. **In-memory token store**: `controller/auth.go` stores OIDC tokens in a Go map. Not suitable for horizontal scaling. Known limitation documented in code comments.
4. **Download rate limiting disabled**: The per-user download slot logic in `controller/xrd.go` is temporarily disabled (slot release bug). The code is commented out with TODO.
5. **No frontend linting**: No ESLint or Prettier configured. Be consistent with existing code style.
6. **XROOTD dependency**: The `go-hep.org/x/hep` XROOTD client is the core integration point. XRD operations require a running XROOTD server — tests that touch XRD need the integration test setup.
7. **Config file location**: Backend expects config at `app/config/application.yaml` (or `application.development.yaml` for dev). Missing config causes startup failure.
8. **Static binary**: Backend is built with `CGO_ENABLED=0` for static linking. Don't add dependencies requiring CGO.
9. **SSL in dev**: Both frontend (Vite) and backend (Gin) support HTTPS. Dev certificates are managed via `web/cert-config.js` and `config/application.development.yaml`.
10. **Dependencies from root**: Always run `make deps` (or `npm install`) from the repo root. Running it only in `web/` may cause lock file inconsistencies.
