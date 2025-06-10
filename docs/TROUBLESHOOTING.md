# Troubleshooting Guide

This document covers common issues and their solutions when working with DataHarbor.

## Authentication Issues

### OIDC Login Fails

**Symptoms**: Login redirects to error page or returns 401

**Solutions**:
1. Check OIDC provider configuration in `app/config/application.yaml`
2. Verify redirect URLs are correctly configured in your OIDC provider
3. Check network connectivity to OIDC provider
4. Review backend logs for authentication errors

```shell
# Check backend logs
cd app
go run . | Select-String "auth"
```

### Session Expires Immediately

**Symptoms**: User gets logged out right after login

**Solutions**:
1. Check system clock synchronization
2. Verify JWT token expiration settings
3. Check for secure cookie settings in HTTPS environments

## XROOTD Connection Issues

### "XRD client not found" Error

**Symptoms**: Backend fails to start or file operations fail

**Solutions**:
1. Install XROOTD client tools
2. Verify `xrdcp` and `xrdfs` are in system PATH
3. Test XROOTD connection manually:

```shell
# Test XROOTD connectivity
xrdfs your-xrootd-server ls /
```

### File Operations Timeout

**Symptoms**: File listing or staging operations hang or timeout

**Solutions**:
1. Check XROOTD server connectivity
2. Increase timeout values in configuration
3. Check network firewall settings
4. Verify XROOTD server permissions

## Frontend Issues

### Development Server Won't Start

**Symptoms**: `npm run dev` fails or shows SSL errors

**Solutions**:
1. Check Node.js version (requires 18+)
2. Clear npm cache: `npm cache clean --force`
3. Remove node_modules and reinstall:

```shell
Remove-Item -Recurse -Force node_modules
Remove-Item package-lock.json
npm install
```

4. For SSL issues, check certificate configuration:

```shell
npm run cert:check
```

### API Calls Fail with CORS Errors

**Symptoms**: Browser console shows CORS policy errors

**Solutions**:
1. Verify backend CORS configuration allows frontend origin
2. Check that frontend is accessing correct backend URL
3. Ensure both frontend and backend use same protocol (HTTP/HTTPS)

## File System Issues

### Directory Listing Returns Empty

**Symptoms**: File explorer shows no files/folders

**Solutions**:
1. Check XROOTD server permissions for the path
2. Verify path exists on XROOTD server
3. Check backend logs for permission errors
4. Test path manually with XROOTD client

### File Staging Fails

**Symptoms**: File download preparation fails

**Solutions**:
1. Check disk space on staging directory
2. Verify write permissions on staging directory
3. Check file size limits in configuration
4. Review staging cleanup configuration

## Performance Issues

### Slow Directory Listing

**Solutions**:
1. Reduce page size in directory requests
2. Check XROOTD server performance
3. Consider caching for frequently accessed directories
4. Monitor network latency to XROOTD server

### High Memory Usage

**Solutions**:
1. Check for memory leaks in file operations
2. Reduce concurrent file operations
3. Monitor staging directory cleanup
4. Check XROOTD client process cleanup

## Configuration Issues

### Backend Won't Start

**Symptoms**: Backend fails to start with configuration errors

**Solutions**:
1. Validate YAML syntax in configuration files
2. Check file permissions on configuration files
3. Verify all required configuration values are set
4. Check port availability

```shell
# Check if port is in use
netstat -an | Select-String ":8081"
```

### Environment Variables Not Working

**Solutions**:
1. Verify environment variable names match configuration
2. Check variable scope (user vs system)
3. Restart terminal/IDE after setting variables
4. Use configuration files instead of environment variables

## Container Issues

### Podman/Docker Build Fails

**Solutions**:
1. Check Containerfile syntax
2. Verify base image availability
3. Check network connectivity for package downloads
4. Clear container cache

```shell
# Clear Podman cache
podman system prune -a
```

### Container Runtime Issues

**Solutions**:
1. Check container logs:

```shell
podman logs container-name
```

2. Verify port mapping and network configuration
3. Check volume mounts and permissions
4. Ensure container has access to required dependencies

## Logging and Debugging

### Enable Debug Logging

**Backend**:
```yaml
# In application.yaml
log:
  level: debug
```

**Frontend**:
```javascript
// In browser console
localStorage.setItem('debug', 'dataharbor:*')
```

### Check System Resources

```shell
# Check disk space
Get-WmiObject -Class Win32_LogicalDisk | Select-Object Size, FreeSpace

# Check running processes
Get-Process | Where-Object {$_.ProcessName -like "*dataharbor*"}

# Check network connectivity
Test-NetConnection -ComputerName your-xrootd-server -Port 1094
```

## Getting Help

If you're still experiencing issues:

1. Check the logs for detailed error messages
2. Verify your configuration matches the examples in `/docs/SETUP.md`
3. Test individual components separately
4. Create a minimal reproduction case
5. Check if the issue is reproducible across different environments

### Log Locations

- **Backend**: Console output or configured log file
- **Frontend**: Browser developer console
- **XROOTD**: System logs or XROOTD server logs
- **Container**: `podman logs <container-name>`
