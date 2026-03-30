# DataHarbor - Makefile
# Full-stack build automation for Go backend + Vue.js frontend

# Variables
MODULE_PATH=github.com/AnarManafov/dataharbor/app
APP_DIR=app
WEB_DIR=web
BIN_DIR=bin
BACKEND_BINARY=dataharbor-backend

# Version information (can be overridden: make build VERSION=1.0.0)
VERSION ?= $(shell grep -o '"version": *"[^"]*"' package.json | head -1 | sed 's/.*"\([^"]*\)".*/\1/' || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Ldflags for version injection
LDFLAGS_VERSION=-X '$(MODULE_PATH)/config.Version=$(VERSION)' -X '$(MODULE_PATH)/config.GitCommit=$(COMMIT)' -X '$(MODULE_PATH)/config.BuildTime=$(BUILD_TIME)'
LDFLAGS=-s -w $(LDFLAGS_VERSION)

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOCLEAN=$(GOCMD) clean
GOVET=$(GOCMD) vet
GOFMT=$(GOCMD) fmt

# Linting
GOLANGCI_LINT=golangci-lint

# Colors for output
GREEN=\033[0;32m
YELLOW=\033[0;33m
RED=\033[0;31m
NC=\033[0m # No Color

.PHONY: all build build-backend build-frontend \
        test test-verbose test-race test-coverage test-coverage-html test-integration test-benchmark \
        clean clean-all clean-backend clean-frontend \
        deps deps-backend deps-frontend deps-all \
        tidy tidy-backend \
        update update-backend update-frontend update-all \
        fmt vet lint \
        dev dev-backend dev-frontend \
        help

# Default target
all: deps fmt vet lint test build

## help: Show this help message
help:
	@echo "$(GREEN)DataHarbor - Available Commands:$(NC)"
	@echo ""
	@echo "  $(YELLOW)Default:$(NC)"
	@echo "    make all                 - deps, fmt, vet, lint, test, build (default target)"
	@echo ""
	@echo "  $(YELLOW)Development:$(NC)"
	@echo "    make dev                 - Run backend + frontend concurrently"
	@echo "    make dev-backend         - Run backend dev server (port 22000)"
	@echo "    make dev-frontend        - Run frontend dev server (port 5173)"
	@echo ""
	@echo "  $(YELLOW)Dependencies:$(NC)"
	@echo "    make deps                - Download and verify all dependencies"
	@echo "    make deps-backend        - Download and verify Go dependencies"
	@echo "    make deps-frontend       - Install frontend npm dependencies"
	@echo "    make tidy                - Tidy go.mod"
	@echo "    make update              - Update all dependencies to latest"
	@echo "    make update-backend      - Update Go dependencies to latest"
	@echo "    make update-frontend     - Update frontend npm dependencies"
	@echo ""
	@echo "  $(YELLOW)Code Quality:$(NC)"
	@echo "    make fmt                 - Format Go code"
	@echo "    make vet                 - Run go vet"
	@echo "    make lint                - Run golangci-lint"
	@echo ""
	@echo "  $(YELLOW)Testing:$(NC)"
	@echo "    make test                - Run backend tests with coverage report"
	@echo "    make test-verbose        - Run backend tests with verbose output"
	@echo "    make test-race           - Run backend tests with race detection"
	@echo "    make test-coverage       - Run tests and display coverage summary"
	@echo "    make test-coverage-html  - Generate HTML coverage report"
	@echo "    make test-integration    - Run integration tests"
	@echo "    make test-benchmark      - Run benchmark tests"
	@echo ""
	@echo "  $(YELLOW)Build:$(NC)"
	@echo "    make build               - Build both backend and frontend"
	@echo "    make build-backend       - Build backend binary (static, CGO_ENABLED=0)"
	@echo "    make build-frontend      - Build frontend for production"
	@echo ""
	@echo "  $(YELLOW)Clean:$(NC)"
	@echo "    make clean               - Clean build artifacts"
	@echo "    make clean-all           - Clean everything including node_modules"
	@echo "    make clean-backend       - Clean backend build artifacts"
	@echo "    make clean-frontend      - Clean frontend build artifacts"
	@echo ""
	@echo "  $(YELLOW)Version Info:$(NC)"
	@echo "    make version             - Show version information"

## version: Show version information
version:
	@echo "Version:    $(VERSION)"
	@echo "Commit:     $(COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"

# ============================================================================
# Development
# ============================================================================

## dev: Run full dev environment (frontend + backend concurrently)
dev:
	@echo "$(GREEN)Starting dev environment...$(NC)"
	npm run dev

## dev-backend: Run backend dev server
dev-backend:
	@echo "$(GREEN)Starting backend dev server...$(NC)"
	@cd $(APP_DIR) && $(GOCMD) run . $${CONFIG_FILE_PATH:+-config $$CONFIG_FILE_PATH}

## dev-frontend: Run frontend dev server
dev-frontend:
	@echo "$(GREEN)Starting frontend dev server...$(NC)"
	@cd $(WEB_DIR) && npm run dev

# ============================================================================
# Dependencies
# ============================================================================

## deps: Download and verify all dependencies
deps: deps-backend deps-frontend
	@echo "$(GREEN)All dependencies installed!$(NC)"

## deps-backend: Download and verify Go dependencies
deps-backend:
	@echo "$(GREEN)Downloading Go dependencies...$(NC)"
	@cd $(APP_DIR) && $(GOMOD) download && $(GOMOD) verify

## deps-frontend: Install frontend npm dependencies
deps-frontend:
	@echo "$(GREEN)Installing frontend dependencies...$(NC)"
	npm install --workspace=web

## tidy: Tidy go.mod
tidy: tidy-backend

## tidy-backend: Tidy go.mod for backend
tidy-backend:
	@echo "$(GREEN)Tidying go.mod...$(NC)"
	@cd $(APP_DIR) && $(GOMOD) tidy

## update: Update all dependencies to latest versions
update: update-backend update-frontend
	@echo "$(GREEN)All dependencies updated!$(NC)"

## update-backend: Update Go dependencies to latest versions
update-backend:
	@echo "$(GREEN)Updating Go dependencies...$(NC)"
	@cd $(APP_DIR) && $(GOCMD) get -u ./... && $(GOMOD) tidy

## update-frontend: Update frontend npm dependencies
update-frontend:
	@echo "$(GREEN)Updating frontend npm dependencies...$(NC)"
	@cd $(WEB_DIR) && npx npm-check-updates -u && npm install

## update-all: Alias for update
update-all: update

# ============================================================================
# Code Quality
# ============================================================================

## fmt: Format Go code
fmt:
	@echo "$(GREEN)Formatting Go code...$(NC)"
	@cd $(APP_DIR) && $(GOFMT) ./...

## vet: Run go vet
vet:
	@echo "$(GREEN)Running go vet...$(NC)"
	@cd $(APP_DIR) && $(GOVET) ./...

## lint: Run golangci-lint
lint:
	@echo "$(GREEN)Running golangci-lint...$(NC)"
	@cd $(APP_DIR) && $(GOLANGCI_LINT) run --timeout=5m $(LINT_OPTS) || ( \
		echo "$(RED)Linting failed. Install golangci-lint: https://golangci-lint.run/usage/install/$(NC)" && \
		exit 1 \
	)

# ============================================================================
# Testing
# ============================================================================

## test: Run backend tests with coverage report
test:
	@echo "$(GREEN)Running backend tests...$(NC)"
	@cd $(APP_DIR) && $(GOTEST) -coverprofile=coverage.out -covermode=count ./...
	@echo "$(GREEN)Coverage report:$(NC)"
	@cd $(APP_DIR) && $(GOCMD) tool cover -func=coverage.out

## test-verbose: Run backend tests with verbose output
test-verbose:
	@echo "$(GREEN)Running backend tests (verbose)...$(NC)"
	@cd $(APP_DIR) && $(GOTEST) -v ./...

## test-race: Run backend tests with race detection
test-race:
	@echo "$(GREEN)Running backend tests with race detection...$(NC)"
	@cd $(APP_DIR) && CGO_ENABLED=1 $(GOTEST) -race ./...

## test-coverage: Run tests and display coverage summary
test-coverage:
	@echo "$(GREEN)Running backend tests with coverage...$(NC)"
	@cd $(APP_DIR) && $(GOTEST) -coverprofile=coverage.out -covermode=count ./...
	@echo "$(GREEN)Coverage summary:$(NC)"
	@cd $(APP_DIR) && $(GOCMD) tool cover -func=coverage.out | tail -1

## test-coverage-html: Generate HTML coverage report
test-coverage-html: test
	@echo "$(GREEN)Generating HTML coverage report...$(NC)"
	@cd $(APP_DIR) && $(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Open $(APP_DIR)/coverage.html in your browser$(NC)"

## test-integration: Run integration tests
test-integration:
	@echo "$(GREEN)Running integration tests...$(NC)"
	@cd $(APP_DIR)/test && $(GOTEST) -v -timeout=2m

## test-benchmark: Run benchmark tests
test-benchmark:
	@echo "$(GREEN)Running benchmark tests...$(NC)"
	@cd $(APP_DIR)/test && $(GOTEST) -bench=. -benchmem -benchtime=200ms -timeout=2m

# ============================================================================
# Build
# ============================================================================

## build: Build both backend and frontend
build: build-backend build-frontend
	@echo "$(GREEN)Full build complete!$(NC)"

## build-backend: Build backend static binary
build-backend:
	@echo "$(GREEN)Building $(BACKEND_BINARY) $(VERSION)...$(NC)"
	@mkdir -p $(BIN_DIR)
	@cd $(APP_DIR) && CGO_ENABLED=0 $(GOBUILD) -v -trimpath -ldflags="$(LDFLAGS)" -o ../$(BIN_DIR)/$(BACKEND_BINARY) .
	@echo "$(GREEN)Build complete: $(BIN_DIR)/$(BACKEND_BINARY)$(NC)"
	@if command -v file > /dev/null 2>&1; then \
		echo ""; \
		echo "Binary information:"; \
		file $(BIN_DIR)/$(BACKEND_BINARY); \
	fi

## build-frontend: Build frontend for production
build-frontend:
	@echo "$(GREEN)Building frontend...$(NC)"
	@cd $(WEB_DIR) && npm run build
	@echo "$(GREEN)Frontend build complete: $(WEB_DIR)/dist/$(NC)"

# ============================================================================
# Sync Versions
# ============================================================================

## sync-versions: Sync version from root package.json to sub-packages
sync-versions:
	@echo "$(GREEN)Syncing versions...$(NC)"
	node tools/sync-versions.js

## prepare-release: Sync versions and build everything
prepare-release: sync-versions build
	@echo "$(GREEN)Release preparation complete!$(NC)"

# ============================================================================
# Clean
# ============================================================================

## clean: Clean build artifacts
clean: clean-backend clean-frontend
	@echo "$(GREEN)Clean complete!$(NC)"

## clean-backend: Clean backend build artifacts
clean-backend:
	@echo "$(GREEN)Cleaning backend artifacts...$(NC)"
	@cd $(APP_DIR) && $(GOCLEAN)
	@rm -rf $(BIN_DIR)
	@rm -f $(APP_DIR)/coverage.out $(APP_DIR)/coverage.html
	@rm -f $(APP_DIR)/$(BACKEND_BINARY) $(APP_DIR)/app

## clean-frontend: Clean frontend build artifacts
clean-frontend:
	@echo "$(GREEN)Cleaning frontend artifacts...$(NC)"
	@rm -rf $(WEB_DIR)/dist

## clean-all: Clean everything including node_modules
clean-all: clean
	@echo "$(GREEN)Cleaning all generated files...$(NC)"
	@rm -rf $(WEB_DIR)/node_modules
	@rm -rf node_modules
	@echo "$(GREEN)Full clean complete!$(NC)"
