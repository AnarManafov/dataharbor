# DataHarbor

![CI Backend](https://github.com/AnarManafov/dataharbor/actions/workflows/backend.yml/badge.svg)
![Coverage](https://img.shields.io/badge/Coverage-32.8%25-yellow)
![CI Frontend](https://github.com/AnarManafov/dataharbor/actions/workflows/frontend.yml/badge.svg)

DataHarbor is a full-stack web application that provides developers with a secure interface to access and manage GSI Lustre cluster data. Built with a Go backend and Vue.js frontend, it offers file browsing, directory navigation, and secure file downloads through XROOTD integration.

## 🎯 Purpose

DataHarbor is designed for **internal use by developers and system administrators** who need to:

- Browse and navigate remote file systems on GSI Lustre clusters
- View detailed file and directory metadata (size, permissions, timestamps)
- Securely download individual files from remote storage
- Manage file staging operations for large data transfers
- Access high-performance computing storage systems through a web interface

## 🏗️ Architecture Overview

- **Backend**: Go REST API server with XROOTD client integration
- **Frontend**: Vue 3 SPA with Element Plus UI components  
- **Authentication**: OpenID Connect (OIDC) with Backend-For-Frontend (BFF) pattern
- **Security**: HTTP-only cookies, server-side session management
- **Storage**: XROOTD protocol for high-performance data access

## 🚀 Quick Start for Developers

### Prerequisites

- **Go** 1.24+ (for backend development)
- **Node.js** 18+ & **npm** (for frontend development)
- **XROOTD client** tools (for file system operations)

### Development Setup

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

## 📚 Developer Documentation

### Getting Started

- **[SETUP.md](./docs/SETUP.md)** - Complete development environment setup and prerequisites
- **[DEVELOPMENT.md](./docs/DEVELOPMENT.md)** - Development workflow, testing, and contribution guidelines

### Architecture & Design

- **[ARCHITECTURE.md](./docs/ARCHITECTURE.md)** - System architecture, design patterns, and component overview
- **[AUTHENTICATION.md](./docs/AUTHENTICATION.md)** - OIDC authentication flow, security model, and BFF pattern

### Component Development

- **[BACKEND.md](./docs/BACKEND.md)** - Go backend development, API design, and XROOTD integration
- **[FRONTEND.md](./docs/FRONTEND.md)** - Vue.js frontend development, components, and state management

### Technical References

| Documentation                                       | Description                                                   |
| --------------------------------------------------- | ------------------------------------------------------------- |
| **[API.md](./docs/API.md)**                         | Complete REST API documentation and examples                  |
| **[XROOTD.md](./docs/XROOTD.md)**                   | XROOTD integration, configuration, and file operations        |
| **[DEPLOYMENT.md](./docs/DEPLOYMENT.md)**           | Production deployment, containerization, and packaging        |
| **[TESTING.md](./docs/TESTING.md)**                 | Testing strategies, coverage requirements, and best practices |
| **[TROUBLESHOOTING.md](./docs/TROUBLESHOOTING.md)** | Common issues and solutions                                   |

## 🔧 Common Development Tasks

### Build for Production

```shell
npm run build
```

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

## 🏗️ Project Structure

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

## 🤝 Contributing

This is an internal project for GSI developers. To contribute:

1. Create a feature branch from `master`
2. Follow the coding standards outlined in development docs
3. Add appropriate tests for new functionality
4. Update documentation as needed
5. Submit a pull request with detailed description

## 📄 License

See [LICENSE](./LICENSE) file for details.

## 🔗 Related Technologies

- [XROOTD](https://xrootd.slac.stanford.edu) - High-performance data access system
- [Vue.js](https://vuejs.org/) - Progressive JavaScript framework  
- [Gin](https://gin-gonic.com/) - Go web framework
- [Element Plus](https://element-plus.org/) - Vue 3 component library
