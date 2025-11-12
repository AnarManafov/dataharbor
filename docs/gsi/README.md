# GSI Environment Documentation

[← Back to Documentation](../README.md)

This section contains specialized documentation for deploying and configuring DataHarbor in the GSI (GSI Helmholtz Centre for Heavy Ion Research) environment.

---

## Deployment Options

DataHarbor supports two deployment methods for GSI infrastructure:

### Method 1: Docker Compose (Recommended)

The **recommended approach** for production deployments. Docker Compose provides:

- ✅ Simplified setup with pre-configured services (Backend, Frontend, XRootD, Nginx)
- ✅ Automatic TLS/SSL configuration
- ✅ Built-in XRootD ZTN authentication
- ✅ Easy version updates via container images
- ✅ Consistent environments across different servers

**See:** [Docker Deployment Guide](../../docker/README.md) — Complete guide for production deployment with Docker Compose

### Method 2: RPM Installation (Manual)

For environments requiring traditional system package deployment:

- Manual installation of backend and frontend RPM packages
- SystemD service configuration
- Manual XRootD server setup and configuration
- Nginx reverse proxy configuration

**See:** [RPM Deployment Guide](./DEPLOYMENT_GSI.md) — Step-by-step manual installation guide

---

## GSI Documentation

| Document                                        | Description                                                 |
| ----------------------------------------------- | ----------------------------------------------------------- |
| **[Docker Deployment](../../docker/README.md)** | **Recommended:** Production deployment with Docker Compose  |
| **[RPM Deployment Guide](./DEPLOYMENT_GSI.md)** | Manual RPM-based installation with XRootD ZTN configuration |

---

## Overview

These guides are designed for the GSI infrastructure and cover:

- **Docker Compose Deployment** (Recommended): Complete containerized setup with all services
- **RPM Package Deployment**: Manual installation using RPM packages on RHEL/CentOS systems
- **SystemD Integration**: Service management with systemd unit files
- **GEANT SSL Certificates**: Using institutional certificates from GEANT CA
- **GSI Keycloak**: Integration with GSI's OIDC provider at id.gsi.de
- **XRootD ZTN**: Zero-Trust Networking protocol for secure file operations

## Prerequisites

Before using these guides, ensure you have:

- Root access to GSI server infrastructure
- SSL certificates from GEANT CA (or self-signed for testing)
- Access to GSI Keycloak (id.gsi.de)
- Docker and Docker Compose (for Docker deployment)
- RHEL/CentOS 8+ system (for RPM deployment)

## Quick Start

### For New Deployments (Recommended)

1. **Docker Compose**: Follow the [Docker Deployment Guide](../../docker/README.md)
   - Download `docker-compose.prod.yml` or `docker-compose.deploy.yml`
   - Configure `.env` file with your settings
   - Run `docker compose up -d`

### For Manual/RPM Deployments

1. **RPM Installation**: Start with [RPM Deployment Guide](./DEPLOYMENT_GSI.md)
2. **XRootD Configuration**: Configure [XRootD ZTN](./XROOTD_ZTN_SETUP.md) for OAuth support

---

## Related Documentation

- **[Main Deployment Guide](../DEPLOYMENT.md)** - General deployment (non-GSI)
- **[Docker Deployment](../../docker/README.md)** - Container-based deployment
- **[XRootD Integration](../xrootd.md)** - General XRootD client documentation

---

[← Back to Documentation](../README.md) | [↑ Top](#gsi-environment-documentation)
