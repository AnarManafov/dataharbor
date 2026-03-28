# DataHarbor

[![CI Backend](https://github.com/AnarManafov/dataharbor/actions/workflows/backend.yml/badge.svg)](https://github.com/AnarManafov/dataharbor/actions/workflows/backend.yml)
[![CI Frontend](https://github.com/AnarManafov/dataharbor/actions/workflows/frontend.yml/badge.svg)](https://github.com/AnarManafov/dataharbor/actions/workflows/frontend.yml)
![Coverage](https://img.shields.io/badge/Coverage-68.8%25-yellow)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![Docker](https://img.shields.io/badge/Docker-Available-2496ED?style=flat&logo=docker)](./docker)
[![Docker Compose](https://img.shields.io/badge/Docker%20Compose-Ready-2496ED?style=flat&logo=docker)](./docker/docker-compose.yml)
[![Devcontainer](https://img.shields.io/badge/Devcontainer-Supported-0078D4?style=flat&logo=visual-studio-code)](./.devcontainer)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![Vue Version](https://img.shields.io/badge/Vue.js-3.0+-4FC08D?style=flat&logo=vue.js)](https://vuejs.org/)


DataHarbor is a high-performance, full-stack web application that provides researchers and developers with a secure, intuitive interface to access and manage GSI Lustre cluster data. Built with a Go backend and Vue.js frontend, it delivers seamless file browsing, directory navigation, and secure file operations through direct XROOTD integration.

## Purpose

DataHarbor empowers users who need to:

- **Browse & Navigate**: Explore GSI Lustre clusters with an intuitive web interface, view metadata, and perform secure file operations
- **High-Performance Streaming**: Download files directly from remote storage with zero temporary storage overhead
- **Secure Access**: Enterprise-grade OIDC authentication with session management
- **Cross-Platform**: Access HPC storage from any device through a modern web browser
- **Observability**: Comprehensive logging and performance metrics for file operations

## Architecture Overview

- **Backend**: Go REST API server with XROOTD client integration
- **Frontend**: Vue 3 SPA with Element Plus UI components
- **Authentication**: OpenID Connect (OIDC) with Backend-For-Frontend (BFF) pattern
- **Security**: HTTP-only cookies, server-side session management
- **Storage**: XROOTD protocol for high-performance data access

## Quick Start

### Using Dev Containers (Recommended for Development)

Zero-configuration development environment with all tools pre-installed:

1. Install [Docker Desktop](https://www.docker.com/products/docker-desktop) + [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
2. Clone and open: `git clone https://github.com/AnarManafov/dataharbor.git && code dataharbor`
3. Click "Reopen in Container" when prompted
4. Run `npm run dev` — access at `https://localhost:5173`

🛠️ **[Dev Container Guide](./.devcontainer/README.md)** — Full setup details, WSL2 instructions, troubleshooting.

### Using Docker Compose

Run the full stack (frontend + backend + XRootD) in containers:

```bash
cd dataharbor/docker
cp .env.example .env && nano .env
docker compose up -d
# Access at https://localhost
```

📦 **[Docker Deployment Guide](./docker/README.md)** — Development & production setup, certificates, troubleshooting.

### Manual Setup

For running services directly on your machine without containers:

#### Prerequisites

- **Go** 1.26+
- **Node.js** 20+

#### Setup

```shell
git clone https://github.com/AnarManafov/dataharbor.git && cd dataharbor

# Install dependencies
cd web && npm install && cd ..
cd app && go mod download && cd ..

# Start development servers
npm run dev  # Or: npm run dev:frontend / npm run dev:backend
```

Access at `https://localhost:5173` (accept the self-signed certificate warning).

## Documentation

> **💡 Complete documentation is available in the [`docs/`](./docs/) folder**

### Getting Started

| Document                                                       | Description                                                 |
| -------------------------------------------------------------- | ----------------------------------------------------------- |
| **[Dev Container Guide](./.devcontainer/README.md)**           | Zero-config development environment (recommended)           |
| **[Setup Guide](./docs/SETUP.md)**                             | Manual environment setup, dependencies, and first steps     |
| **[Development Guide](./docs/DEVELOPMENT.md)**                 | Development workflow, Git conventions, and testing          |
| **[Backend Configuration](./docs/BACKEND_CONFIGURATION.md)**   | Go backend config, environment variables, and YAML settings |
| **[Frontend Configuration](./docs/FRONTEND_CONFIGURATION.md)** | Vue.js frontend config, SSL setup, and deployment settings  |

### Architecture & Design

| Document                                              | Description                                                       |
| ----------------------------------------------------- | ----------------------------------------------------------------- |
| **[System Architecture](./docs/ARCHITECTURE.md)**     | Overall architecture, design patterns, and component interactions |
| **[Authentication System](./docs/AUTHENTICATION.md)** | OIDC integration, BFF pattern, and security model                 |

### Component Development

| Document                                       | Description                                            |
| ---------------------------------------------- | ------------------------------------------------------ |
| **[Backend Development](./docs/BACKEND.md)**   | Go API development, middleware, and XROOTD integration |
| **[Frontend Development](./docs/FRONTEND.md)** | Vue.js development, components, and state management   |

### Technical References

| Document                                   | Description                                       |
| ------------------------------------------ | ------------------------------------------------- |
| **[REST API Reference](./docs/API.md)**    | Complete API endpoints, request/response examples |
| **[XROOTD Integration](./docs/xrootd.md)** | XROOTD client configuration and file operations   |

### Operations & Deployment

| Document                                               | Description                                                   |
| ------------------------------------------------------ | ------------------------------------------------------------- |
| **[Docker Deployment](./docker/README.md)**            | Docker Compose setup, certificates, troubleshooting           |
| **[Deployment Guide](./docs/DEPLOYMENT.md)**           | Production deployment and environment setup                   |
| **[Testing Guide](./docs/TESTING.md)**                 | Testing strategies, coverage requirements, and test execution |
| **[Troubleshooting Guide](./docs/TROUBLESHOOTING.md)** | Comprehensive issue resolution and debugging                  |

### Quick Reference

- **Development**: [Dev Container](./.devcontainer/README.md) (recommended) or [Manual Setup](./docs/SETUP.md)
- **Docker Deployment**: [Docker Guide](./docker/README.md) — run full stack in containers
- **Architecture**: [System Architecture](./docs/ARCHITECTURE.md) → [Authentication](./docs/AUTHENTICATION.md)
- **API Development**: [Backend Guide](./docs/BACKEND.md) → [API Reference](./docs/API.md)
- **UI Development**: [Frontend Guide](./docs/FRONTEND.md)
- **Production**: [Deployment Guide](./docs/DEPLOYMENT.md)

## Common Development Tasks

### Build for Production

```shell
npm run build
```

### Update Dependencies

```shell
# Update all frontend dependencies
npx npm-check-updates --workspaces -u
npm install

# Update backend dependencies
cd app && go get -u ./... && go mod tidy && cd ..
```

See [DEVELOPMENT.md](./docs/DEVELOPMENT.md#dependency-management) for detailed dependency management guidelines.

### Run Tests

```shell
# Backend tests
cd app
go test -v ./...

# Frontend tests (if available)
cd web
npm test
```

### Version Management

```shell
# Sync versions across components
npm run sync-versions

# Prepare release
npm run prepare-release
```

## Project Structure

```text
dataharbor/
├── .devcontainer/          # Dev Container configuration (VS Code)
├── app/                    # Go backend application
│   ├── common/             # Shared utilities (logger, XRootD client)
│   ├── config/             # Configuration management
│   ├── controller/         # HTTP request handlers
│   ├── middleware/         # Authentication, CORS, logging middleware
│   ├── response/           # API response helpers
│   └── route/              # API route definitions
├── web/                    # Vue.js frontend application
│   └── src/
│       ├── api/            # API client and HTTP services
│       ├── components/     # Reusable Vue components
│       ├── router/         # Vue Router configuration
│       ├── store/          # Vuex state management
│       └── views/          # Page-level components
├── docker/                 # Docker Compose deployments
│   ├── backend/            # Backend Dockerfile
│   ├── frontend/           # Frontend Dockerfile
│   ├── nginx/              # Gateway configuration
│   └── xrootd/             # XRootD server container
├── docs/                   # Developer documentation
└── packaging/              # RPM packaging and build scripts
```

## Contributing

This is an internal project for GSI developers. To contribute:

1. Create a feature branch from `master`
2. Follow the coding standards outlined in development docs
3. Add appropriate tests for new functionality
4. Update documentation as needed
5. Submit a pull request with detailed description

## License

See [LICENSE](./LICENSE) file for details.

## Related Technologies

- [XROOTD](https://xrootd.slac.stanford.edu) - High-performance data access system
- [Vue.js](https://vuejs.org/) - Progressive JavaScript framework
- [Gin](https://gin-gonic.com/) - Go web framework
- [Element Plus](https://element-plus.org/) - Vue 3 component library
