# Authentication and Security

[← Back to Documentation](./README.md)

This document details the authentication mechanisms, security architecture, and security best practices implemented in DataHarbor.

## Authentication Overview

DataHarbor implements OpenID Connect (OIDC) authentication using the Backend-For-Frontend (BFF) pattern. This approach enhances security by keeping sensitive authentication tokens on the server side, away from client-side JavaScript.

## Authentication Architecture Diagrams

### Complete OIDC Authentication Sequence (Technical)

```mermaid
sequenceDiagram
    participant User as User Browser
    participant Frontend as Vue Frontend
    participant Backend as Go Backend  
    participant OIDC as OIDC Provider
    
    Note over User,OIDC: Phase 1: Initial Access & Authentication Request
    User->>Frontend: Access protected resource (/browse)
    Frontend->>Backend: API request to protected endpoint
    Backend->>Backend: AuthMiddleware: Check session cookies
    Backend->>Frontend: 401 Unauthorized (no valid session)
    Frontend->>User: Redirect to login page (/login)
    
    Note over User,OIDC: Phase 2: OIDC Authorization Flow Initiation
    User->>Frontend: Click login button
    Frontend->>Backend: GET /api/v1/auth/login
    Backend->>Backend: Generate random state (CSRF protection)
    Backend->>Backend: session.Set("oauth_state", state)
    Backend->>Backend: session.Set("original_url", redirect_url)
    Backend->>Backend: Build authorization URL with state
    Backend->>User: 302 Redirect to OIDC Provider
    
    Note over User,OIDC: Phase 3: User Authentication at OIDC Provider
    User->>OIDC: Authorization request with state parameter
    OIDC->>User: Present authentication form
    User->>OIDC: Submit credentials (username/password)
    OIDC->>OIDC: Validate user credentials
    OIDC->>Backend: GET /api/v1/auth/callback?code=ABC&state=XYZ
    
    Note over Backend,OIDC: Phase 4: Token Exchange (Server-to-Server)
    Backend->>Backend: Verify state parameter matches session
    Backend->>OIDC: POST /token<br/>grant_type=authorization_code<br/>code=ABC<br/>client_id=dataharbor<br/>client_secret=***
    OIDC->>Backend: {<br/>  access_token: "...",<br/>  id_token: "...",<br/>  refresh_token: "...",<br/>  expires_in: 3600<br/>}
    
    Note over Backend: Phase 5: Secure Session Establishment
    Backend->>Backend: session.Set("access_token", access_token)
    Backend->>Backend: session.Set("id_token", id_token)
    Backend->>Backend: session.Set("refresh_token", refresh_token)
    Backend->>Backend: session.Save() → HTTP-only cookies
    Backend->>User: 302 Redirect to original protected resource
    
    Note over User,Backend: Phase 6: Authenticated Resource Access
    User->>Frontend: Navigate to original resource (/browse)
    Frontend->>Backend: API request (cookies auto-attached by browser)
    Backend->>Backend: AuthMiddleware: Extract & validate access token
    Backend->>Backend: Parse JWT claims, verify signature & expiration
    Backend->>Backend: ctx.Set("user", userInfo)
    Backend->>Frontend: Protected resource data
    Frontend->>User: Display protected content
    
    Note over Backend,OIDC: Background: Automatic Token Refresh
    Backend->>Backend: Token expiration check on each request
    Backend->>OIDC: POST /token<br/>grant_type=refresh_token<br/>refresh_token=...
    OIDC->>Backend: New access_token + refresh_token
    Backend->>Backend: Update session with new tokens
```

### Session Management & Token Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Unauthenticated: Application start
    
    Unauthenticated --> LoginInitiated: User clicks login
    LoginInitiated --> OIDCRedirect: Generate state, redirect to OIDC
    OIDCRedirect --> UserAuthentication: User enters credentials
    UserAuthentication --> CallbackReceived: OIDC redirects with code
    
    CallbackReceived --> StateValidation: Verify state parameter
    StateValidation --> TokenExchange: State valid
    StateValidation --> SecurityError: State mismatch (CSRF attempt)
    
    TokenExchange --> SessionEstablished: Tokens stored in HTTP-only cookies
    TokenExchange --> TokenExchangeError: OIDC token exchange failed
    
    SessionEstablished --> Authenticated: Session active
    
    Authenticated --> TokenValidation: Each API request
    TokenValidation --> Authenticated: Token valid & not expired
    TokenValidation --> TokenRefresh: Token expired but refresh available
    TokenValidation --> SessionExpired: No refresh token or refresh failed
    
    TokenRefresh --> Authenticated: New access token obtained
    TokenRefresh --> SessionExpired: Refresh token expired/invalid
    
    Authenticated --> LogoutInitiated: User clicks logout
    LogoutInitiated --> SessionCleanup: Clear all session data
    SessionCleanup --> Unauthenticated: Clean logout
    
    SessionExpired --> Unauthenticated: Force re-authentication
    SecurityError --> Unauthenticated: Security violation
    TokenExchangeError --> Unauthenticated: Authentication failed
    
    note right of SessionEstablished : Tokens never exposed to JavaScript<br/>Stored in HTTP-only, Secure, SameSite cookies
    note right of TokenRefresh : Automatic background refresh<br/>Transparent to user experience
    note right of StateValidation : CSRF protection prevents<br/>malicious callback attacks
```

### Backend-for-Frontend (BFF) Security Architecture

```mermaid
graph TB
    subgraph "Browser (Untrusted Environment)"
        JS[JavaScript Application]
        Cookies[HTTP-only Cookies]
        LocalStorage[Local Storage]
        SessionStorage[Session Storage]
    end
    
    subgraph "DataHarbor Backend (Trusted Environment)"
        AuthMW[Auth Middleware]
        SessionMgr[Session Manager]
        TokenStore[Token Storage]
        OIDCClient[OIDC Client]
    end
    
    subgraph "OIDC Provider (External)"
        AuthServer[Authorization Server]
        TokenEndpoint[Token Endpoint]
        UserInfo[UserInfo Endpoint]
    end
    
    JS -.->|❌ CANNOT ACCESS| Cookies
    JS -->|✅ Safe to use| LocalStorage
    JS -->|✅ Safe to use| SessionStorage
    
    Cookies -->|Automatic attachment| AuthMW
    AuthMW --> SessionMgr
    SessionMgr --> TokenStore
    
    OIDCClient -->|Server-to-server| AuthServer
    OIDCClient -->|Token exchange| TokenEndpoint
    OIDCClient -->|User details| UserInfo
    
    Note1[🔒 Access tokens never<br/>exposed to JavaScript]
    Note2[🔒 Refresh tokens never<br/>leave server environment]
    Note3[🔒 CSRF protection via<br/>SameSite cookie attribute]
    Note4[🔒 XSS protection via<br/>HTTP-only cookie flag]
    
    TokenStore -.-> Note1
    TokenStore -.-> Note2
    Cookies -.-> Note3
    Cookies -.-> Note4
    
    classDef browser fill:#ffebee
    classDef backend fill:#e8f5e8
    classDef oidc fill:#e3f2fd
    classDef security fill:#fff3e0
    
    class JS,Cookies,LocalStorage,SessionStorage browser
    class AuthMW,SessionMgr,TokenStore,OIDCClient backend
    class AuthServer,TokenEndpoint,UserInfo oidc
    class Note1,Note2,Note3,Note4 security
```

### Authentication Components

#### OIDC Provider Configuration

```yaml
# Configuration example
auth:
  enabled: true
  oidc:
    issuer: "https://keycloak.example.com/realms/dataharbor"
    client_id: "dataharbor-client"
    client_secret: "${OIDC_CLIENT_SECRET}"
    redirect_uri: "https://dataharbor.example.com/api/v1/auth/callback"
    scopes: ["openid", "profile", "email"]
  session:
    secret: "${SESSION_SECRET}"
    max_age: 3600  # 1 hour
    secure: true
    http_only: true
    same_site: "strict"
```

## Security Architecture

### Security Layers

1. **Transport Security**
   - HTTPS/TLS encryption for all communications
   - HTTP Strict Transport Security (HSTS) headers
   - Certificate pinning in production

2. **Authentication Security**
   - OIDC standard compliance
   - State parameter for CSRF protection
   - Secure token storage in HTTP-only cookies

3. **Session Security**
   - HTTP-only cookies prevent XSS access
   - Secure flag ensures HTTPS-only transmission
   - SameSite attribute prevents CSRF attacks
   - Configurable session timeout

4. **Authorization Security**
   - Token-based authorization
   - Role-based access control (RBAC)
   - Resource-level permissions

5. **Input Security**
   - Request validation and sanitization
   - Path traversal protection
   - SQL injection prevention (when database is used)

## Token Management

### Token Types and Usage

1. **Access Token**
   - Used for API authorization
   - Short-lived (typically 15-60 minutes)
   - Contains user permissions and roles
   - Validated on each API request

2. **ID Token**
   - Contains user identity information
   - Used for user profile display
   - JWT format with signature verification

3. **Refresh Token**
   - Used to obtain new access tokens
   - Longer-lived (hours to days)
   - Securely stored and rotated

---

## Related Documentation

- **[System Architecture](./ARCHITECTURE.md)** - Overall system design
- **[Backend Development](./BACKEND.md)** - Auth middleware implementation
- **[API Reference](./API.md)** - Authentication endpoints
- **[Troubleshooting](./TROUBLESHOOTING.md)** - Auth issues and solutions

---

[← Back to Documentation](./README.md) | [↑ Top](#authentication-and-security)
