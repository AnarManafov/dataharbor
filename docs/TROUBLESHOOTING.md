# DataHarbor Troubleshooting Guide

Comprehensive troubleshooting guide for common issues in DataHarbor development, deployment, and operations.

## Quick Navigation

- [Development Environment Issues](#development-environment-issues)
- [Backend Development Issues](#backend-development-issues)
- [Frontend Development Issues](#frontend-development-issues)
- [Configuration Issues](#configuration-issues)
- [XROOTD Connection Issues](#xrootd-connection-issues)
- [Deployment and Production Issues](#deployment-and-production-issues)
- [Authentication Issues](#authentication-issues)
- [Performance Issues](#performance-issues)
- [Debugging Tools](#debugging-tools)

## Development Environment Issues

### Initial Setup Problems

**Port conflicts**: Change ports in configuration files

- Backend: Modify `server.port` in `app/config/application.yaml`
- Frontend: Use `--port` flag or environment variable

**SSL certificate issues**: Use HTTP mode for development or setup proper certificates

```bash
# Check certificate status
npm run cert:check

# Setup development certificates
npm run cert:setup

# Use PKM certificates
npm run dev:pkm-certs
```

**Go module issues**: Run dependency cleanup

```bash
cd app
go mod tidy
go mod download
```

**npm dependency issues**: Reset Node.js dependencies

```bash
cd web
rm -rf node_modules package-lock.json
npm install
```

### Environment Variables

Set these for consistent development:

```bash
# Backend config file
export CONFIG_FILE_PATH="app/config/application.development.yaml"

# SSL certificates (optional)
export VITE_SSL_KEY="path/to/server.key"
export VITE_SSL_CERT="path/to/server.crt"
```

## Backend Development Issues

### XROOTD Connection Failures

**Test XROOTD connectivity**:

```bash
# Basic connectivity test
xrdfs root://server.gsi.de:1094 ls /

# Server ping test
xrdfs root://server.gsi.de:1094 ping

# Check server configuration
cat app/config/application.development.yaml
```

**Common causes**:

- Network connectivity issues
- Firewall blocking port 1094
- XROOTD server down or misconfigured
- Authentication/authorization problems

### Authentication Problems

**Verify OIDC configuration**:

```bash
# Check OIDC discovery endpoint
curl -s https://keycloak.gsi.de/realms/dataharbor/.well-known/openid_configuration

# Verify redirect URIs match configuration
# Check browser dev tools -> Application -> Cookies for session data
```

**Common issues**:

- Incorrect redirect URIs
- Invalid client credentials
- Clock synchronization issues
- Network connectivity to OIDC provider

### Service Startup Issues

**Backend won't start**:

```bash
# Run with debug logging
cd app
CONFIG_FILE_PATH=config/application.development.yaml go run .

# Run with race detection
go run -race .

# Check for port conflicts
lsof -i :8081
```

## Frontend Development Issues

### Build Issues

**Certificate Problems**:

```bash
# Verify SSL certificate paths and permissions
npm run cert:check

# Check environment variables
echo $VITE_SSL_KEY
echo $VITE_SSL_CERT
```

**Dependency Conflicts**:

```bash
# Clear and reinstall dependencies
rm -rf node_modules package-lock.json
npm install
```

**Memory Issues**:

```bash
# Increase Node.js memory limit for large builds
export NODE_OPTIONS="--max-old-space-size=4096"
npm run build
```

### Runtime Issues

**Authentication Failures**:

- Check OIDC configuration and backend connectivity
- Verify session cookies in browser dev tools
- Check for CORS issues in browser console

**API Connection Problems**:

```bash
# Verify backend is running
curl http://localhost:8081/api/v1/health

# Check proxy configuration in vite.config.js
# Look for CORS errors in browser console
```

**Performance Issues**:

- Monitor network requests in dev tools
- Check for component re-renders in Vue DevTools
- Analyze bundle size: `npm run build:analyze`

### Development Environment

**Hot Reload Issues**:

- Check file watchers are working
- Restart development server
- Clear browser cache

**HTTPS Certificate Warnings**:

- Install and trust development certificates
- Use `npm run cert:setup` for automatic setup

**Port Conflicts**:

- Use alternative ports if defaults are occupied
- Check what's using the port: `lsof -i :5173`

## Configuration Issues

### Backend Configuration

**Config file not found**:

- Check file path in `--config` flag
- Ensure file exists and is readable
- Application will create a default config if none exists

**Environment variables not working**:

```bash
# Verify DATAHARBOR_ prefix
export DATAHARBOR_SERVER_PORT=8082

# Check underscore vs dot conversion
# DATAHARBOR_SERVER_PORT maps to server.port in YAML

# Use quotes for complex values
export DATAHARBOR_OIDC_CLIENT_SECRET="complex-secret-value"
```

**Validation errors**:

- Check required fields are set
- Verify data types match expectations
- Review conditional requirements in configuration

**Logging not working**:

- Ensure log directory exists and is writable
- Check `logging.enabled` settings
- Verify log level configuration

### Frontend Configuration

**Certificate not found**:

```bash
# Check certificate locations
npm run cert:check

# Set environment variables
export VITE_SSL_KEY="/path/to/key"
export VITE_SSL_CERT="/path/to/cert"
```

**API calls failing**:

- Check `public/config.json` exists
- Verify `apiBaseUrl` configuration
- Ensure backend is running and accessible

**Config not loading**:

- Check `/config.json` is accessible
- Verify JSON syntax is valid
- Check browser network tab for 404 errors

**HTTPS not working**:

- Verify certificate files exist and are readable
- Check certificate paths in environment variables
- Accept browser certificate warnings for development

## XROOTD Connection Issues

### Connection Timeouts

**Symptoms**: Operations hang or timeout

```bash
# Test server connectivity
xrdfs root://server.example.com:1094 ping

# Check server status
xrdfs root://server.example.com:1094 query config all
```

**Solutions**:

- Increase timeout values in configuration
- Check network connectivity to XROOTD server
- Verify server availability and load

### Permission Errors

**Symptoms**: "Permission denied" or "Access forbidden"

```bash
# Test file access
xrdfs root://server.example.com:1094 stat /path/to/test/file

# Check path existence
xrdfs root://server.example.com:1094 ls /path/to/directory
```

**Solutions**:

- Check XROOTD server access controls
- Verify user authentication credentials
- Ensure proper path permissions on server

### Command Not Found

**Symptoms**: "xrdfs: command not found"

```bash
# Check XROOTD client installation
which xrdfs
which xrdcp

# Verify PATH includes XROOTD binaries
echo $PATH
```

**Solutions**:

- Install XROOTD client tools
- Add XROOTD binaries to PATH
- Confirm client tools are executable

## Deployment and Production Issues

### Service Startup Failures

**Service fails to start**:

```bash
# Check systemd logs
journalctl -u dataharbor-backend -f

# Check configuration file syntax
go run . --config=/path/to/config.yaml --validate

# Check file permissions
ls -la /opt/dataharbor/config/
```

### OIDC Authentication Issues

**OIDC authentication fails**:

- Verify OIDC provider configuration
- Check redirect URIs match exactly
- Validate client credentials
- Check network connectivity to OIDC provider

### XROOTD Production Issues

**XROOTD connection issues**:

```bash
# Test XROOTD connectivity from production server
xrdfs root://server.com:1094 ls /

# Check firewall rules
iptables -L | grep 1094

# Test network connectivity
telnet server.com 1094
```

### SSL Certificate Issues

**SSL certificate problems**:

```bash
# Validate certificate chain
openssl verify -CAfile ca.pem server.crt

# Check certificate expiration
openssl x509 -in server.crt -text -noout | grep -A2 "Validity"

# Verify certificate permissions
ls -la /etc/ssl/certs/dataharbor/
```

### Log Analysis

```bash
# View backend logs
journalctl -u dataharbor-backend -f --since "1 hour ago"

# View nginx logs
tail -f /var/log/nginx/access.log
tail -f /var/log/nginx/error.log

# Check application logs
tail -f /opt/dataharbor/logs/application.log
```

## Authentication Issues

### Session Management

**Session expires immediately**:

- Check system clock synchronization
- Verify JWT token expiration settings
- Check secure cookie settings in HTTPS environments

**Login redirects fail**:

- Verify OIDC provider configuration
- Check redirect URLs in OIDC provider settings
- Review backend logs for authentication errors

**Session persistence issues**:

- Check cookie domain and path settings
- Verify HTTPS requirements for secure cookies
- Check browser cookie storage

## Performance Issues

### Backend Performance

**Slow XROOTD operations**:

- Monitor XROOTD server performance
- Check network latency to XROOTD server
- Review connection pooling configuration
- Implement caching for frequently accessed data

**High memory usage**:

```bash
# Profile memory usage
go build -o dataharbor .
./dataharbor --memprofile=mem.prof

# Analyze memory profile
go tool pprof mem.prof
```

**CPU performance issues**:

```bash
# Profile CPU usage
./dataharbor --cpuprofile=cpu.prof

# Analyze CPU profile
go tool pprof cpu.prof
```

### Frontend Performance

**Slow page loads**:

- Enable browser caching
- Implement lazy loading for large directories
- Optimize bundle size
- Use CDN for static assets

**API response delays**:

- Check backend performance
- Monitor network requests in dev tools
- Implement request caching where appropriate

### System Performance

**Resource optimization**:

- Configure appropriate ulimits
- Tune kernel parameters for network performance
- Monitor system resources (CPU, memory, disk I/O)
- Set up log rotation to prevent disk space issues

## Debugging Tools

### Backend Debugging

**Development debugging**:

```bash
# Run with debug logging
cd app
export DATAHARBOR_LOGGING_LEVEL="debug"
go run .

# Run with race detection
go run -race .

# Enable detailed XROOTD logging
export XRD_LOGLEVEL="Debug"
export XRD_LOGFILE="/tmp/xrd_debug.log"
```

**Production debugging**:

```bash
# Enable debug configuration
export DATAHARBOR_LOGGING_LEVEL="debug"
export DATAHARBOR_SERVER_DEBUG="true"

# Monitor performance
./dataharbor --cpuprofile=cpu.prof --memprofile=mem.prof
```

### Frontend Debugging

**Development debugging**:

```bash
# Run in debug mode
npm run dev:debug

# Enable verbose logging
DEBUG=* npm run dev

# Analyze bundle size
npm run build:analyze
```

**Browser debugging**:

- Open browser developer tools
- Check Console tab for JavaScript errors
- Monitor Network tab for API calls
- Use Vue DevTools browser extension

### XROOTD Debugging

**Connection debugging**:

```bash
# Test server connectivity with debug
XRD_LOGLEVEL=Debug xrdfs root://server.com:1094 ping

# Monitor server performance
xrdfs root://server.com:1094 query stats info

# Check server configuration
xrdfs root://server.com:1094 query config all
```

## Getting Additional Help

### Log Locations

- **Backend**: Console output or configured log file
- **Frontend**: Browser developer console
- **XROOTD**: System logs or XROOTD server logs
- **Container**: `podman logs <container-name>`

### Common Resolution Steps

1. **Check logs** for detailed error messages
2. **Verify configuration** matches examples in documentation
3. **Test components** separately to isolate issues
4. **Create minimal reproduction** case
5. **Check network connectivity** between components
6. **Verify permissions** on files and directories
7. **Test in different environments** to confirm reproducibility

### Documentation References

- **[Setup Guide](./SETUP.md)** - Development environment setup
- **[Development Guide](./DEVELOPMENT.md)** - Development workflow and tools
- **[Backend Configuration](./BACKEND_CONFIGURATION.md)** - Complete backend config reference
- **[Frontend Configuration](./FRONTEND_CONFIGURATION.md)** - Frontend config and deployment
- **[Deployment Guide](./DEPLOYMENT.md)** - Production deployment strategies
- **[XROOTD Integration](./xrootd.md)** - XROOTD server configuration and integration
- **[Architecture Guide](./ARCHITECTURE.md)** - System architecture overview
- **[API Reference](./API.md)** - Complete API documentation
