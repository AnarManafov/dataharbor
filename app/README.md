# DataHarbor Backend

Go backend for DataHarbor - provides REST API for XROOTD file system operations and authentication.

> **📖 Complete Documentation**: See [../docs/](../docs/) for comprehensive setup, API, and development guides.

## Quick Start

### Prerequisites

- Go 1.24+
- XROOTD client tools

### Development

```shell
# Install dependencies
go mod download

# Run with auto-reload
go run .

# Run with specific config
go run . --config=config/application.development.yaml
```

### Testing

```shell
# Run all tests
go test -v ./...

# Run with coverage
go test -cover ./...
```

## Key Features

- **File System Operations**: Directory listing, file streaming via XROOTD
- **Authentication**: OIDC integration with BFF pattern
- **RESTful API**: JSON-based API with consistent error handling
- **File Streaming**: Direct file streaming for downloads with zero temporary storage

## Architecture

- **XROOTD Integration**: Uses command-line client calls with async timeouts
- **HTTP Server**: Lightweight web server with configurable port
- **Middleware Stack**: Authentication, CORS, logging, recovery
- **File Operations**: Direct file streaming with performance logging

## Production Deployment

### Container (Recommended)

```shell
# Build container
podman build -t dataharbor-backend:latest .

# Run container
podman run --network=host dataharbor-backend:latest
```

### RPM Package

```shell
# Build RPM (requires rpm tools)
python3 ../packaging/build_rpm.py -b
```

## Documentation

### Getting Started

| Topic                      | Location                                                               |
| -------------------------- | ---------------------------------------------------------------------- |
| **Complete Setup Guide**   | [../docs/SETUP.md](../docs/SETUP.md)                                   |
| **Development Guide**      | [../docs/DEVELOPMENT.md](../docs/DEVELOPMENT.md)                       |
| **Backend Configuration**  | [../docs/BACKEND_CONFIGURATION.md](../docs/BACKEND_CONFIGURATION.md)   |
| **Frontend Configuration** | [../docs/FRONTEND_CONFIGURATION.md](../docs/FRONTEND_CONFIGURATION.md) |

### Backend Development

| Topic                   | Location                                               |
| ----------------------- | ------------------------------------------------------ |
| **Backend Development** | [../docs/BACKEND.md](../docs/BACKEND.md)               |
| **API Documentation**   | [../docs/API.md](../docs/API.md)                       |
| **Authentication**      | [../docs/AUTHENTICATION.md](../docs/AUTHENTICATION.md) |
| **Testing Guide**       | [../docs/TESTING.md](../docs/TESTING.md)               |

### Architecture & Operations

| Topic                     | Location                                                 |
| ------------------------- | -------------------------------------------------------- |
| **Architecture Overview** | [../docs/ARCHITECTURE.md](../docs/ARCHITECTURE.md)       |
| **XROOTD Integration**    | [../docs/xrootd.md](../docs/xrootd.md)                   |
| **Deployment Guide**      | [../docs/DEPLOYMENT.md](../docs/DEPLOYMENT.md)           |
| **Troubleshooting**       | [../docs/TROUBLESHOOTING.md](../docs/TROUBLESHOOTING.md) |

### Quick Navigation

- **📁 All Documentation**: [../docs/](../docs/)
- **🔧 Development Setup**: [../docs/DEVELOPMENT.md](../docs/DEVELOPMENT.md)
- **🌐 API Reference**: [../docs/API.md](../docs/API.md)
- **🏗️ Architecture**: [../docs/ARCHITECTURE.md](../docs/ARCHITECTURE.md)
