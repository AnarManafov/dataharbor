# Certificate Init Container

## Purpose

This lightweight init container generates self-signed SSL/TLS certificates for development environments. It runs once at startup, creates the necessary certificates, and exits successfully. Other services depend on it using Docker Compose's `service_completed_successfully` condition.

## How It Works

1. **Startup**: Runs before any other service that needs certificates
2. **Generation**: Creates self-signed certificates with multiple naming conventions
3. **Completion**: Exits with success status, signaling dependent services to start
4. **Reuse**: If certificates already exist in the volume, skips generation

## Generated Files

The container generates certificates in a shared volume with multiple naming conventions to support different services:

### For Nginx

- `server.crt` - SSL certificate
- `server.key` - Private key

### For XRootD

- `hostcert.pem` - SSL certificate (symlink to server.crt)
- `hostkey.pem` - Private key (symlink to server.key)
- `hostcert_combined.pem` - Combined cert+key file

### Certificate Details

- **Algorithm**: RSA 2048-bit
- **Validity**: 365 days
- **Subject**: `/C=DE/ST=Hessen/L=Darmstadt/O=DataHarbor/OU=Development/CN=localhost`
- **SANs**: localhost, *.localhost, xrootd, nginx, 127.0.0.1
- **Permissions**: 644 (world-readable)

## Architecture Pattern

This follows Docker Compose best practices for init containers:

```yaml
services:
  cert-init:
    # Runs once and exits
    
  nginx:
    depends_on:
      cert-init:
        condition: service_completed_successfully
    volumes:
      - shared-certs:/etc/nginx/ssl:ro
      
  xrootd:
    depends_on:
      cert-init:
        condition: service_completed_successfully
    volumes:
      - shared-certs:/var/run/xrootd/certs:ro
```

## Production

This container is **development-only**. In production environments, real certificates are provided via environment variables and mounted directly into services.
