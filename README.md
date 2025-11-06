# DataHarbor

[![CI Backend](https://github.com/AnarManafov/dataharbor/actions/workflows/backend.yml/badge.svg)](https://github.com/AnarManafov/dataharbor/actions/workflows/backend.yml)
[![CI Frontend](https://github.com/AnarManafov/dataharbor/actions/workflows/frontend.yml/badge.svg)](https://github.com/AnarManafov/dataharbor/actions/workflows/frontend.yml)
![Coverage](https://img.shields.io/badge/Coverage-19.9%25-red)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![Docker](https://img.shields.io/badge/Docker-Available-2496ED?style=flat&logo=docker)](./docker)
[![Docker Compose](https://img.shields.io/badge/Docker%20Compose-Ready-2496ED?style=flat&logo=docker)](./docker/docker-compose.yml)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![Vue Version](https://img.shields.io/badge/Vue.js-3.0+-4FC08D?style=flat&logo=vue.js)](https://vuejs.org/)

DataHarbor is a high-performance, full-stack web application that provides researchers and developers with a secure, intuitive interface to access and manage GSI Lustre cluster data. Built with a Go backend and Vue.js frontend, it delivers seamless file browsing, directory navigation, and secure file operations through direct XROOTD integration.

## Purpose

DataHarbor empowers users who need to:

- **Browse & Navigate**: Explore remote file systems on GSI Lustre clusters with an intuitive web interface
- **File Operations**: View detailed metadata (size, permissions, timestamps) and perform secure file operations
- **High-Performance Downloads**: Stream individual files directly from remote storage with zero temporary storage
- **Secure Access**: Leverage enterprise-grade authentication with OIDC integration and session management
- **Large-Scale Data Management**: Handle file operations for massive datasets in high-performance computing environments
- **Cross-Platform Access**: Access HPC storage systems from any device through a modern web browser
- **Real-Time Monitoring**: Track file operations with comprehensive logging and performance metrics

## Architecture Overview

- **Backend**: Go REST API server with XROOTD client integration
- **Frontend**: Vue 3 SPA with Element Plus UI components  
- **Authentication**: OpenID Connect (OIDC) with Backend-For-Frontend (BFF) pattern
- **Security**: HTTP-only cookies, server-side session management
- **Storage**: XROOTD protocol for high-performance data access

## Quick Start

### Using Docker (Recommended)

The fastest way to get started with DataHarbor is using Docker Compose:

```bash
# Clone the repository
git clone https://github.com/AnarManafov/dataharbor.git
cd dataharbor

# Start development environment (includes XRootD server)
cd docker
docker compose up -d

# Access the application at https://localhost:443
```

See **[Docker Deployment Guide](./docker/README.md)** for complete Docker setup instructions.

### Manual Development Setup

For developers who prefer manual setup:

#### Prerequisites

- **Go** 1.24+ (for backend development)
- **Node.js** 18+ & **npm** (for frontend development)
- **XROOTD client** tools (for file system operations)

#### Development Setup

1. **Clone and setup the repository**

   ```shell
   git clone https://github.com/AnarManafov/dataharbor.git
   cd dataharbor
   
   # Install frontend dependencies
   cd web
   npm install
   cd ..
   
   # Install backend dependencies
   cd app
   go mod download
   cd ..
   ```

2. **Start development servers**

   ```shell
   # Start both frontend and backend concurrently
   npm run dev
   
   # Or start them separately:
   npm run dev:frontend  # Frontend on https://localhost:5173
   npm run dev:backend   # Backend on http://localhost:8081
   ```

3. **Access the application**
   - Open your browser to `https://localhost:5173`
   - Accept the self-signed certificate warning for development

## Documentation

> **💡 Complete documentation is available in the [`docs/`](./docs/) folder**

### Getting Started

| Document                                                       | Description                                                  |
| -------------------------------------------------------------- | ------------------------------------------------------------ |
| **[Setup Guide](./docs/SETUP.md)**                             | Development environment setup, dependencies, and first steps |
| **[Development Guide](./docs/DEVELOPMENT.md)**                 | Development workflow, Git conventions, and testing           |
| **[Backend Configuration](./docs/BACKEND_CONFIGURATION.md)**   | Go backend config, environment variables, and YAML settings  |
| **[Frontend Configuration](./docs/FRONTEND_CONFIGURATION.md)** | Vue.js frontend config, SSL setup, and deployment settings   |

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

| Document                                          | Description                                                         |
| ------------------------------------------------- | ------------------------------------------------------------------- |
| **[REST API Reference](./docs/API.md)**           | Complete API endpoints, request/response examples                   |
| **[System Architecture](./docs/ARCHITECTURE.md)** | Overall architecture, design patterns, and streaming implementation |
| **[XROOTD Integration](./docs/xrootd.md)**        | XROOTD client configuration and file operations                     |

### Operations & Deployment

| Document                                               | Description                                                   |
| ------------------------------------------------------ | ------------------------------------------------------------- |
| **[Docker Deployment](./docker/README.md)**            | Docker Compose setup for development and production           |
| **[Docker Quick Start](./docker/QUICKSTART.md)**       | Quick reference commands for Docker deployment                |
| **[Deployment Guide](./docs/DEPLOYMENT.md)**           | Production deployment and environment setup                   |
| **[Testing Guide](./docs/TESTING.md)**                 | Testing strategies, coverage requirements, and test execution |
| **[Troubleshooting Guide](./docs/TROUBLESHOOTING.md)** | Comprehensive issue resolution and debugging                  |

### Quick Reference

- **Development**: Start with [Setup Guide](./docs/SETUP.md) → [Development Guide](./docs/DEVELOPMENT.md)
- **Architecture**: Read [System Architecture](./docs/ARCHITECTURE.md) → [Authentication](./docs/AUTHENTICATION.md)
- **API Development**: Check [Backend Guide](./docs/BACKEND.md) → [API Reference](./docs/API.md)
- **UI Development**: See [Frontend Guide](./docs/FRONTEND.md) → [Components](./docs/FRONTEND.md#components)
- **Deployment**: Follow [Deployment Guide](./docs/DEPLOYMENT.md) → [Backend Configuration](./docs/BACKEND_CONFIGURATION.md)

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
├── app/                    # Go backend application
│   ├── controller/         # HTTP request handlers
│   ├── middleware/         # Authentication, CORS, logging middleware
│   ├── route/             # API route definitions
│   ├── config/            # Configuration management
│   └── docs/api/          # Backend API documentation
├── web/                   # Vue.js frontend application
│   ├── src/
│   │   ├── components/    # Reusable Vue components
│   │   ├── views/         # Page-level components
│   │   ├── api/           # API client and HTTP services
│   │   └── stores/        # Pinia state management
│   └── public/            # Static assets and configuration
├── docs/                  # Developer documentation
├── packaging/             # RPM packaging and build scripts
├── tools/                 # Development and release utilities
└── playground/            # Experimental code and prototypes
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
