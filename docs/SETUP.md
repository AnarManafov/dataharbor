# Development Environment Setup

[← Back to Documentation](./README.md)

This guide covers the complete setup process for DataHarbor development environment.

## Setup Process Overview

### System Requirements

- **Operating System**: Windows, macOS, or Linux
- **Git**: Latest version
- **Go**: Version 1.24 or higher
- **Node.js**: Version 18 or higher (includes npm)
- **XROOTD Client**: Required for backend file operations

### Installing Dependencies

#### Go Installation

```shell
# Windows (using winget)
winget install GoLang.Go

# macOS (using Homebrew)
brew install go

# Linux (Ubuntu/Debian)
sudo apt update
sudo apt install golang-go
```

Verify installation:

```shell
go version
```

#### Node.js and npm Installation

```shell
# Windows (using winget)
winget install OpenJS.NodeJS.LTS

# macOS (using Homebrew)
brew install node

# Linux (Ubuntu/Debian)
sudo apt update sudo apt install nodejs npm
```

Verify installation:

```shell
node --version
npm --version
```

#### XROOTD Client Installation

```shell
# macOS (using Homebrew)
brew install xrootd

# Linux (Ubuntu/Debian)
sudo apt update
sudo apt install xrootd-client

# CentOS/RHEL/Fedora
sudo dnf install xrootd-client
```

For Windows, you may need to use WSL or install through alternative methods.

## Project Setup

### 1. Clone Repository

```shell
git clone https://github.com/AnarManafov/dataharbor.git
cd dataharbor
```

### 2. Install Project Dependencies

```shell
make deps
```

This installs both backend (Go modules) and frontend (npm) dependencies.

### 3. Development Configuration

#### Backend Configuration

1. Copy the template configuration:

   ```shell
   cd app/config
   cp application.template.yaml application.development.yaml
   ```

2. Edit `application.development.yaml` with your settings. Key settings to configure:

   | Setting           | Description                                        |
   | ----------------- | -------------------------------------------------- |
   | `server.address`  | Backend port (default: `:8081`)                    |
   | `xrd.host`        | Your XROOTD server hostname                        |
   | `xrd.initial_dir` | Starting directory for file browser                |
   | `auth.enabled`    | Set `false` for local dev, `true` for auth testing |

   > **📖 Complete configuration reference:** See **[Backend Configuration Guide](./BACKEND_CONFIGURATION.md)** for all available options, environment variables, and production examples.

#### Frontend Configuration

1. The frontend automatically proxies API calls to the backend during development
2. SSL certificates are handled automatically (see [Certificate Setup](#ssl-certificate-setup))

   > **📖 Complete frontend config:** See **[Frontend Configuration Guide](./FRONTEND_CONFIGURATION.md)** for all options.

### 4. SSL Certificate Setup (Development)

For HTTPS development, DataHarbor supports multiple certificate locations:

#### Option 1: Environment Variables (Recommended)

```shell
$env:VITE_SSL_KEY = "C:\path\to\your\server.key"
$env:VITE_SSL_CERT = "C:\path\to\your\server.crt"
```

#### Option 2: Check Certificate Status

```shell
cd web
npm run cert:check
```

#### Option 3: Use Example Setup Script

```shell
cd web
# Review and modify scripts/setup-certs-example.sh for your needs
npm run cert:setup
```

If no certificates are found, the development server will run in HTTP mode.

## Running the Development Environment

### Quick Start Commands

| Command             | Description                                  |
| ------------------- | -------------------------------------------- |
| `make dev`          | Start both frontend and backend concurrently |
| `make dev-frontend` | Start frontend only (https://localhost:5173) |
| `make dev-backend`  | Start backend only (http://localhost:8081)   |
| `make build`        | Build both for production                    |

### Starting Both Services

```shell
make dev
```

This starts:
- **Frontend**: https://localhost:5173 (with hot reload)
- **Backend**: http://localhost:8081 (with auto-restart)

### Starting Services Separately

```shell
# Terminal 1: Backend
make dev-backend

# Terminal 2: Frontend
make dev-frontend
```

> **📖 For detailed development workflow**, including Git branching, CI/CD, and contribution guidelines, see **[Development Guide](./DEVELOPMENT.md)**.

## Building for Production

```shell
# Build both frontend and backend
make build

# Build individually
make build-frontend
make build-backend
```

## Running Tests

#### Backend Tests

```shell
# Run all tests with coverage
make test

# Verbose output
make test-verbose

# HTML coverage report
make test-coverage-html
```

#### Frontend Tests

```shell
cd web
npm test  # If test framework is configured
```

## Development Tools and Scripts

### Version Management

```shell
# Synchronize versions across components
make sync-versions

# Prepare for release
make prepare-release
```

### Certificate Management

```shell
cd web
npm run cert:check  # Check certificate status
npm run cert:setup  # Setup certificates (review script first)
```

## IDE Configuration

### VS Code (Recommended)

Install these extensions:

- Go extension
- Vue Language Features (Volar)
- TypeScript Vue Plugin (Volar)
- GitLens
- Better Comments

### Go Module Configuration

Ensure your Go workspace is properly configured:

```shell
cd app
go mod init github.com/AnarManafov/dataharbor/app  # Already done
make tidy
```

## Next Steps

Once your environment is set up:

1. Read the [ARCHITECTURE.md](./ARCHITECTURE.md) to understand the system design
2. Review [BACKEND.md](./BACKEND.md) for backend development
3. Review [FRONTEND.md](./FRONTEND.md) for frontend development
4. Check [API.md](./API.md) for available endpoints

### Need Help?

For troubleshooting common issues, see the **[Troubleshooting Guide](./TROUBLESHOOTING.md)**.

### Logs and Debugging

- Backend logs: Check console output when running `npm run dev:backend`
- Frontend logs: Check browser console and terminal output
- Network requests: Use browser DevTools Network tab

---

[← Back to Documentation](./README.md) | [↑ Top](#development-environment-setup)
