# DataHarbor Documentation

[← Back to Main README](../README.md)

Welcome to the complete documentation for DataHarbor - a secure web interface for accessing XRootD-backed storage.

---

## Documentation Overview

This documentation is organized into logical sections to help you find information quickly. Each document includes detailed technical information with diagrams and examples.

## 📚 Quick Navigation

| I want to...                      | Go to                                                                                         |
| --------------------------------- | --------------------------------------------------------------------------------------------- |
| Set up my development environment | [Setup Guide](./SETUP.md) or [Dev Container Guide](../.devcontainer/README.md)                |
| Understand the architecture       | [System Architecture](./ARCHITECTURE.md)                                                      |
| Learn the development workflow    | [Development Guide](./DEVELOPMENT.md)                                                         |
| Work on the backend               | [Backend Development](./BACKEND.md) → [Backend Configuration](./BACKEND_CONFIGURATION.md)     |
| Work on the frontend              | [Frontend Development](./FRONTEND.md) → [Frontend Configuration](./FRONTEND_CONFIGURATION.md) |
| Use or integrate with the API     | [REST API Reference](./API.md)                                                                |
| Deploy to production              | [Deployment Guide](./DEPLOYMENT.md) or [Docker Guide](../docker/README.md)                    |
| Debug an issue                    | [Troubleshooting Guide](./TROUBLESHOOTING.md)                                                 |

---

## Getting Started

Essential guides for new developers and setting up development environments.

| Document                                                  | Description                                                  |
| --------------------------------------------------------- | ------------------------------------------------------------ |
| **[Setup Guide](./SETUP.md)**                             | Development environment setup, dependencies, and first steps |
| **[Development Guide](./DEVELOPMENT.md)**                 | Git workflow, coding standards, testing, and CI/CD           |
| **[Backend Configuration](./BACKEND_CONFIGURATION.md)**   | Go backend config, environment variables, and YAML settings  |
| **[Frontend Configuration](./FRONTEND_CONFIGURATION.md)** | Vue.js frontend config, SSL setup, and deployment settings   |

## Architecture & Design

Core architectural concepts and security implementation.

| Document                                         | Description                                               |
| ------------------------------------------------ | --------------------------------------------------------- |
| **[System Architecture](./ARCHITECTURE.md)**     | Component architecture, BFF pattern, and design decisions |
| **[Authentication System](./AUTHENTICATION.md)** | OIDC integration, security model, and session management  |

## Component Development

Detailed development guides for backend and frontend components.

| Document                                  | Description                                            |
| ----------------------------------------- | ------------------------------------------------------ |
| **[Backend Development](./BACKEND.md)**   | Go API development, middleware, and XROOTD integration |
| **[Frontend Development](./FRONTEND.md)** | Vue.js development, components, and state management   |

## Technical References

Comprehensive technical documentation and API references.

| Document                                     | Description                                          |
| -------------------------------------------- | ---------------------------------------------------- |
| **[REST API Reference](./API.md)**           | Complete API endpoints and examples                  |
| **[System Architecture](./ARCHITECTURE.md)** | Overall architecture, design patterns, and data flow |
| **[XROOTD Integration](./xrootd.md)**        | XROOTD client and file operations                    |

## Operations & Deployment

Production deployment and operational guides.

| Document                                          | Description                                  |
| ------------------------------------------------- | -------------------------------------------- |
| **[Deployment Guide](./DEPLOYMENT.md)**           | Production deployment and environment setup  |
| **[Docker Deployment](../docker/README.md)**      | Docker Compose setup for dev and production  |
| **[Testing Guide](./TESTING.md)**                 | Testing strategies and coverage requirements |
| **[Troubleshooting Guide](./TROUBLESHOOTING.md)** | Comprehensive issue resolution and debugging |
| **[Versioning](./VERSIONING.md)**                 | Backend version management and build process |

## Quick Navigation

### For New Developers

1. **[Setup Guide](./SETUP.md)** - Get your development environment running
2. **[System Architecture](./ARCHITECTURE.md)** - Understand the overall design
3. **[Development Guide](./DEVELOPMENT.md)** - Learn the development workflow
4. Choose your focus: **[Backend Development](./BACKEND.md)** or **[Frontend Development](./FRONTEND.md)**

### For System Administrators

1. **[System Architecture](./ARCHITECTURE.md)** - Understand the system design
2. **[Authentication System](./AUTHENTICATION.md)** - Critical security information
3. **[Deployment Guide](./DEPLOYMENT.md)** - Production deployment strategies
4. **[Troubleshooting Guide](./TROUBLESHOOTING.md)** - Handle operational and development issues

### For API Integration

1. **[Authentication System](./AUTHENTICATION.md)** - Authentication requirements
2. **[REST API Reference](./API.md)** - Complete endpoint documentation
3. **[XROOTD Integration](./xrootd.md)** - File system operations

## Documentation Features

This documentation includes:

- **Visual Diagrams**: Mermaid diagrams for complex workflows and architectures
- **Code Examples**: Tested and verified code snippets
- **Comprehensive Coverage**: All system components and integrations
- **Troubleshooting**: Common issues with step-by-step solutions
- **Best Practices**: Development and deployment recommendations

## Contributing

To contribute to the documentation:

1. Verify accuracy against current implementation
2. Follow existing style and format conventions
3. Include diagrams for complex concepts
4. Test any code examples
5. Update related documentation as needed

## Additional Resources

- **[Main Repository](https://github.com/AnarManafov/dataharbor)** - Source code and issues
- **[XROOTD Documentation](https://xrootd.slac.stanford.edu)** - Official XROOTD docs
- **[Vue.js Guide](https://vuejs.org/guide/)** - Vue.js framework documentation
- **[Go Documentation](https://golang.org/doc/)** - Go programming language docs

---

[← Back to Main README](../README.md) | [↑ Top](#dataharbor-documentation)

