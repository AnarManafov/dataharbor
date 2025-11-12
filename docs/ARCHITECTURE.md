# System Architecture

[← Back to Documentation](./README.md)

This document describes the overall architecture, design patterns, and technical decisions for DataHarbor.

## Overview

DataHarbor follows a modern web application architecture with a clear separation between frontend and backend components, implementing security best practices and scalable design patterns.

## Architecture Diagrams

### System Overview

```mermaid
graph TB
    User[👤 User] --> Browser[🌐 Browser]
    Browser --> Frontend[📱 Vue.js SPA<br/>HTTPS:5173]
    Frontend --> Backend[⚙️ Go Backend<br/>HTTPS:8081]
    Backend --> XROOTD[📁 XROOTD Server<br/>XRD:1094]
    Backend --> OIDC[🔐 OIDC Provider<br/>Keycloak]
    
    subgraph "Client Layer"
        User
        Browser
    end
    
    subgraph "Application Layer"
        Frontend
        Backend
    end
    
    subgraph "Infrastructure Layer"
        XROOTD
        OIDC
    end
    
    classDef client fill:#e1f5fe
    classDef app fill:#f3e5f5
    classDef infra fill:#e8f5e8
    
    class User,Browser client
    class Frontend,Backend app
    class XROOTD,OIDC infra
```

### Detailed Component Architecture

```mermaid
graph TB
    subgraph "Frontend (Vue.js SPA)"
        VueApp[Vue 3 App] --> VueRouter[Vue Router]
        VueApp --> PiniaStore[Pinia Stores]
        VueApp --> Components[UI Components]
        
        PiniaStore --> AuthStore[Auth Store]
        PiniaStore --> FilesStore[Files Store]
        PiniaStore --> UIStore[UI Store]
        
        Components --> TopBar[TopBar.vue]
        Components --> Sidebar[GlobalSidebar.vue]
        Components --> BrowseView[BrowseXrdView.vue]
        Components --> LoginView[LoginView.vue]
    end
    
    subgraph "Backend (Go REST API)"
        GinEngine[Gin HTTP Engine] --> Middleware[Middleware Pipeline]
        GinEngine --> Controllers[Controllers]
        GinEngine --> Routes[Route Handlers]
        
        Middleware --> Recovery[Recovery MW]
        Middleware --> Logger[Logger MW]
        Middleware --> CORS[CORS MW]
        Middleware --> Auth[Auth MW]
        
        Controllers --> AuthController[Auth Controller]
        Controllers --> XRDController[XRD Controller]
        Controllers --> HealthController[Health Controller]
        
        XRDController --> XRDClient[XRD Client]
    end
    
    subgraph "External Services"
        XRDServer[XROOTD Server<br/>File Storage]
        OIDCProvider[OIDC Provider<br/>Authentication]
    end
    
    VueApp -->|HTTPS API Calls| GinEngine
    XRDClient -->|XRD Protocol| XRDServer
    AuthController -->|OIDC Flow| OIDCProvider
    
    classDef frontend fill:#e3f2fd
    classDef backend fill:#fff3e0
    classDef external fill:#f1f8e9
    
    class VueApp,VueRouter,PiniaStore,Components,AuthStore,FilesStore,UIStore,TopBar,Sidebar,BrowseView,LoginView frontend
    class GinEngine,Middleware,Controllers,Routes,Recovery,Logger,CORS,Auth,AuthController,XRDController,HealthController,XRDClient backend
    class XRDServer,OIDCProvider external
```

## Components

### Frontend (Vue.js SPA)

**Technology Stack:** Vue 3, Vite, Element Plus, Pinia, Vue Router, Axios

> For detailed frontend technology information, see **[Frontend Development](./FRONTEND.md)**.

**Key Features:**

- Single Page Application (SPA) architecture
- Responsive design with modern UI components
- Client-side routing for seamless navigation
- State management for user session and application data
- HTTPS-only communication with backend

**Key Directories:** `web/src/components/`, `web/src/views/`, `web/src/store/`, `web/src/api/`

> For complete directory structure, see **[Frontend Development → Project Structure](./FRONTEND.md#project-structure)**.

### Backend (Go REST API)

**Technology Stack:** Go 1.24+, Gin, Viper, Zap, Gorilla Sessions, Go XROOTD Client

> For detailed backend technology information, see **[Backend Development](./BACKEND.md)**.

**Key Features:**

- RESTful API design
- Middleware-based architecture
- Structured logging and monitoring
- Configuration-driven deployment
- Asynchronous file operations with timeouts
- Session-based authentication

**Key Directories:** `app/controller/`, `app/middleware/`, `app/config/`, `app/common/`

> For complete directory structure, see **[Backend Development → Project Structure](./BACKEND.md#project-structure)**.

## Authentication Architecture

DataHarbor implements the **Backend-For-Frontend (BFF)** pattern with OpenID Connect (OIDC) for secure authentication.

### BFF Authentication Flow

```mermaid
sequenceDiagram
    participant U as User
    participant F as Frontend (Vue)
    participant B as Backend (Go)
    participant O as OIDC Provider
    
    Note over U,O: Initial Access Attempt
    U->>F: Access protected resource
    F->>B: API request
    B->>F: 401 Unauthorized
    F->>U: Redirect to login
    
    Note over U,O: OIDC Authentication Flow
    U->>F: Click login
    F->>B: GET /auth/login
    B->>B: Generate state parameter (CSRF protection)
    B->>U: 302 Redirect to OIDC
    
    U->>O: Authorization request + state
    O->>U: Authentication prompt
    U->>O: Provide credentials
    O->>B: GET /auth/callback?code=X&state=Y
    
    Note over B,O: Token Exchange (Server-to-Server)
    B->>B: Verify state parameter
    B->>O: POST /token (exchange code for tokens)
    O->>B: Access token + ID token + Refresh token
    
    Note over U,F: Secure Session Establishment
    B->>B: Store tokens in HTTP-only cookies
    B->>U: 302 Redirect to original resource
    
    Note over U,B: Authenticated Access
    U->>F: Access original resource
    F->>B: API request (cookies auto-attached)
    B->>B: Validate access token
    B->>F: Protected resource data
    F->>U: Display content
    
    Note over B: Background Token Management
    B->>B: Token refresh when needed
    B->>O: Refresh token exchange
    O->>B: New access token
```

### Advanced Authentication States

```mermaid
stateDiagram-v2
    [*] --> Unauthenticated
    
    Unauthenticated --> AuthInProgress: Login initiated
    AuthInProgress --> OIDCRedirect: Redirect to provider
    OIDCRedirect --> CallbackProcessing: User returns with code
    CallbackProcessing --> TokenExchange: Exchange code for tokens
    TokenExchange --> Authenticated: Tokens stored in session
    TokenExchange --> AuthFailed: Exchange failed
    
    Authenticated --> TokenValidation: Each API request
    TokenValidation --> Authenticated: Token valid
    TokenValidation --> TokenRefresh: Token expired
    TokenRefresh --> Authenticated: Refresh successful
    TokenRefresh --> AuthFailed: Refresh failed
    
    Authenticated --> LogoutInitiated: User logout
    LogoutInitiated --> SessionCleanup: Clear cookies & session
    SessionCleanup --> Unauthenticated: Logout complete
    
    AuthFailed --> Unauthenticated: Redirect to login
    
    note right of TokenExchange : Tokens stored in HTTP-only cookies<br/>Never exposed to JavaScript
    note right of TokenRefresh : Automatic background refresh<br/>No user intervention needed
```

### Security Benefits

1. **Token Security**: Tokens stored in HTTP-only cookies, inaccessible to JavaScript
2. **XSS Protection**: Prevents token theft through client-side attacks
3. **CSRF Protection**: State parameter validation and SameSite cookies
4. **Token Refresh**: Server-side token refresh without user intervention
5. **Session Management**: Centralized session control and logout

## Data Flow

### File Operations Flow

```mermaid
flowchart TD
    A[User Request] --> B[Frontend Validation]
    B --> C[API Call to Backend]
    C --> D[Authentication Check]
    D --> E[Request Validation]
    E --> F[XROOTD Command Execution]
    F --> G[Response Processing]
    G --> H[JSON Response]
    H --> I[Frontend Update]
    I --> J[UI Refresh]
    
    subgraph "Error Handling"
        D --> D1[Token Expired?]
        D1 -->|Yes| D2[Refresh Token]
        D1 -->|No| E
        D2 -->|Success| E
        D2 -->|Failed| D3[Return 401]
        
        F --> F1[XROOTD Error?]
        F1 -->|Yes| F2[Parse Error Type]
        F1 -->|No| G
        F2 --> F3[Return Appropriate HTTP Status]
    end
    
    classDef frontend fill:#e3f2fd
    classDef backend fill:#fff3e0
    classDef xrootd fill:#f1f8e9
    classDef error fill:#ffebee
    
    class A,B,I,J frontend
    class C,D,E,G,H backend
    class F xrootd
    class D1,D2,D3,F1,F2,F3 error
```

### File Download Process (Streaming Architecture)

```mermaid
sequenceDiagram
    participant U as User Browser
    participant F as Frontend
    participant B as Backend
    participant X as XROOTD Server
    
    Note over U,X: Direct Streaming Approach
    U->>F: Click download file
    F->>B: GET /download?path=/file.txt
    
    Note over B: Authentication & Validation
    B->>B: Validate session cookies
    B->>B: Check download slots (rate limiting)
    B->>B: Validate file path (security)
    
    Note over B,X: Direct Streaming Setup
    B->>X: Open file stream (xrdfs cat)
    X->>B: Begin file stream
    
    Note over B,U: HTTP Response Headers
    B->>U: Set response headers<br/>Content-Disposition: attachment<br/>Content-Type: application/octet-stream<br/>Content-Length: file_size
    
    Note over B,U: Streaming Transfer
    loop For each 512KB chunk
        X->>B: Stream data chunk
        B->>U: Forward chunk immediately
        B->>B: Log progress (every 100MB)
    end
    
    Note over B: Completion & Cleanup
    B->>B: Calculate transfer speed
    B->>B: Log completion statistics
    B->>B: Release download slot
```

### XROOTD Client Integration Patterns

```mermaid
graph TB
    subgraph "DataHarbor Backend"
        Controller[Controller Layer]
        XRDClient[XRD Client Wrapper]
        AuthMW[Auth Middleware]
    end
    
    subgraph "XROOTD Native Client"
        NativeClient[go-hep/xrootd Client]
        FileSystem[FileSystem Interface]
        Connection[Connection Management]
    end
    
    subgraph "XROOTD Server"
        XRDServer[XROOTD Daemon]
        Storage[File Storage]
        Auth[XRD Authentication]
    end
    
    Controller --> AuthMW
    AuthMW --> XRDClient
    XRDClient --> NativeClient
    
    NativeClient --> FileSystem
    FileSystem --> Connection
    Connection -->|XRD Protocol| XRDServer
    
    XRDServer --> Storage
    XRDServer --> Auth
    
    Note1[Simple Client Creation<br/>No Connection Pooling<br/>Fresh client per request]
    Note2[Direct Protocol Communication<br/>Binary-safe streaming<br/>Efficient memory usage]
    Note3[File-level permissions<br/>Token-based authentication<br/>Path validation]
    
    XRDClient -.-> Note1
    Connection -.-> Note2
    Auth -.-> Note3
    
    classDef dataharbor fill:#e3f2fd
    classDef native fill:#fff3e0
    classDef server fill:#f1f8e9
    classDef note fill:#f5f5f5
    
    class Controller,XRDClient,AuthMW dataharbor
    class NativeClient,FileSystem,Connection native
    class XRDServer,Storage,Auth server
    class Note1,Note2,Note3 note
```

### File Download Process

**Direct Streaming Approach:**

1. **Request**: User initiates file download via GET request with file path
2. **Authentication**: Backend validates user session and XROOTD token
3. **Path Validation**: File path validated for security (no directory traversal)
4. **Concurrency Check**: Ensures only one download per user session
5. **Direct Streaming**: File streamed from XROOTD to client using `xrdfs cat`
6. **Cleanup**: Download slot released upon completion or error

**Benefits:**

- No temporary storage required
- Immediate download start
- Secure per-request authentication
- Memory efficient streaming

## Design Patterns

### Backend Patterns

#### Middleware Pattern

```go
// Request processing pipeline
router.Use(
    middleware.Recovery(),
    middleware.Logger(),
    middleware.CORS(),
    middleware.Auth(),
)
```

#### Handler Pattern

```go
// Standardized request handling
func (c *Controller) HandleRequest(ctx *gin.Context) {
    // 1. Parse request
    // 2. Validate input
    // 3. Execute business logic
    // 4. Format response
    // 5. Return JSON
}
```

#### Configuration Pattern

```go
// Centralized configuration management
type Config struct {
    Server ServerConfig `yaml:"server"`
    Auth   AuthConfig   `yaml:"auth"`
    XRD    XRDConfig    `yaml:"xrd"`
}
```

### Frontend Patterns

#### Composition API Pattern

```javascript
// Reusable logic with composables
import { useFileOperations } from '@/composables/useFileOperations'

export default {
  setup() {
    const { files, loading, loadDirectory } = useFileOperations()
    return { files, loading, loadDirectory }
  }
}
```

#### Store Pattern (Pinia)

```javascript
// Centralized state management
export const useUserStore = defineStore('user', {
  state: () => ({
    user: null,
    isAuthenticated: false
  }),
  actions: {
    async login() { /* ... */ },
    async logout() { /* ... */ }
  }
})
```

## Configuration Management

### Environment-Specific Configurations

```yaml
# application.development.yaml
server:
  port: 8081
  debug: true
auth:
  enabled: false
xrd:
  timeout: 30s

# application.production.yaml
server:
  port: 8080
  debug: false
auth:
  enabled: true
xrd:
  timeout: 60s
```

### Configuration Hierarchy

1. **Command-line arguments**: `--config=path/to/config.yaml`
2. **Environment variables**: `DATAHARBOR_*`
3. **Configuration files**: YAML format
4. **Default values**: Hardcoded fallbacks

---

## Related Documentation

- **[Authentication System](./AUTHENTICATION.md)** - Security and OIDC integration details
- **[Backend Development](./BACKEND.md)** - Backend implementation guide
- **[Frontend Development](./FRONTEND.md)** - Frontend implementation guide
- **[API Reference](./API.md)** - Complete REST API documentation

---

[← Back to Documentation](./README.md) | [↑ Top](#system-architecture)
