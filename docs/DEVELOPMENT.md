# Development Guide

This document covers the development workflow, contribution guidelines, CI/CD processes, and best practices for DataHarbor developers.

## Development Workflow

### Getting Started

1. **Fork and Clone**

   ```bash
   git clone https://github.com/AnarManafov/dataharbor.git
   cd dataharbor
   ```

2. **Install Dependencies**

   ```bash
   # Install all dependencies (uses npm workspaces)
   npm install
   
   # Or install individually
   cd web && npm install && cd ..
   cd app && go mod download && cd ..
   ```

3. **Start Development Environment**

   ```bash
   # Start both frontend and backend with hot reload
   npm run dev
   
   # Or start separately
   npm run dev:frontend  # https://localhost:5173
   npm run dev:backend   # http://localhost:8081
   ```

### Branch Strategy

- **`master`**: Main development branch, always deployable
- **Feature branches**: `feature/description` or `feature/issue-number`
- **Bug fixes**: `fix/description` or `fix/issue-number`
- **Releases**: Tagged with semantic versioning (`v1.2.3`)

### Development Environment Configuration

#### Backend Configuration

Create or modify `app/config/application.development.yaml`:

```yaml
server:
  host: "localhost"
  port: 8081
  debug: true

logging:
  level: "debug"
  format: "console"

auth:
  oidc:
    # Use development OIDC settings
    issuer: "https://dev-keycloak.gsi.de/realms/dataharbor"
    client_id: "dataharbor-dev"

xrootd:
  servers:
    - "root://dev-xrootd.gsi.de:1094"
  timeout: "30s"
```

#### Frontend Configuration

Set environment variables or create `.env.local`:

```bash
# Optional: Custom backend URL
VITE_API_BASE_URL=http://localhost:8081/api/v1

# SSL Certificate paths (for HTTPS development)
VITE_SSL_KEY=/path/to/server.key
VITE_SSL_CERT=/path/to/server.crt
```

### Code Style and Standards

#### Go Backend Standards

- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` and `goimports` for formatting
- Run linting with `golangci-lint`
- Maintain test coverage > 80%

#### Vue.js Frontend Standards

- Follow [Vue.js Style Guide](https://vuejs.org/style-guide/)
- Use ESLint and Prettier for code formatting
- Follow TypeScript best practices where applicable
- Use composition API for new components

#### Commit Message Format

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```text
type(scope): description

[optional body]

[optional footer]
```

Examples:

```text
feat(backend): add file staging endpoint
fix(frontend): resolve authentication redirect loop
docs(readme): update installation instructions
```

### Testing Requirements

#### Before Submitting PR

```bash
# Run all backend tests
cd app && go test -v ./...

# Run frontend tests (when available)
cd web && npm test

# Check code coverage
cd app && go test -cover ./...

# Run linting
cd app && golangci-lint run
cd web && npm run lint
```

#### Test Coverage Requirements

- **Backend**: Minimum 80% overall coverage
- **Critical paths** (auth, file operations): 90% coverage
- **New features**: Must include tests
- **Bug fixes**: Must include regression tests## Release Management

### Versioning Strategy

DataHarbor follows [Semantic Versioning](https://semver.org/) with the following structure:

- **Global Versions**: `vX.Y.Z` (e.g., `v1.0.0`) for complete releases
- **Component Versions**: Automatically generated from global versions
  - Backend: `app/vX.Y.Z`
  - Frontend: `web/vX.Y.Z`

### Creating a Release

DataHarbor uses a **pre-release trigger** approach to ensure consistent repository state and proper documentation updates.

1. **Prepare Release**

   ```bash
   # Ensure all changes are committed and pushed
   git checkout master
   git pull origin master
   
   # Run tests and build to verify everything works
   npm run test
   npm run build
   ```

2. **Create Pre-Release Trigger Tag**

   Instead of creating the final release tag directly, create a **trigger tag** that initiates the release preparation process:

   ```bash
   # For regular releases
   git tag -a release-v1.2.3 -m "Prepare release v1.2.3
   
   Features:
   - Added file staging improvements
   - Enhanced authentication security
   
   Bug Fixes:
   - Fixed directory navigation issue
   - Resolved authentication timeout"
   
   # For hotfix releases
   git tag -a hotfix-v1.2.4 -m "Prepare hotfix v1.2.4"
   
   # For pre-releases
   git tag -a prerelease-v1.3.0-beta.1 -m "Prepare pre-release v1.3.0-beta.1"
   
   # Push trigger tag to start the automated release process
   git push origin release-v1.2.3
   ```

3. **Automated Release Process**

   The CI/CD pipeline automatically:
   - **Updates all version files** (package.json, web/package.json)
   - **Generates and updates CHANGELOG.md** with commit history
   - **Creates RELEASE_NOTES.md** with release-specific notes
   - **Commits all changes** to master branch
   - **Creates the actual release tag** (`v1.2.3`) pointing to the prepared commit
   - **Creates component tags** (`app/v1.2.3`, `web/v1.2.3`)
   - **Builds and packages components**
   - **Publishes GitHub release** with artifacts and changelog

#### Release Tag Types

- **`release-v1.2.3`** → Creates final release `v1.2.3`
- **`hotfix-v1.2.4`** → Creates hotfix release `v1.2.4`  
- **`prerelease-v1.3.0-beta.1`** → Creates pre-release `v1.3.0-beta.1`

#### Why Pre-Release Triggers?

This approach ensures:

- ✅ **Consistent State**: Release tags always point to commits with updated changelog and versions
- ✅ **Automated Documentation**: CHANGELOG.md and version files are automatically maintained
- ✅ **No Manual Steps**: No need to manually update package.json or changelog files
- ✅ **Rollback Safety**: Failed preparation doesn't create invalid release tags
- ✅ **Clear Audit Trail**: Separate commits for preparation vs. development changes

### CI/CD Workflows

#### Main Workflows

1. **Backend CI** (`.github/workflows/backend.yml`)
   - Triggers on changes to `app/**` files
   - Runs tests, linting, coverage reporting
   - Builds RPM packages for deployment

2. **Frontend CI** (`.github/workflows/frontend.yml`)
   - Triggers on changes to `web/**` files
   - Runs build, security scanning
   - Creates deployable artifacts

3. **Release Automation** (`.github/workflows/version-tag-processor.yml`)
   - Triggers on version tag pushes
   - Manages versioning across components
   - Generates changelogs and release notes

#### Workflow Dependencies

```text
Trigger Tag Push (release-vX.Y.Z)
    ↓
version-tag-processor.yml
    ├─ Update package versions
    ├─ Generate changelog & release notes
    ├─ Commit all changes
    ├─ Create actual release tag (vX.Y.Z)
    └─ Create component tags (app/vX.Y.Z, web/vX.Y.Z)
    ↓
publish-release.yml (triggered by vX.Y.Z tag)
    ├─ Build frontend & backend
    ├─ Create RPM packages
    └─ Publish GitHub release with artifacts
```

## Development Best Practices

### Code Organization

#### Backend Structure

```text
app/
├── controller/         # HTTP request handlers
├── middleware/         # Cross-cutting concerns
├── route/             # API route definitions
├── config/            # Configuration management
├── common/            # Shared utilities
├── core/              # Business logic
├── request/           # Request DTOs
└── response/          # Response DTOs
```

#### Frontend Structure

```text
web/src/
├── components/        # Reusable UI components
├── views/             # Page-level components
├── composables/       # Vue 3 composition functions
├── stores/            # Pinia state management
├── router/            # Vue Router configuration
├── api/               # HTTP client and endpoints
└── utils/             # Helper functions
```

### Error Handling

#### Backend Error Handling

```go
// Use consistent error response format
func HandleError(c *gin.Context, err error, code int) {
    logger.Error("Request failed", zap.Error(err))
    
    c.JSON(code, response.ErrorResponse{
        Code:    code,
        Message: err.Error(),
    })
}

// Example usage
func FileHandler(c *gin.Context) {
    files, err := xrdClient.ListDirectory(path)
    if err != nil {
        HandleError(c, err, http.StatusInternalServerError)
        return
    }
    
    c.JSON(http.StatusOK, response.SuccessResponse{
        Code:    http.StatusOK,
        Message: "success",
        Data:    files,
    })
}
```

#### Frontend Error Handling

```javascript
// Use consistent error handling in composables
export function useFileOperations() {
  const error = ref(null)
  const loading = ref(false)
  
  const listDirectory = async (path) => {
    try {
      loading.value = true
      error.value = null
      
      const response = await api.get(`/dir`, { params: { path } })
      return response.data
    } catch (err) {
      error.value = err.response?.data?.message || 'An error occurred'
      throw err
    } finally {
      loading.value = false
    }
  }
  
  return { listDirectory, error, loading }
}
```

### Security Considerations

#### Input Validation

```go
// Backend: Always validate and sanitize inputs
func validatePath(path string) error {
    if strings.Contains(path, "..") {
        return errors.New("path traversal not allowed")
    }
    
    if !filepath.IsAbs(path) {
        return errors.New("absolute path required")
    }
    
    return nil
}
```

#### Authentication Middleware

```go
// Verify authentication on protected routes
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        session := sessions.Default(c)
        
        if session.Get("access_token") == nil {
            c.JSON(http.StatusUnauthorized, gin.H{
                "code": http.StatusUnauthorized,
                "message": "authentication required",
            })
            c.Abort()
            return
        }
        
        c.Next()
    }
}
```

### Performance Optimization

#### Backend Performance

- Use connection pooling for XROOTD operations
- Implement request timeouts
- Add response caching where appropriate
- Monitor memory usage with file operations
- Use structured logging for debugging

#### Frontend Performance

- Implement lazy loading for large directory listings
- Use virtual scrolling for file lists
- Optimize bundle size with tree shaking
- Implement service worker for caching
- Use appropriate loading states

### Configuration Management

#### Environment-Specific Configuration

```yaml
# app/config/application.development.yaml
server:
  debug: true
  port: 8081

logging:
  level: debug
  format: console

# app/config/application.production.yaml
server:
  debug: false
  port: 8081

logging:
  level: info
  format: json
```

#### Configuration Validation

```go
// Validate configuration on startup
func ValidateConfig(cfg *Config) error {
    if cfg.Auth.OIDC.ClientID == "" {
        return errors.New("OIDC client ID required")
    }
    
    if len(cfg.XROOTD.Servers) == 0 {
        return errors.New("at least one XROOTD server required")
    }
    
    return nil
}
```

## Troubleshooting

### Common Development Issues

#### Backend Issues

1. **XROOTD Connection Failures**

   ```bash
   # Test XROOTD connectivity
   xrdfs root://server.gsi.de:1094 ls /

   # Check server configuration
   cat app/config/application.development.yaml
   ```

2. **Authentication Problems**

   ```bash
   # Verify OIDC configuration
   curl -s https://keycloak.gsi.de/realms/dataharbor/.well-known/openid_configuration

   # Check session cookies
   # Use browser dev tools -> Application -> Cookies
   ```

#### Frontend Issues

1. **SSL Certificate Problems**

   ```bash
   # Check certificate status
   npm run cert:check
   
   # Setup development certificates
   npm run cert:setup
   ```

2. **API Connection Issues**

   ```bash
   # Verify backend is running
   curl http://localhost:8081/api/v1/health
   
   # Check CORS configuration
   # Look for CORS errors in browser console
   ```

### Debugging Tools

#### Backend Debugging

```bash
# Run with debug logging
cd app
CONFIG_FILE_PATH=config/application.development.yaml go run .

# Run with race detection
go run -race .

# Profile memory usage
go build -o dataharbor .
./dataharbor --cpuprofile=cpu.prof --memprofile=mem.prof
```

#### Frontend Debugging

```bash
# Run in debug mode
npm run dev:debug

# Analyze bundle size
npm run build:analyze

# Run with verbose logging
DEBUG=* npm run dev
```

## Contributing Guidelines

### Pull Request Process

1. **Create Feature Branch**

   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make Changes**
   - Follow coding standards
   - Add appropriate tests
   - Update documentation

3. **Test Changes**

   ```bash
   # Run all tests
   cd app && go test -v ./...
   cd web && npm test
   
   # Check code coverage
   cd app && go test -cover ./...
   ```

4. **Submit Pull Request**
   - Use descriptive title and description
   - Reference related issues
   - Ensure CI checks pass

### Code Review Checklist

- [ ] Code follows project standards
- [ ] Tests are included and passing
- [ ] Documentation is updated
- [ ] No sensitive information exposed
- [ ] Error handling is appropriate
- [ ] Performance impact considered

### Internal Development Guidelines

Since DataHarbor is for internal GSI use:

- Focus on developer experience and maintainability
- Document integration points with GSI infrastructure
- Consider security requirements for internal networks
- Plan for integration with existing GSI authentication systems
- Design for internal deployment and monitoring tools
