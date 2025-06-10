# XROOTD Integration

This document describes how DataHarbor integrates with XROOTD for file system operations and provides guidance for XROOTD setup and configuration.

## Overview

XROOTD (eXtended ROOT daemon) is a high-performance, scalable data access system primarily used in high-energy physics environments. DataHarbor integrates with XROOTD through command-line client tools to provide file system operations such as directory listing, file staging, and metadata retrieval.

## XROOTD Architecture

### Components

- **XROOTD Server**: The main data server that hosts files
- **XROOTD Client**: Command-line tools for accessing XROOTD servers
- **DataHarbor Backend**: Interfaces with XROOTD client tools

### Integration Pattern

```
┌─────────────────┐    Command Line    ┌─────────────────┐    Network     ┌─────────────────┐
│                 │◄──────────────────►│                 │◄──────────────►│                 │
│ DataHarbor      │     (xrdfs, etc.)  │ XROOTD Client   │   (XRD Proto)  │ XROOTD Server   │
│ Backend (Go)    │                    │ Tools           │                │ (File System)   │
│                 │                    │                 │                │                 │
└─────────────────┘                    └─────────────────┘                └─────────────────┘
```

## XROOTD Client Installation

### macOS Installation

```bash
# Using Homebrew
brew install xrootd
```

### Linux Installation

#### Ubuntu/Debian
```bash
sudo apt update
sudo apt install xrootd-client
```

#### CentOS/RHEL/Fedora
```bash
# Using dnf (Fedora)
sudo dnf install xrootd-client

# Using yum (CentOS/RHEL)
sudo yum install xrootd-client
```

### Windows Installation

For Windows development, consider using:
- WSL (Windows Subsystem for Linux) with Linux installation
- Docker containers with XROOTD client tools
- Cross-compilation from Linux systems

### Verification

Verify installation:
```bash
xrdfs --help
xrdcp --help
which xrdfs
```

## DataHarbor XROOTD Integration

### XROOTD Client Wrapper

The DataHarbor backend includes a Go wrapper for XROOTD client operations:

```go
// common/xrd.go
type XRDClient struct {
    server  string
    timeout time.Duration
    logger  *zap.Logger
}

func NewXRDClient(server string, timeout time.Duration) *XRDClient {
    return &XRDClient{
        server:  server,
        timeout: timeout,
        logger:  common.Logger,
    }
}

func (x *XRDClient) ExecuteCommand(cmd string, args ...string) ([]byte, error) {
    ctx, cancel := context.WithTimeout(context.Background(), x.timeout)
    defer cancel()
    
    // Build command with server prefix if needed
    fullArgs := make([]string, 0, len(args)+1)
    if len(args) > 0 && !strings.Contains(args[0], x.server) {
        fullArgs = append(fullArgs, x.server)
    }
    fullArgs = append(fullArgs, args...)
    
    command := exec.CommandContext(ctx, cmd, fullArgs...)
    
    x.logger.Debug("Executing XROOTD command",
        zap.String("cmd", cmd),
        zap.Strings("args", fullArgs),
    )
    
    output, err := command.Output()
    if err != nil {
        x.logger.Error("XROOTD command failed",
            zap.String("cmd", cmd),
            zap.Error(err),
        )
        return nil, fmt.Errorf("xrootd command failed: %w", err)
    }
    
    return output, nil
}
```

### Directory Listing Implementation

```go
func (x *XRDClient) ListDirectory(path string) ([]FileInfo, error) {
    // Execute: xrdfs <server> ls -l <path>
    output, err := x.ExecuteCommand("xrdfs", "ls", "-l", path)
    if err != nil {
        return nil, fmt.Errorf("directory listing failed: %w", err)
    }
    
    return x.parseDirectoryOutput(output)
}

func (x *XRDClient) parseDirectoryOutput(output []byte) ([]FileInfo, error) {
    lines := strings.Split(string(output), "\n")
    var files []FileInfo
    
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line == "" {
            continue
        }
        
        // Parse xrdfs ls -l output format:
        // -rw-r--r--   1 user group      1024 Oct 15 10:30 filename.txt
        // drwxr-xr-x   1 user group         0 Oct 14 15:20 directory
        
        parts := strings.Fields(line)
        if len(parts) < 9 {
            continue // Skip malformed lines
        }
        
        permissions := parts[0]
        sizeStr := parts[4]
        
        // Parse size
        size, err := strconv.ParseInt(sizeStr, 10, 64)
        if err != nil {
            size = 0
        }
        
        // Determine file type
        fileType := "file"
        if strings.HasPrefix(permissions, "d") {
            fileType = "dir"
            size = 0 // Directories don't have meaningful sizes
        }
        
        // Parse date and time (parts[5], parts[6], parts[7])
        dateTime := fmt.Sprintf("%s %s %s", parts[5], parts[6], parts[7])
        
        // File name (rest of the parts)
        fileName := strings.Join(parts[8:], " ")
        
        files = append(files, FileInfo{
            Name:     fileName,
            Type:     fileType,
            Size:     size,
            DateTime: dateTime,
        })
    }
    
    return files, nil
}
```

### File Staging Implementation

```go
func (x *XRDClient) StageFile(remotePath, localStagingDir string) (string, error) {
    // Generate unique staging directory
    stagingID := fmt.Sprintf("stg_%d", time.Now().Unix())
    stagingPath := filepath.Join(localStagingDir, stagingID)
    
    // Create staging directory
    if err := os.MkdirAll(stagingPath, 0755); err != nil {
        return "", fmt.Errorf("failed to create staging directory: %w", err)
    }
    
    // Extract filename from remote path
    fileName := filepath.Base(remotePath)
    localFilePath := filepath.Join(stagingPath, fileName)
    
    // Build full remote URL
    remoteURL := fmt.Sprintf("%s/%s", x.server, strings.TrimPrefix(remotePath, "/"))
    
    // Execute: xrdcp <remote_url> <local_path>
    _, err := x.ExecuteCommand("xrdcp", remoteURL, localFilePath)
    if err != nil {
        // Clean up staging directory on failure
        os.RemoveAll(stagingPath)
        return "", fmt.Errorf("file staging failed: %w", err)
    }
    
    x.logger.Info("File staged successfully",
        zap.String("remote_path", remotePath),
        zap.String("local_path", localFilePath),
    )
    
    return localFilePath, nil
}
```

### Server Information Retrieval

```go
func (x *XRDClient) GetServerInfo() (*ServerInfo, error) {
    // Execute: xrdfs <server> query config version
    output, err := x.ExecuteCommand("xrdfs", "query", "config", "version")
    if err != nil {
        return nil, fmt.Errorf("server info query failed: %w", err)
    }
    
    version := strings.TrimSpace(string(output))
    
    // Parse server hostname from server URL
    hostname := x.server
    if strings.Contains(hostname, "://") {
        parts := strings.Split(hostname, "://")
        if len(parts) > 1 {
            hostname = strings.Split(parts[1], ":")[0]
        }
    }
    
    return &ServerInfo{
        Hostname: hostname,
        Version:  version,
        Server:   x.server,
    }, nil
}
```

## Configuration

### Backend Configuration

```yaml
# app/config/application.yaml
xrd:
  server: "root://xrootd.example.com:1094"
  timeout: 60  # seconds
  initial_dir: "/store/data"
  staging:
    directory: "/tmp/dataharbor/staged"
    cleanup_interval: 3600  # seconds (1 hour)
    max_file_size: 1073741824  # bytes (1GB)
    max_concurrent_stages: 10
```

### Go Configuration Structure

```go
type XRDConfig struct {
    Server   string        `yaml:"server"`
    Timeout  int          `yaml:"timeout"`
    InitialDir string     `yaml:"initial_dir"`
    Staging  StagingConfig `yaml:"staging"`
}

type StagingConfig struct {
    Directory       string `yaml:"directory"`
    CleanupInterval int    `yaml:"cleanup_interval"`
    MaxFileSize     int64  `yaml:"max_file_size"`
    MaxConcurrentStages int `yaml:"max_concurrent_stages"`
}
```

## Command Reference

### Common XROOTD Client Commands

#### Directory Operations
```bash
# List directory contents
xrdfs root://server.example.com:1094 ls /path/to/directory

# List with detailed information
xrdfs root://server.example.com:1094 ls -l /path/to/directory

# List recursively
xrdfs root://server.example.com:1094 ls -R /path/to/directory
```

#### File Operations
```bash
# Copy file from XROOTD to local
xrdcp root://server.example.com:1094//path/to/file.txt /local/path/file.txt

# Copy file from local to XROOTD
xrdcp /local/path/file.txt root://server.example.com:1094//path/to/file.txt

# Get file information
xrdfs root://server.example.com:1094 stat /path/to/file.txt
```

#### Server Information
```bash
# Query server configuration
xrdfs root://server.example.com:1094 query config version

# Check server status
xrdfs root://server.example.com:1094 query stats info

# Get server hostname
xrdfs root://server.example.com:1094 query config hostname
```

### URL Format

XROOTD URLs follow this format:
```
root://hostname:port//absolute/path/to/file
```

Examples:
```
root://xrootd.example.com:1094//store/data/file.txt
root://192.168.1.100:1094//tmp/test.dat
```

## Error Handling

### Common XROOTD Errors

```go
func (x *XRDClient) handleXRDError(err error) error {
    errStr := err.Error()
    
    switch {
    case strings.Contains(errStr, "No such file or directory"):
        return &XRDError{
            Code:    404,
            Type:    "FileNotFound",
            Message: "File or directory not found",
            Detail:  errStr,
        }
    case strings.Contains(errStr, "Permission denied"):
        return &XRDError{
            Code:    403,
            Type:    "PermissionDenied",
            Message: "Access denied",
            Detail:  errStr,
        }
    case strings.Contains(errStr, "Connection timeout"):
        return &XRDError{
            Code:    504,
            Type:    "Timeout",
            Message: "Connection timeout",
            Detail:  errStr,
        }
    case strings.Contains(errStr, "Server not responding"):
        return &XRDError{
            Code:    503,
            Type:    "ServerUnavailable",
            Message: "XROOTD server unavailable",
            Detail:  errStr,
        }
    default:
        return &XRDError{
            Code:    500,
            Type:    "InternalError",
            Message: "XROOTD operation failed",
            Detail:  errStr,
        }
    }
}

type XRDError struct {
    Code    int    `json:"code"`
    Type    string `json:"type"`
    Message string `json:"message"`
    Detail  string `json:"detail"`
}

func (e *XRDError) Error() string {
    return fmt.Sprintf("%s: %s", e.Type, e.Message)
}
```

## File Sanitation Service

### Automatic Cleanup

```go
// core/sanitation.go
type SanitationService struct {
    stagingDir      string
    cleanupInterval time.Duration
    maxAge          time.Duration
    logger          *zap.Logger
}

func (s *SanitationService) Start() {
    ticker := time.NewTicker(s.cleanupInterval)
    
    go func() {
        for {
            select {
            case <-ticker.C:
                s.cleanupStagedFiles()
            }
        }
    }()
}

func (s *SanitationService) cleanupStagedFiles() {
    s.logger.Info("Starting staged file cleanup")
    
    err := filepath.Walk(s.stagingDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        
        // Skip if it's the staging directory itself
        if path == s.stagingDir {
            return nil
        }
        
        // Check if file/directory is older than maxAge
        if time.Since(info.ModTime()) > s.maxAge {
            s.logger.Info("Removing old staged file",
                zap.String("path", path),
                zap.Time("modified", info.ModTime()),
            )
            
            if info.IsDir() {
                return os.RemoveAll(path)
            } else {
                return os.Remove(path)
            }
        }
        
        return nil
    })
    
    if err != nil {
        s.logger.Error("File cleanup failed", zap.Error(err))
    } else {
        s.logger.Info("File cleanup completed")
    }
}
```

## Performance Optimization

### Connection Pooling

```go
type XRDConnectionPool struct {
    servers []string
    current int
    mutex   sync.RWMutex
    clients map[string]*XRDClient
}

func (p *XRDConnectionPool) GetClient() *XRDClient {
    p.mutex.RLock()
    defer p.mutex.RUnlock()
    
    server := p.servers[p.current]
    p.current = (p.current + 1) % len(p.servers)
    
    if client, exists := p.clients[server]; exists {
        return client
    }
    
    // Create new client if not exists
    client := NewXRDClient(server, 60*time.Second)
    p.clients[server] = client
    return client
}
```

### Concurrent Operations

```go
func (x *XRDClient) ListDirectoryConcurrent(paths []string) (map[string][]FileInfo, error) {
    var wg sync.WaitGroup
    results := make(map[string][]FileInfo)
    errors := make(map[string]error)
    mutex := sync.RWMutex{}
    
    for _, path := range paths {
        wg.Add(1)
        go func(p string) {
            defer wg.Done()
            
            files, err := x.ListDirectory(p)
            
            mutex.Lock()
            if err != nil {
                errors[p] = err
            } else {
                results[p] = files
            }
            mutex.Unlock()
        }(path)
    }
    
    wg.Wait()
    
    if len(errors) > 0 {
        return results, fmt.Errorf("some operations failed: %v", errors)
    }
    
    return results, nil
}
```

## Troubleshooting

### Common Issues

1. **Connection Timeouts**
   - Increase timeout values in configuration
   - Check network connectivity to XROOTD server
   - Verify server availability

2. **Permission Errors**
   - Check XROOTD server access controls
   - Verify user authentication
   - Ensure proper path permissions

3. **Command Not Found**
   - Verify XROOTD client installation
   - Check PATH environment variable
   - Confirm client tools are executable

4. **File Staging Failures**
   - Check disk space in staging directory
   - Verify write permissions
   - Monitor staging directory cleanup

### Debugging Commands

```bash
# Test server connectivity
xrdfs root://server.example.com:1094 ping

# Check server configuration
xrdfs root://server.example.com:1094 query config all

# Test file access
xrdfs root://server.example.com:1094 stat /path/to/test/file

# Monitor server performance
xrdfs root://server.example.com:1094 query stats info
```

### Logging Configuration

```go
// Enable detailed XROOTD logging
func (x *XRDClient) EnableDebugLogging() {
    x.logger = x.logger.With(zap.String("component", "xrd_client"))
    
    // Set environment variables for XROOTD client debugging
    os.Setenv("XRD_LOGLEVEL", "Debug")
    os.Setenv("XRD_LOGFILE", "/tmp/xrd_debug.log")
}
```

## Best Practices

### Performance
- Use connection pooling for multiple operations
- Implement proper timeout handling
- Cache directory listings when appropriate
- Use concurrent operations for bulk tasks

### Security
- Validate all file paths to prevent directory traversal
- Implement proper authentication with XROOTD server
- Use secure staging directories with proper permissions
- Regularly clean up staged files

### Reliability
- Implement retry logic for transient failures
- Monitor XROOTD server availability
- Use health checks to detect server issues
- Log all operations for debugging

### Monitoring
- Track operation success/failure rates
- Monitor response times
- Alert on server unavailability
- Log security-relevant events

## References

- [XROOTD Official Documentation](https://xrootd.slac.stanford.edu)
- [XROOTD Client API](https://xrootd.slac.stanford.edu/doc/doxygen/current/html/classXrdCl_1_1FileSystem.html)
- [XROOTD Configuration Guide](https://xrootd.slac.stanford.edu/doc/prod/cms_config.htm)
