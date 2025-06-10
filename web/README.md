# DataHarbor Frontend

Vue 3 + Vite frontend for DataHarbor - provides web interface for XROOTD file system operations.

> **📖 Complete Documentation**: See [../docs/](../docs/) for comprehensive setup, API, and development guides.

## Quick Start

### Prerequisites

- Node.js 18+ / npm

### Development

```shell
# Install dependencies
npm install

# Development server (with HTTPS)
npm run dev

# Development server with PKM certificates
npm run dev:pkm-certs

# Build for production
npm run build
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

### Container (Recommended)

```shell
# Build container with nginx
podman build -t dataharbor-frontend:latest .

# Run container
podman run -p 8080:8080 dataharbor-frontend:latest
```

### RPM Package

```shell
# Build RPM (requires rpm tools)
python3 ../packaging/build_rpm.py -f
```

## Documentation

| Topic                     | Location                                           |
| ------------------------- | -------------------------------------------------- |
| **Complete Setup Guide**  | [../docs/SETUP.md](../docs/SETUP.md)               |
| **Frontend Development**  | [../docs/FRONTEND.md](../docs/FRONTEND.md)         |
| **Architecture Overview** | [../docs/ARCHITECTURE.md](../docs/ARCHITECTURE.md) |
| **Testing Guide**         | [../docs/TESTING.md](../docs/TESTING.md)           |
| **Deployment Guide**      | [../docs/DEPLOYMENT.md](../docs/DEPLOYMENT.md)     |
