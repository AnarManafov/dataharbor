# Backend Development Guide

This guide covers backend development for DataHarbor's Go-based REST API server.

## Overview

The backend is a Go web application built with the Gin framework, providing RESTful APIs for file system operations through XROOTD integration, user authentication via OIDC, and secure session management.

## Technology Stack

- **Go 1.24+**: Main programming language
- **Gin**: HTTP web framework
- **Viper**: Configuration management
- **Zap**: Structured logging
- **Gorilla Sessions**: Session management
- **XROOTD Client**: File system operations
- **testify**: Testing framework

## Project Structure

```text
app/
├── main.go                 # Application entry point
├── go.mod                  # Go module definition
├── go.sum                  # Dependency checksums
├── config/                 # Configuration management
│   ├── config.go           # Configuration structures
│   ├── cmd.go              # Command-line argument parsing
│   └── application.*.yaml  # Environment-specific configs
├── controller/             # HTTP request handlers
│   ├── auth.go             # Authentication endpoints
│   ├── fs.go               # File system operations
│   ├── health.go           # Health check endpoint
│   ├── user.go             # User management
│   └── xrd.go              # XROOTD-specific operations
├── middleware/             # HTTP middleware
│   ├── auth_middleware.go  # Authentication middleware
│   ├── cors.go             # CORS handling
│   ├── common_middleware.go # Common middleware utilities
│   └── recovery.go         # Panic recovery
├── route/                  # API route definitions
│   └── routes.go           # Route registration
├── common/                 # Shared utilities
│   ├── logger.go           # Logging configuration
│   ├── sysconf.go          # System configuration
│   └── xrd.go              # XROOTD client wrapper
├── core/                   # Business logic
│   └── sanitation.go       # File cleanup operations
├── request/                # Request DTOs
│   └── fs.go               # File system request structures
├── response/               # Response DTOs
│   ├── response.go         # Common response structures
│   ├── error.go            # Error response handling
│   └── fs.go               # File system response structures
└── util/                   # General utilities
    └── util.go             # Helper functions
```

## Getting Started

### Prerequisites

1. **Go 1.24+** installed
2. **XROOTD client** tools available in PATH
3. **Git** for version control

### Setup

```shell
# Clone and navigate to backend
cd app

# Install dependencies
go mod download
go mod tidy

# Copy configuration template
cd config
copy application.template.yaml application.development.yaml

# Edit configuration as needed
notepad application.development.yaml
```

### Running the Backend

```shell
# Development mode (with hot reload via air if installed)
go run .

# With custom configuration
go run . --config=config/application.development.yaml

# Build and run
go build -o dataharbor-backend .
./dataharbor-backend
```

## Configuration

### Configuration Structure

```go
type Config struct {
    Server ServerConfig `yaml:"server"`
    Auth   AuthConfig   `yaml:"auth"`
    XRD    XRDConfig    `yaml:"xrd"`
    Log    LogConfig    `yaml:"log"`
}

type ServerConfig struct {
    Port    int    `yaml:"port"`
    Host    string `yaml:"host"`
    Debug   bool   `yaml:"debug"`
    Timeout int    `yaml:"timeout"`
}

type AuthConfig struct {
    Enabled bool       `yaml:"enabled"`
    OIDC    OIDCConfig `yaml:"oidc"`
    Session SessionConfig `yaml:"session"`
}
```

### Environment-Specific Configs

Create configuration files for different environments:

- `application.development.yaml` - Development settings
- `application.production.yaml` - Production settings
- `application.testing.yaml` - Test settings

### Configuration Loading

```go
// Initialize command-line flags
config.InitCmd()

// Load configuration
cfg, err := config.LoadConfig(config.ConfigFile)
if err != nil {
    log.Fatal("Failed to load config:", err)
}

// Set as global config
config.SetConfig(cfg)
```

## API Development

### Creating New Endpoints

1. **Define Request/Response DTOs**:

    ```go
    // request/example.go
    type ExampleRequest struct {
        Name        string `json:"name" binding:"required"`
        Description string `json:"description"`
    }

    // response/example.go
    type ExampleResponse struct {
        ID          int    `json:"id"`
        Name        string `json:"name"`
        Description string `json:"description"`
        CreatedAt   string `json:"created_at"`
    }
    ```

1. **Create Controller Handler**:

    ```go
    // controller/example.go
    func (c *Controller) HandleExample(ctx *gin.Context) {
        var req request.ExampleRequest
        
        // Parse and validate request
        if err := ctx.ShouldBindJSON(&req); err != nil {
            response.ErrorResponse(ctx, http.StatusBadRequest, "Invalid request", err)
            return
        }
        
        // Business logic
        result, err := c.processExample(req)
        if err != nil {
            response.ErrorResponse(ctx, http.StatusInternalServerError, "Processing failed", err)
            return
        }
        
        // Success response
        response.SuccessResponse(ctx, result)
    }
    ```

1. **Register Route**:

    ```go
    // route/routes.go
    func RegisterRoutes(router *gin.Engine) {
        api := router.Group("/api/v1")
        
        // Protected routes
        protected := api.Group("/")
        protected.Use(middleware.AuthRequired())
        protected.POST("/example", controller.HandleExample)
    }
    ```

### Request/Response Patterns

#### Standard Response Structure

```go
type Response struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

// Usage
response.SuccessResponse(ctx, data)
response.ErrorResponse(ctx, statusCode, message, err)
```

#### Error Handling

```go
// Standardized error responses
func ErrorResponse(ctx *gin.Context, statusCode int, message string, err error) {
    logger.Error("Request failed", 
        zap.String("path", ctx.Request.URL.Path),
        zap.Error(err),
    )
    
    ctx.JSON(statusCode, Response{
        Code:    statusCode,
        Message: message,
    })
}
```

## XROOTD Integration

### XROOTD Client Wrapper

```go
// common/xrd.go
type XRDClient struct {
    timeout time.Duration
    logger  *zap.Logger
}

func (x *XRDClient) ExecuteCommand(cmd string, args ...string) ([]byte, error) {
    ctx, cancel := context.WithTimeout(context.Background(), x.timeout)
    defer cancel()
    
    command := exec.CommandContext(ctx, cmd, args...)
    return command.Output()
}

func (x *XRDClient) ListDirectory(server, path string) ([]FileInfo, error) {
    output, err := x.ExecuteCommand("xrdfs", server, "ls", "-l", path)
    if err != nil {
        return nil, fmt.Errorf("xrdfs ls failed: %w", err)
    }
    
    return parseXRDOutput(output)
}
```

### File Operations

```go
// controller/fs.go
func (c *Controller) ListDirectory(ctx *gin.Context) {
    var req request.DirectoryRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        response.ErrorResponse(ctx, http.StatusBadRequest, "Invalid request", err)
        return
    }
    
    // Execute XROOTD command
    files, err := c.xrdClient.ListDirectory(c.config.XRD.Server, req.Path)
    if err != nil {
        response.ErrorResponse(ctx, http.StatusInternalServerError, "Directory listing failed", err)
        return
    }
    
    // Format response
    resp := response.DirectoryResponse{
        Items:     files,
        TotalItems: len(files),
        Path:      req.Path,
    }
    
    response.SuccessResponse(ctx, resp)
}
```

## Authentication & Authorization

### OIDC Authentication Flow

```go
// controller/auth.go
func (c *Controller) HandleOIDCCallback(ctx *gin.Context) {
    code := ctx.Query("code")
    state := ctx.Query("state")
    
    // Verify state parameter (CSRF protection)
    if !c.verifyState(ctx, state) {
        response.ErrorResponse(ctx, http.StatusBadRequest, "Invalid state", nil)
        return
    }
    
    // Exchange authorization code for tokens
    tokens, err := c.exchangeCodeForTokens(code)
    if err != nil {
        response.ErrorResponse(ctx, http.StatusInternalServerError, "Token exchange failed", err)
        return
    }
    
    // Store tokens in secure session
    c.storeTokensInSession(ctx, tokens)
    
    // Redirect to original destination
    ctx.Redirect(http.StatusFound, c.getOriginalURL(ctx))
}
```

### Session Management

```go
// middleware/auth_middleware.go
func AuthRequired() gin.HandlerFunc {
    return func(ctx *gin.Context) {
        session := sessions.Default(ctx)
        
        accessToken := session.Get("access_token")
        if accessToken == nil {
            ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
            ctx.Abort()
            return
        }
        
        // Validate token and refresh if needed
        if !isTokenValid(accessToken.(string)) {
            if !refreshToken(ctx, session) {
                ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Token refresh failed"})
                ctx.Abort()
                return
            }
        }
        
        ctx.Next()
    }
}
```

## Middleware Development

### Custom Middleware Pattern

```go
func CustomMiddleware() gin.HandlerFunc {
    return func(ctx *gin.Context) {
        // Pre-processing
        start := time.Now()
        
        // Process request
        ctx.Next()
        
        // Post-processing
        duration := time.Since(start)
        logger.Info("Request processed",
            zap.String("method", ctx.Request.Method),
            zap.String("path", ctx.Request.URL.Path),
            zap.Int("status", ctx.Writer.Status()),
            zap.Duration("duration", duration),
        )
    }
}
```

### Request Logging Middleware

```go
func LoggerMiddleware() gin.HandlerFunc {
    return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
        return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
            param.ClientIP,
            param.TimeStamp.Format(time.RFC1123),
            param.Method,
            param.Path,
            param.Request.Proto,
            param.StatusCode,
            param.Latency,
            param.Request.UserAgent(),
            param.ErrorMessage,
        )
    })
}
```

## Testing

### Unit Testing

```go
// controller/health_test.go
func TestHealthController_HealthCheck(t *testing.T) {
    // Setup
    gin.SetMode(gin.TestMode)
    router := gin.New()
    controller := &HealthController{}
    router.GET("/health", controller.HealthCheck)
    
    // Test
    req, _ := http.NewRequest("GET", "/health", nil)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    // Assertions
    assert.Equal(t, http.StatusOK, w.Code)
    
    var response map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &response)
    assert.Equal(t, "ok", response["data"])
}
```

### Integration Testing

```go
func TestFileSystemIntegration(t *testing.T) {
    // Setup test server
    config := &Config{
        XRD: XRDConfig{
            Server: "test-server.example.com",
            Timeout: 30,
        },
    }
    
    // Create test request
    reqBody := `{"path": "/test/directory", "pageSize": 10}`
    req, _ := http.NewRequest("POST", "/api/v1/dir", strings.NewReader(reqBody))
    req.Header.Set("Content-Type", "application/json")
    
    // Execute request
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    // Verify response
    assert.Equal(t, http.StatusOK, w.Code)
}
```

### Running Tests

```shell
# Run all tests
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific tests
go test -v ./controller -run TestHealthController
```

## Logging

### Structured Logging with Zap

```go
// common/logger.go
var Logger *zap.Logger

func InitLogger() {
    config := zap.NewProductionConfig()
    config.OutputPaths = []string{"stdout", "logs/app.log"}
    
    logger, err := config.Build()
    if err != nil {
        panic(fmt.Sprintf("Failed to initialize logger: %v", err))
    }
    
    Logger = logger
}

// Usage in controllers
Logger.Info("Processing request",
    zap.String("path", req.Path),
    zap.Int("pageSize", req.PageSize),
    zap.String("userID", userID),
)

Logger.Error("Operation failed",
    zap.Error(err),
    zap.String("operation", "listDirectory"),
)
```

### Log Levels and Contexts

- **DEBUG**: Detailed debugging information
- **INFO**: General information about operations
- **WARN**: Warning conditions that should be noted
- **ERROR**: Error conditions that require attention

## Performance Optimization

### Concurrency

```go
func (c *Controller) ProcessMultipleFiles(ctx *gin.Context, files []string) {
    var wg sync.WaitGroup
    results := make(chan FileResult, len(files))
    
    for _, file := range files {
        wg.Add(1)
        go func(filename string) {
            defer wg.Done()
            result := c.processFile(filename)
            results <- result
        }(file)
    }
    
    go func() {
        wg.Wait()
        close(results)
    }()
    
    // Collect results
    var allResults []FileResult
    for result := range results {
        allResults = append(allResults, result)
    }
    
    response.SuccessResponse(ctx, allResults)
}
```

### Connection Pooling

```go
// Configure HTTP client with connection pooling
var httpClient = &http.Client{
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    },
    Timeout: 30 * time.Second,
}
```

## Error Handling

### Error Types

```go
type AppError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
    Err     error  `json:"-"`
}

func (e *AppError) Error() string {
    return e.Message
}

// Common errors
var (
    ErrInvalidPath     = &AppError{Code: 400, Message: "Invalid file path"}
    ErrFileNotFound    = &AppError{Code: 404, Message: "File not found"}
    ErrPermissionDenied = &AppError{Code: 403, Message: "Permission denied"}
    ErrInternalServer  = &AppError{Code: 500, Message: "Internal server error"}
)
```

## Deployment

### Building for Production

```shell
# Build statically linked binary
$env:CGO_ENABLED=0
$env:GOOS="linux"
go build -a -installsuffix cgo -o dataharbor-backend .

# Build for current platform
go build -o dataharbor-backend .
```

### Configuration for Production

```yaml
# application.production.yaml
server:
  port: 8080
  host: "0.0.0.0"
  debug: false
  timeout: 60

log:
  level: "info"
  output: ["stdout", "/var/log/dataharbor/app.log"]

auth:
  enabled: true
  oidc:
    issuer: "https://auth.example.com"
    client_id: "${OIDC_CLIENT_ID}"
    client_secret: "${OIDC_CLIENT_SECRET}"

xrd:
  server: "xrootd.example.com"
  timeout: 120
```

## Monitoring

### Health Checks

```go
func (c *HealthController) HealthCheck(ctx *gin.Context) {
    health := map[string]interface{}{
        "status":    "ok",
        "timestamp": time.Now().UTC(),
        "version":   BuildVersion,
        "uptime":    time.Since(StartTime),
    }
    
    // Check dependencies
    if err := c.checkXRDConnection(); err != nil {
        health["xrd_status"] = "error"
        health["status"] = "degraded"
    } else {
        health["xrd_status"] = "ok"
    }
    
    response.SuccessResponse(ctx, health)
}
```

### Metrics Collection

```go
var (
    requestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "http_request_duration_seconds",
            Help: "HTTP request duration in seconds",
        },
        []string{"method", "path", "status"},
    )
    
    requestCount = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "path", "status"},
    )
)
```

## Best Practices

### Code Organization

1. **Separation of Concerns**: Keep controllers thin, business logic in services
2. **Dependency Injection**: Pass dependencies explicitly
3. **Interface Usage**: Define interfaces for testability
4. **Error Handling**: Use consistent error patterns
5. **Configuration**: Externalize all configuration

### Security Best Practices

1. **Input Validation**: Validate all incoming requests
2. **Path Sanitization**: Prevent directory traversal attacks
3. **Rate Limiting**: Implement request rate limiting
4. **HTTPS Only**: Force HTTPS in production
5. **Secure Headers**: Set appropriate security headers

### Performance Best Practices

1. **Connection Reuse**: Use HTTP connection pooling
2. **Goroutine Management**: Avoid goroutine leaks
3. **Memory Management**: Profile memory usage regularly
4. **Caching**: Cache frequently accessed data
5. **Database Connections**: Use connection pooling if database is added
