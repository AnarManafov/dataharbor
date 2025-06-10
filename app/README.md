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

- **File System Operations**: Directory listing, file staging via XROOTD
- **Authentication**: OIDC integration with BFF pattern
- **RESTful API**: JSON-based API with consistent error handling
- **File Staging**: Temporary file staging for downloads with automatic cleanup

## Architecture

- **XROOTD Integration**: Uses command-line client calls with async timeouts
- **HTTP Server**: Lightweight web server with configurable port
- **Middleware Stack**: Authentication, CORS, logging, recovery
- **File Operations**: Asynchronous file staging with cleanup jobs

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

| Topic                     | Location                                           |
| ------------------------- | -------------------------------------------------- |
| **Complete Setup Guide**  | [../docs/SETUP.md](../docs/SETUP.md)               |
| **API Documentation**     | [../docs/API.md](../docs/API.md)                   |
| **Backend Development**   | [../docs/BACKEND.md](../docs/BACKEND.md)           |
| **Architecture Overview** | [../docs/ARCHITECTURE.md](../docs/ARCHITECTURE.md) |
| **Testing Guide**         | [../docs/TESTING.md](../docs/TESTING.md)           |
| **Deployment Guide**      | [../docs/DEPLOYMENT.md](../docs/DEPLOYMENT.md)     |
