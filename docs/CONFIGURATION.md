# Configuration Guide

This document describes DataHarbor's unified configuration system using Viper and environment variables.

## Overview

DataHarbor uses a **single, unified configuration approach** with the following features:

- **Viper-based configuration** loading with struct binding
- **Environment variable support** with `DATAHARBOR_` prefix
- **Configuration validation** for critical fields
- **Unified logging configuration** replacing dual log/logger sections
- **Optimized default values** for production and development

## Configuration Structure

### Main Configuration Sections

```yaml
# Environment: development, production, testing
env: development

# HTTP server configuration
server:
  address: ":8080"
  debug: false
  shutdown_timeout: "30s"
  cors: { ... }
  ssl: { ... }

# Unified logging configuration (replaces old log + logger sections)
logging:
  level: "info"          # Global log level: debug, info, warn, error
  format: "json"         # Global format: json, text
  console: { ... }       # Console output settings
  file: { ... }          # File output with rotation

# XRootD server configuration
xrd:
  host: "localhost"
  port: 1094
  initial_dir: "/tmp"
  user_required: false
  # ... other XRD settings

# Authentication and OIDC
auth:
  enabled: false
  oidc: { ... }

# Frontend asset serving
frontend:
  url: "http://localhost:5173"
  # ... other frontend settings
```

## Unified Logging Configuration

### Migration from Old Structure

**Old structure (deprecated):**
```yaml
# Basic logging
log:
  format: text
  level: info

# Advanced logging  
logger:
  console:
    driver: console
    level: debug
  file:
    driver: file
    filename: ./log/app.log
```

**New unified structure:**
```yaml
# Single logging configuration
logging:
  level: info           # Global level
  format: json         # Global format
  console:
    enabled: true      # Enable/disable console output
    level: info        # Override global level (optional)
    format: text       # Override global format (optional)
  file:
    enabled: true      # Enable/disable file output
    level: info        # Override global level (optional)
    format: json       # Override global format (optional)
    filename: "./log/dataharbor.log"
    maxsize: 10        # MB
    maxbackups: 5      # Number of backup files
    maxage: 30         # Days to retain
    compress: true     # Compress rotated files
```

## Environment Variables

All configuration values can be overridden using environment variables with the `DATAHARBOR_` prefix:

### Environment Variable Naming

- Use `DATAHARBOR_` prefix
- Replace dots (`.`) with underscores (`_`)
- Use uppercase

### Examples

```bash
# Server configuration
export DATAHARBOR_SERVER_ADDRESS=":8081"
export DATAHARBOR_SERVER_DEBUG="true"

# Logging configuration
export DATAHARBOR_LOGGING_LEVEL="debug"
export DATAHARBOR_LOGGING_CONSOLE_ENABLED="true"
export DATAHARBOR_LOGGING_FILE_ENABLED="false"

# XRootD configuration
export DATAHARBOR_XRD_HOST="xrootd.example.com"
export DATAHARBOR_XRD_PORT="1094"

# Authentication
export DATAHARBOR_AUTH_ENABLED="true"
export DATAHARBOR_AUTH_OIDC_CLIENT_SECRET="your-secret"
export DATAHARBOR_AUTH_OIDC_SESSION_SECRET="your-session-secret"
```

## Configuration Validation

The system validates critical configuration fields on startup:

### Validated Fields

- `server.address` - Required, must not be empty
- `xrd.host` - Required, must not be empty  
- `xrd.port` - Required, must be > 0
- `logging.level` - Must be one of: debug, info, warn, error
- `auth.oidc.issuer` - Required when auth is enabled
- `auth.oidc.client_id` - Required when auth is enabled

### Validation Errors

If validation fails, the application will:
1. Log the specific validation error
2. Exit with error code 1
3. Prevent startup with invalid configuration

## Configuration Loading Process

1. **Command line flags**: `--config=path/to/config.yaml`
2. **Environment variables**: `DATAHARBOR_*` variables
3. **Configuration file**: YAML format with validation
4. **Default values**: Optimized fallbacks

### Loading Order (highest to lowest precedence)

1. Environment variables
2. Configuration file values
3. Default values

## Configuration Files

### Development Configuration

**File**: `app/config/application.development.yaml`

```yaml
env: development

server:
  address: ":22000"
  debug: true

logging:
  level: debug
  format: text
  console:
    enabled: true
    level: debug
    format: text
  file:
    enabled: true
    level: debug
    format: json
    filename: "./log/dataharbor_app.log"
    maxsize: 10
    maxbackups: 2
    maxage: 7
    compress: true

auth:
  enabled: true
  oidc:
    issuer: "https://id.gsi.de/realms/wl"
    client_id: "xrootd"
```

### Production Configuration

**File**: `app/config/application.production.yaml`

```yaml
env: production

server:
  address: ":8080"
  debug: false

logging:
  level: info
  format: json
  console:
    enabled: true
    level: info
    format: json
  file:
    enabled: true
    level: info
    format: json
    filename: "/var/log/dataharbor/app.log"
    maxsize: 100
    maxbackups: 10
    maxage: 30
    compress: true

auth:
  enabled: true
  oidc:
    client_secret: "${DATAHARBOR_CLIENT_SECRET}"
    session_secret: "${DATAHARBOR_SESSION_SECRET}"
```
