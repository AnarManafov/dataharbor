# DataHarbor Frontend

Vue 3 + Vite frontend for DataHarbor - provides web interface for XROOTD file system operations.

> **📖 Complete Documentation**: See [../docs/](../docs/) for comprehensive setup, API, and development guides.

## Quick Start

### Prerequisites

- Node.js 18+ / npm

### Development

```shell
# Install dependencies (from repo root)
make deps-frontend

# Development server (with HTTPS)
make dev-frontend

# Development server with PKM certificates
cd web && npm run dev:pkm-certs

# Build for production
make build-frontend
```

## Key Features

- **Vue 3 + Composition API**: Modern reactive framework
- **Pinia State Management**: Centralized application state
- **OIDC Authentication**: Secure authentication with session management
- **File Explorer Interface**: Intuitive directory browsing and file operations
- **Responsive Design**: Works on desktop and mobile devices

## SSL Configuration

The development server uses HTTPS with automatic certificate detection:

1. **Environment variables** (highest priority):

   ```shell
   $env:VITE_SSL_KEY = "/path/to/server.key"
   $env:VITE_SSL_CERT = "/path/to/server.crt"
   npm run dev
   ```

2. **PKM workspace certificates**:

   ```shell
   npm run dev:pkm-certs
   ```

3. **Check certificate status**:

   ```shell
   npm run cert:check
   ```

## Architecture

- **Vue 3 + Composition API**: Modern reactive patterns with TypeScript support
- **Vite Build System**: Fast development server and optimized production builds
- **Component Architecture**: Reusable, well-tested UI components
- **State Management**: Pinia stores for authentication and file operations

## Production Deployment

### RPM Package

```shell
# Build RPM (requires rpm tools)
python3 ../packaging/build_rpm.py -f
```

## Documentation

### Getting Started

| Topic                      | Location                                                               |
| -------------------------- | ---------------------------------------------------------------------- |
| **Complete Setup Guide**   | [../docs/SETUP.md](../docs/SETUP.md)                                   |
| **Development Guide**      | [../docs/DEVELOPMENT.md](../docs/DEVELOPMENT.md)                       |
| **Backend Configuration**  | [../docs/BACKEND_CONFIGURATION.md](../docs/BACKEND_CONFIGURATION.md)   |
| **Frontend Configuration** | [../docs/FRONTEND_CONFIGURATION.md](../docs/FRONTEND_CONFIGURATION.md) |

### Frontend Development

| Topic                    | Location                                               |
| ------------------------ | ------------------------------------------------------ |
| **Frontend Development** | [../docs/FRONTEND.md](../docs/FRONTEND.md)             |
| **Authentication**       | [../docs/AUTHENTICATION.md](../docs/AUTHENTICATION.md) |
| **Testing Guide**        | [../docs/TESTING.md](../docs/TESTING.md)               |

### Architecture & Operations

| Topic                     | Location                                                 |
| ------------------------- | -------------------------------------------------------- |
| **Architecture Overview** | [../docs/ARCHITECTURE.md](../docs/ARCHITECTURE.md)       |
| **API Documentation**     | [../docs/API.md](../docs/API.md)                         |
| **Deployment Guide**      | [../docs/DEPLOYMENT.md](../docs/DEPLOYMENT.md)           |
| **Troubleshooting**       | [../docs/TROUBLESHOOTING.md](../docs/TROUBLESHOOTING.md) |

### Quick Navigation

- **📁 All Documentation**: [../docs/](../docs/)
- **🎨 Frontend Development**: [../docs/FRONTEND.md](../docs/FRONTEND.md)
- **🔒 Authentication**: [../docs/AUTHENTICATION.md](../docs/AUTHENTICATION.md)
- **🏗️ Architecture**: [../docs/ARCHITECTURE.md](../docs/ARCHITECTURE.md)
