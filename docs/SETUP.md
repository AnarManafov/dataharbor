# Development Environment Setup

This guide covers the complete setup process for DataHarbor development environment.

## Prerequisites

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

#### Backend Dependencies

```shell
cd app
go mod download
go mod tidy
cd ..
```

#### Frontend Dependencies

```shell
cd web
npm install
cd ..
```

### 3. Development Configuration

#### Backend Configuration

1. Copy the template configuration:

   ```shell
   cd app/config
   copy application.template.yaml application.development.yaml
   ```

1. Edit `application.development.yaml` with your settings:

   ```yaml
   # Server configuration
   server:
     port: 8081
     host: "localhost"
   
   # XROOTD configuration
   xrd:
     server: "your-xrootd-server.example.com"
     initial_dir: "/your/initial/path"
   
   # Authentication (optional for development)
   auth:
     enabled: false  # Set to true when testing auth
     oidc:
       issuer: "https://your-oidc-provider.com"
       client_id: "your-client-id"
       client_secret: "your-client-secret"
   ```

#### Frontend Configuration

1. The frontend automatically proxies API calls to the backend during development
2. SSL certificates are handled automatically (see [Certificate Setup](#ssl-certificate-setup))

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

## Development Workflow

### Starting Development Servers

#### Option 1: Start Both Services Concurrently

```shell
npm run dev
```

This starts both frontend (https://localhost:5173) and backend (http://localhost:8081).

#### Option 2: Start Services Separately

**Backend:**

```shell
npm run dev:backend
# Or with custom config:
$env:CONFIG_FILE_PATH = "app/config/application.development.yaml"
npm run dev:backend
```

**Frontend:**

```shell
npm run dev:frontend
```

### Building for Production

```shell
# Build both frontend and backend
npm run build

# Build individually
npm run build:frontend
npm run build:backend
```

### Running Tests

#### Backend Tests

```shell
cd app
go test -v ./...

# With coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
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
npm run sync-versions

# Prepare for release
npm run prepare-release
```

### Sandbox Environment

```shell
# Create sandbox
npm run sandbox:create

# Run sandbox
npm run sandbox:run

# Clean sandbox
npm run sandbox:clean
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
go mod tidy
```

## Troubleshooting

### Common Issues

1. **Port conflicts**: Change ports in configuration files
2. **SSL certificate issues**: Use HTTP mode for development or setup proper certificates
3. **Go module issues**: Run `go mod tidy` and `go mod download`
4. **npm dependency issues**: Delete `node_modules` and run `npm install`

### Environment Variables

Set these for consistent development:

```shell
# Optional: Backend config file
$env:CONFIG_FILE_PATH = "app/config/application.development.yaml"

# Optional: SSL certificates
$env:VITE_SSL_KEY = "path/to/server.key"
$env:VITE_SSL_CERT = "path/to/server.crt"
```

### Logs and Debugging

- Backend logs: Check console output when running `npm run dev:backend`
- Frontend logs: Check browser console and terminal output
- Network requests: Use browser DevTools Network tab

## Next Steps

Once your environment is set up:

1. Read the [ARCHITECTURE.md](./ARCHITECTURE.md) to understand the system design
2. Review [BACKEND.md](./BACKEND.md) for backend development
3. Review [FRONTEND.md](./FRONTEND.md) for frontend development
4. Check [API.md](./API.md) for available endpoints
