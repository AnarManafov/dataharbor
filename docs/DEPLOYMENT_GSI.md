# DataHarbor Deployment Guide for GSI Environment

This guide provides step-by-step instructions for deploying DataHarbor on GSI servers using RPM packages.

**Document Status**: Production-ready configuration for HTTPS deployment  
**Last Updated**: October 2025 (Port 443 HTTPS configuration)

---

## 📋 Table of Contents

1. [Prerequisites](#prerequisites)
2. [Environment Overview](#environment-overview)
3. [Initial Installation (One-Time Setup)](#initial-installation-one-time-setup)
4. [Version Updates](#version-updates-upgrading-dataharbor)
5. [Verification & Testing](#verification--testing)
6. [Troubleshooting](#troubleshooting)
7. [Quick Reference](#quick-reference)

---

## Prerequisites

### Required Access & Tools
- ✅ Root access to GSI server
- ✅ SSH access with public key authentication
- ✅ XRootD server already running on the system (port 80 and 1094)
- ✅ SSL certificates available (self-signed acceptable for testing)
- ✅ OIDC provider configured at Keycloak (id.gsi.de)

### Pre-existing Infrastructure
- **XRootD Server**: Already running on port 80 (HTTP) and 1094 (XRootD protocol)
- **Keycloak OIDC**: https://id.gsi.de/realms/wl
- **Network**: GSI institutional network with firewall (port 443 typically open)

---

## Environment Overview

### Server Configuration Example

**Example Server**: punch2.gsi.de (140.181.3.31)

### Port Allocation

| Service             | Port  | Protocol | Purpose              | Notes                                 |
| ------------------- | ----- | -------- | -------------------- | ------------------------------------- |
| XRootD Protocol     | 1094  | XRootD   | File system access   | Pre-existing                          |
| XRootD HTTP         | 80    | HTTP     | XRootD web interface | Pre-existing                          |
| DataHarbor Backend  | 22000 | HTTPS    | API server           | SSL enabled                           |
| DataHarbor Frontend | 443   | HTTPS    | Web UI               | SSL enabled, reverse proxy to backend |
| Keycloak OIDC       | 443   | HTTPS    | Authentication       | External service                      |

### File Locations (Standard)

| Item             | Location                                                      |
| ---------------- | ------------------------------------------------------------- |
| Backend binary   | `/usr/local/bin/dataharbor-backend`                           |
| Backend config   | `/root/dataharbor/config/backend-config-gsi-test-server.yaml` |
| Backend service  | `/etc/systemd/system/dataharbor-backend.service`              |
| Backend logs     | `/var/log/dataharbor/dataharbor-backend.log`                  |
| Frontend files   | `/usr/share/dataharbor-frontend/`                             |
| Frontend config  | `/usr/share/dataharbor-frontend/config.json`                  |
| Nginx config     | `/etc/nginx/conf.d/dataharbor.conf`                           |
| SSL certificates | `/root/dataharbor/config/cert/server.crt` and `server.key`    |

---

---

## Initial Installation (One-Time Setup)

This section covers the complete initial setup. These steps are **only performed once** per server.

---

### Step 1: Prepare Configuration Directory

Create the configuration directory structure before installing RPM packages:

```bash
# Create configuration directory
sudo mkdir -p /root/dataharbor/config/cert

# Create log directory
sudo mkdir -p /var/log/dataharbor
sudo chmod 755 /var/log/dataharbor
```

### Step 2: Prepare SSL Certificates

Place your SSL certificates in the configuration directory:

```bash
# Copy your SSL certificates (adjust paths as needed)
sudo cp /path/to/your/server.crt /root/dataharbor/config/cert/
sudo cp /path/to/your/server.key /root/dataharbor/config/cert/

# Set proper permissions
sudo chmod 600 /root/dataharbor/config/cert/server.key
sudo chmod 644 /root/dataharbor/config/cert/server.crt

# Verify certificates exist
ls -la /root/dataharbor/config/cert/
```

**Note**: Self-signed certificates are acceptable for testing environments.

### Step 3: Create Backend Configuration File

Create the backend configuration file **before** installing the RPM:

```bash
sudo tee /root/dataharbor/config/backend-config-gsi-test-server.yaml << 'EOF'
env: production

server:
  address: ":22000"
  debug: false
  shutdown_timeout: 30s
  cors:
    allow_credentials: true
    allow_headers:
      - Origin
      - Content-Length
      - Content-Type
      - Authorization
    allow_methods:
      - GET
      - POST
      - PUT
      - DELETE
      - OPTIONS
    allow_origins:
      - https://punch2
      - https://punch2.gsi.de
  ssl:
    enabled: true
    cert_file: /root/dataharbor/config/cert/server.crt
    key_file: /root/dataharbor/config/cert/server.key

logging:
  level: info
  console:
    enabled: true
    level: info
    format: text
  file:
    enabled: true
    level: info
    filename: "/var/log/dataharbor/dataharbor-backend.log"
    maxsize: 10
    maxbackups: 2
    maxage: 27
    compress: true

xrd:
  host: "localhost"
  port: 1094
  initial_dir: "/data/xrootd"  # GSI XRootD root directory
  user: ""
  usergroup: ""
  user_required: false
  tls: false
  client_cert: ""
  client_key: ""

frontend:
  url: "https://punch2"
  dist_dir: "dist"
  asset_paths:
    - "../sandbox/public"
    - "web"

auth:
  enabled: true
  skip_auth_paths:
    - "/health"
    - "/api/auth/login"
    - "/api/auth/callback"
  oidc:
    issuer: "https://id.gsi.de/realms/wl"
    client_id: "xrootd"
    client_secret: "your-client-secret-here"  # Replace with actual secret
    discovery_url: "https://id.gsi.de/realms/wl/.well-known/openid-configuration"
    allowed_roles:
      - "xrootd-user"
    session_secret: "GENERATE-YOUR-OWN-SESSION-SECRET-HERE"  # Generate unique random string
    token_refresh_buffer_sec: 60
EOF
```

**Important**: Replace the following placeholders:
- `your-client-secret-here`: Your actual Keycloak client secret
- `session_secret`: Generate a unique random string
- `https://punch2` and `https://punch2.gsi.de`: Replace `punch2` with your actual server hostname

**Note**: The `initial_dir` is set to `/data/xrootd` which is the standard XRootD root directory for GSI servers.

### Step 4: Create Frontend Configuration File

```bash
sudo tee /root/dataharbor/config/frontend-config-gsi-test-server.json << 'EOF'
{
  "apiBaseUrl": "/api",
  "features": {
    "enableDocumentation": true
  }
}
EOF
```

**Note**: The `apiBaseUrl: "/api"` uses a relative path because nginx will act as a reverse proxy.

### Step 5: Install RPM Packages

Now install both backend and frontend RPM packages:

```bash
# Install backend RPM
sudo rpm -ivh dataharbor-backend-*.rpm

# Install frontend RPM
sudo rpm -ivh dataharbor-frontend-*.rpm

# Verify installation
rpm -ql dataharbor-backend
rpm -ql dataharbor-frontend
```

**Installation Locations After RPM Install:**
- Backend binary: `/usr/local/bin/dataharbor-backend`
- Frontend files: `/usr/share/dataharbor-frontend/`

### Step 6: Create SystemD Service for Backend

The backend RPM only installs the binary. You must create the systemd service file manually:

```bash
sudo tee /etc/systemd/system/dataharbor-backend.service << 'EOF'
[Unit]
Description=DataHarbor Backend Service
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/dataharbor-backend --config=/root/dataharbor/config/backend-config-gsi-test-server.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
```

**Note**: Adjust the `--config` path if you used a different location.

### Step 7: Enable and Start Backend Service

```bash
# Reload systemd to recognize the new service
sudo systemctl daemon-reload

# Enable service to start on boot
sudo systemctl enable dataharbor-backend

# Start the service
sudo systemctl start dataharbor-backend

# Check status
sudo systemctl status dataharbor-backend
```

### Step 8: Verify Backend is Running

```bash
# Test health endpoint
curl -k https://localhost:22000/health

# Expected response:
# {"code":200,"data":"ok","message":"success"}

# Check systemd logs
sudo journalctl -u dataharbor-backend -n 50

# Check file logs (if enabled)
tail -f /var/log/dataharbor/dataharbor-backend.log
```

If health check fails, see [Troubleshooting](#troubleshooting) section.

### Step 9: Deploy Frontend Configuration

Copy the frontend config to the installation directory:

```bash
# Copy frontend config
sudo cp /root/dataharbor/config/frontend-config-gsi-test-server.json \
        /usr/share/dataharbor-frontend/config.json

# Verify the config
cat /usr/share/dataharbor-frontend/config.json
```

### Step 10: Configure Nginx (HTTPS on Port 443)

Create the nginx configuration for HTTPS access:

```bash
sudo tee /etc/nginx/conf.d/dataharbor.conf << 'EOF'
server {
    listen 443 ssl http2;
    server_name punch2 punch2.gsi.de;

    # SSL Configuration
    ssl_certificate /root/dataharbor/config/cert/server.crt;
    ssl_certificate_key /root/dataharbor/config/cert/server.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # Frontend files
    root /usr/share/dataharbor-frontend;
    index index.html;

    # Serve frontend static files
    location / {
        try_files $uri $uri/ /index.html;
    }

    # Cache static assets
    location /assets/ {
        alias /usr/share/dataharbor-frontend/assets/;
        expires max;
        access_log off;
        add_header Cache-Control "public";
    }

    # Reverse proxy to backend API
    location /api/ {
        proxy_pass https://localhost:22000;
        proxy_ssl_verify off;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Logging
    error_log /var/log/nginx/dataharbor-frontend-error.log;
    access_log /var/log/nginx/dataharbor-frontend-access.log;
}
EOF
```

**Important Notes**:
- **Replace `punch2` and `punch2.gsi.de`** with your actual server hostname
- **Port 443 only**: No HTTP redirect on port 80 (XRootD uses port 80)
- SSL certificates path must match your actual certificate locations

### Step 11: Comment Out Default Nginx Server Block

The default nginx server block may try to listen on port 80, conflicting with XRootD:

```bash
# Backup the default config
sudo cp /etc/nginx/nginx.conf /etc/nginx/nginx.conf.bak

# Edit nginx.conf and comment out the default server block
sudo sed -i '/^[[:space:]]*server[[:space:]]*{/,/^[[:space:]]*}/s/^/#/' /etc/nginx/nginx.conf
```

Or manually edit `/etc/nginx/nginx.conf` and comment out the `server { }` block inside the `http { }` section.

### Step 12: Test and Start Nginx

```bash
# Test nginx configuration syntax
sudo nginx -t

# If test passes, enable nginx
sudo systemctl enable nginx

# Start nginx
sudo systemctl start nginx

# Check status
sudo systemctl status nginx

# Verify nginx is listening on port 443 only
sudo ss -tlnp | grep nginx
# Should show: LISTEN on 0.0.0.0:443
```

### Step 13: Initial Testing

Test both frontend and backend locally:

```bash
# Test frontend HTML loads
curl -k https://localhost/

# Should return HTML content with <title>DataHarbor</title>

# Test API proxy to backend
curl -k https://localhost/api/health

# Should return: {"code":200,"data":"ok","message":"success"}
```

### Step 14: External Access Testing

From your local machine (outside the server), test external access:

```bash
# Test frontend access (replace punch2.gsi.de with your server)
curl -k https://punch2.gsi.de/

# Test API access
curl -k https://punch2.gsi.de/api/health
```

If external access fails, check:
- Firewall allows port 443 (usually open by default for HTTPS)
- SELinux is not blocking (set to Permissive if needed)
- DNS resolution for your hostname

### Step 15: Browser Testing

Open your browser and navigate to:

**URL**: `https://punch2.gsi.de/` (replace with your server hostname)

**Verify**:
- ✅ Frontend loads successfully (you may need to accept self-signed cert)
- ✅ No console errors in browser DevTools (F12)
- ✅ Can click "Login" button
- ✅ Redirects to Keycloak (id.gsi.de)
- ✅ After login, can browse XRootD directories
- ✅ Network tab shows successful API calls to `/api/*`

---

## Version Updates (Upgrading DataHarbor)

This section covers upgrading to a new version of DataHarbor. These steps are performed **every time you update** to a new version.

---

### Update Checklist

**What needs to be updated:**
- ✅ Backend RPM package
- ✅ Frontend RPM package
- ✅ Frontend config.json (copy to installation directory)

**What does NOT need to be changed:**
- ❌ Backend config YAML (unless new features require it)
- ❌ SystemD service file (already created during initial setup)
- ❌ Nginx configuration (already created during initial setup)
- ❌ SSL certificates (unless renewing/replacing)
- ❌ Log directory (already exists)

### Update Procedure

#### Step 1: Stop Running Services

```bash
# Stop backend service
sudo systemctl stop dataharbor-backend

# Stop nginx (optional, can update without stopping)
# sudo systemctl stop nginx
```

#### Step 2: Backup Current Configuration

Always backup before updating:

```bash
# Create backup directory with timestamp
BACKUP_DIR=~/dataharbor-backups/backup-$(date +%Y%m%d-%H%M%S)
mkdir -p $BACKUP_DIR

# Backup configuration files
sudo cp -r /root/dataharbor/config/ $BACKUP_DIR/
sudo cp /etc/systemd/system/dataharbor-backend.service $BACKUP_DIR/
sudo cp /etc/nginx/conf.d/dataharbor.conf $BACKUP_DIR/
sudo cp /usr/share/dataharbor-frontend/config.json $BACKUP_DIR/

# List backup
ls -la $BACKUP_DIR/

echo "Backup saved to: $BACKUP_DIR"
```

#### Step 3: Update Backend RPM

```bash
# Update (or reinstall) backend RPM
sudo rpm -Uvh dataharbor-backend-*.rpm

# Verify new version installed
rpm -qi dataharbor-backend | grep Version
```

**Note**: The binary location remains the same: `/usr/local/bin/dataharbor-backend`

#### Step 4: Update Frontend RPM

```bash
# Update (or reinstall) frontend RPM
sudo rpm -Uvh dataharbor-frontend-*.rpm

# Verify new version installed
rpm -qi dataharbor-frontend | grep Version
```

**Note**: Frontend files are updated in: `/usr/share/dataharbor-frontend/`

#### Step 5: Update Frontend Configuration

The RPM installation may overwrite `config.json`, so re-copy your configuration:

```bash
# Copy your frontend config back
sudo cp /root/dataharbor/config/frontend-config-gsi-test-server.json \
        /usr/share/dataharbor-frontend/config.json

# Verify the config
cat /usr/share/dataharbor-frontend/config.json
```

#### Step 6: Review Backend Configuration (If Needed)

Check release notes for any new configuration options:

```bash
# Edit backend config if needed
sudo nano /root/dataharbor/config/backend-config-gsi-test-server.yaml
```

**Common reasons to update backend config:**
- New authentication options
- New XRootD features
- Performance tuning options
- CORS origins changes

#### Step 7: Restart Services

```bash
# Reload systemd (in case service file changed)
sudo systemctl daemon-reload

# Restart backend
sudo systemctl restart dataharbor-backend

# Check backend status
sudo systemctl status dataharbor-backend

# Reload nginx (reload is enough, no restart needed)
sudo systemctl reload nginx

# Check nginx status
sudo systemctl status nginx
```

#### Step 8: Verify Update

```bash
# Test backend health
curl -k https://localhost:22000/health

# Expected: {"code":200,"data":"ok","message":"success"}

# Test frontend
curl -k https://localhost/

# Should return updated HTML

# Test API proxy
curl -k https://localhost/api/health

# Check backend logs for errors
sudo journalctl -u dataharbor-backend -n 50
```

#### Step 9: Browser Testing After Update

Open browser to `https://your-server/` and verify:

- ✅ Frontend loads with new version
- ✅ Clear browser cache (Ctrl+Shift+R or Cmd+Shift+R)
- ✅ No console errors
- ✅ Login still works
- ✅ XRootD browsing works
- ✅ All features functional

#### Step 10: Monitor Logs

After update, monitor logs for any issues:

```bash
# Watch backend logs in real-time
sudo journalctl -u dataharbor-backend -f

# Watch nginx error logs
sudo tail -f /var/log/nginx/dataharbor-frontend-error.log
```

Press `Ctrl+C` to stop watching.

---

## Verification & Testing

### Quick Health Check

Run these commands to verify everything is working:

```bash
# 1. Check backend service
sudo systemctl status dataharbor-backend

# 2. Check nginx service
sudo systemctl status nginx

# 3. Check listening ports
sudo ss -tlnp | grep -E ':(22000|443)'

# 4. Test backend health
curl -k https://localhost:22000/health

# 5. Test frontend through nginx
curl -k https://localhost/

# 6. Test API proxy
curl -k https://localhost/api/health
```

All should return success/running status.

### External Access Test

From a different machine on the network:

```bash
# Replace punch2.gsi.de with your server hostname
curl -k https://punch2.gsi.de/

curl -k https://punch2.gsi.de/api/health
```

### Browser End-to-End Test

1. Open browser: `https://your-server/`
2. Accept self-signed certificate warning (if applicable)
3. Click "Login" button
4. Redirected to Keycloak → Enter credentials
5. Redirected back to DataHarbor → Logged in
6. Browse XRootD directories
7. Open browser DevTools (F12):
   - **Console tab**: No errors
   - **Network tab**: API calls to `/api/*` return 200 OK

---

## Troubleshooting

### Backend Issues

#### Issue: Backend won't start

```bash
# Check logs for errors
sudo journalctl -u dataharbor-backend -n 100

# Check config file syntax
cat /root/dataharbor/config/backend-config-gsi-test-server.yaml

# Verify certificates exist and are readable
ls -la /root/dataharbor/config/cert/

# Check if port 22000 is already in use
sudo ss -tlnp | grep ':22000'

# Check for permission issues
sudo journalctl -u dataharbor-backend | grep -i permission
```

#### Issue: Health check fails

```bash
# Try both health endpoints
curl -k https://localhost:22000/health
curl -k https://localhost:22000/api/health

# Check if backend is listening on port 22000
sudo ss -tlnp | grep ':22000'

# View real-time logs
sudo journalctl -u dataharbor-backend -f

# Check SSL certificate issues
openssl s_client -connect localhost:22000 -showcerts
```

#### Issue: Backend logs show SSL errors

```bash
# Verify SSL certificate files exist and have correct permissions
ls -la /root/dataharbor/config/cert/

# Certificates should be:
# - server.crt: 644 (readable by all)
# - server.key: 600 (readable by owner only)

# Fix permissions if needed
sudo chmod 644 /root/dataharbor/config/cert/server.crt
sudo chmod 600 /root/dataharbor/config/cert/server.key

# Restart backend
sudo systemctl restart dataharbor-backend
```

### Frontend / Nginx Issues

#### Issue: Nginx won't start - Port conflict

**If you see "bind() to 0.0.0.0:80 failed":**

Port 80 is used by XRootD HTTP service. DataHarbor uses port 443 (HTTPS) only.

```bash
# Check what's on port 80
sudo ss -tlnp | grep ':80'

# Verify nginx config uses port 443 (NOT port 80)
sudo grep -n 'listen' /etc/nginx/conf.d/dataharbor.conf

# Should show: listen 443 ssl http2;
# Should NOT have: listen 80;

# If you see port 80, remove it from config
sudo nano /etc/nginx/conf.d/dataharbor.conf
```

#### Issue: Frontend loads but API calls fail (502 Bad Gateway)

```bash
# Check nginx error logs
sudo tail -f /var/log/nginx/dataharbor-frontend-error.log

# Verify backend is running
curl -k https://localhost:22000/health

# Check proxy configuration
sudo grep -A 10 'location /api' /etc/nginx/conf.d/dataharbor.conf

# Verify proxy_pass points to correct backend
# Should be: proxy_pass https://localhost:22000;
```

#### Issue: CORS errors in browser console

Update backend CORS configuration to include your frontend HTTPS origin:

```yaml
# Edit: /root/dataharbor/config/backend-config-gsi-test-server.yaml
server:
  cors:
    allow_origins:
      - https://punch2
      - https://punch2.gsi.de
```

Then restart backend:

```bash
sudo systemctl restart dataharbor-backend
```

#### Issue: External access fails (connection timeout)

```bash
# Check if port 443 is open in firewall
sudo firewall-cmd --list-ports
# or
sudo iptables -L -n | grep 443

# Check if nginx is listening on all interfaces (0.0.0.0:443)
sudo ss -tlnp | grep nginx

# Test from server itself
curl -k https://localhost/

# If local works but external doesn't, check:
# 1. Institutional firewall (contact GSI network team)
# 2. SELinux blocking: sudo setenforce 0 (temporarily)
```

### Authentication Issues

#### Issue: "Invalid parameter: redirect_uri" Error

**Symptom**: After clicking Login, redirected to Keycloak but get error: "Invalid parameter: redirect_uri"

**Cause**: Keycloak client configuration doesn't include your new server URL in allowed redirect URIs.

**Solution**: Contact your Keycloak administrator to add redirect URIs for your server.

**What to tell the Keycloak admin:**

```
Please add the following Valid Redirect URIs to the "xrootd" client in Keycloak:

1. https://punch2.gsi.de/*
2. https://punch2/*
3. https://punch2.gsi.de/auth/callback
4. https://punch2/auth/callback

Also add to Valid Post Logout Redirect URIs:
1. https://punch2.gsi.de/*
2. https://punch2/*

Client ID: xrootd
Realm: wl
Keycloak URL: https://id.gsi.de
```

**Alternative - Check current Keycloak configuration** (if you have access):

1. Log into Keycloak admin console: `https://id.gsi.de/admin`
2. Select realm: `wl`
3. Go to: Clients → `xrootd`
4. Check "Valid Redirect URIs" field
5. Should include: `https://punch2.gsi.de/*` and `https://punch2/*`

**If you were previously using localhost:**

The old redirect URIs were probably:
- `http://localhost:8080/*` or
- `http://localhost/*`

Now you need the HTTPS production URIs listed above.

#### Issue: Can't log in / OIDC redirect fails (general)

```bash
# Check backend logs for OIDC errors
sudo journalctl -u dataharbor-backend | grep -i oidc

# Look for specific error messages like:
# - "redirect_uri mismatch"
# - "invalid redirect_uri"
# - "unauthorized client"

# Verify OIDC configuration
grep -A 10 'oidc:' /root/dataharbor/config/backend-config-gsi-test-server.yaml

# Test OIDC discovery URL
curl https://id.gsi.de/realms/wl/.well-known/openid-configuration

# Check what redirect_uri DataHarbor is using
sudo journalctl -u dataharbor-backend | grep redirect_uri

# Common issues:
# 1. Frontend URL mismatch - backend config frontend.url should match actual access URL
# 2. Keycloak client redirect URIs not updated
# 3. HTTP vs HTTPS mismatch
```

#### Issue: "Unauthorized" or "Forbidden" after login

```bash
# Check allowed roles in backend config
grep -A 5 'allowed_roles:' /root/dataharbor/config/backend-config-gsi-test-server.yaml

# Ensure role matches what Keycloak provides
# Example: "xrootd-user"

# Check user's Keycloak roles (in Keycloak admin console)
# User must have the role specified in allowed_roles
```

### XRootD Integration Issues

#### Issue: Can't browse XRootD directories

```bash
# Check XRootD configuration in backend
grep -A 10 'xrd:' /root/dataharbor/config/backend-config-gsi-test-server.yaml

# Test XRootD connectivity directly
xrdfs localhost:1094 ls /

# If xrdfs works but DataHarbor doesn't, check backend logs
sudo journalctl -u dataharbor-backend | grep -i xrd

# Common issues:
# - initial_dir path doesn't exist in XRootD
# - XRootD authentication required but not configured
# - XRootD server not running
```

#### Issue: XRootD authentication errors

```bash
# If XRootD requires authentication, update backend config:
# xrd:
#   host: "localhost"
#   port: 1094
#   tls: true
#   client_cert: "/path/to/client.crt"
#   client_key: "/path/to/client.key"

# Verify XRootD server allows connections
sudo systemctl status xrootd

# Check XRootD logs
sudo journalctl -u xrootd -n 50
```

### Performance Issues

#### Issue: Slow directory browsing

```bash
# Check XRootD server performance
xrdfs localhost:1094 ls -l /store/  # Time this command

# Check network latency to XRootD
ping localhost

# Monitor backend resource usage
top -p $(pgrep dataharbor-backend)

# Increase logging to debug performance
# In backend config, set: logging.level: debug
# Then restart and monitor logs
```

---

## Quick Reference

### Service Management Commands

```bash
# Backend Service
sudo systemctl status dataharbor-backend     # Check status
sudo systemctl start dataharbor-backend      # Start
sudo systemctl stop dataharbor-backend       # Stop
sudo systemctl restart dataharbor-backend    # Restart
sudo systemctl enable dataharbor-backend     # Enable on boot

# Nginx Service
sudo systemctl status nginx                   # Check status
sudo systemctl start nginx                    # Start
sudo systemctl stop nginx                     # Stop
sudo systemctl restart nginx                  # Restart
sudo systemctl reload nginx                   # Reload config (no downtime)
sudo systemctl enable nginx                   # Enable on boot
```

### Health Check Commands

```bash
# Backend Health (Direct)
curl -k https://localhost:22000/health

# Backend Health (Through Nginx Proxy)
curl -k https://localhost/api/health

# Frontend HTML
curl -k https://localhost/

# All should return successful responses
```

### Log Viewing Commands

```bash
# Backend Logs (SystemD)
sudo journalctl -u dataharbor-backend -f          # Follow real-time
sudo journalctl -u dataharbor-backend -n 100      # Last 100 lines
sudo journalctl -u dataharbor-backend --since "1 hour ago"

# Backend Logs (File, if enabled)
tail -f /var/log/dataharbor/dataharbor-backend.log

# Nginx Error Logs
tail -f /var/log/nginx/dataharbor-frontend-error.log

# Nginx Access Logs
tail -f /var/log/nginx/dataharbor-frontend-access.log
```

### Configuration File Locations

```bash
# Backend Configuration
/root/dataharbor/config/backend-config-gsi-test-server.yaml

# Frontend Configuration
/usr/share/dataharbor-frontend/config.json

# Backend SystemD Service
/etc/systemd/system/dataharbor-backend.service

# Nginx Configuration
/etc/nginx/conf.d/dataharbor.conf

# SSL Certificates
/root/dataharbor/config/cert/server.crt
/root/dataharbor/config/cert/server.key
```

### Port and Service Overview

| Service             | Port  | Protocol | URL                       | Notes                           |
| ------------------- | ----- | -------- | ------------------------- | ------------------------------- |
| XRootD Protocol     | 1094  | XRootD   | `root://localhost:1094`   | Pre-existing                    |
| XRootD HTTP         | 80    | HTTP     | `http://localhost/`       | Pre-existing, **Do not modify** |
| DataHarbor Backend  | 22000 | HTTPS    | `https://localhost:22000` | SSL enabled                     |
| DataHarbor Frontend | 443   | HTTPS    | `https://your-server/`    | SSL enabled, reverse proxy      |
| Keycloak OIDC       | 443   | HTTPS    | `https://id.gsi.de`       | External authentication         |

### Quick Diagnostic Commands

```bash
# Check all services
sudo systemctl status dataharbor-backend nginx

# Check listening ports
sudo ss -tlnp | grep -E ':(22000|443)'

# Test complete stack
curl -k https://localhost:22000/health && \
curl -k https://localhost/ && \
curl -k https://localhost/api/health && \
echo "All tests passed!"

# Check recent errors
sudo journalctl -u dataharbor-backend -p err --since "1 hour ago"
sudo tail -50 /var/log/nginx/dataharbor-frontend-error.log
```

### Common Maintenance Tasks

#### Update frontend config only

```bash
# Edit config
sudo nano /usr/share/dataharbor-frontend/config.json

# No restart needed - browser will fetch on next reload
```

#### Update backend config

```bash
# Edit config
sudo nano /root/dataharbor/config/backend-config-gsi-test-server.yaml

# Restart backend
sudo systemctl restart dataharbor-backend
```

#### Update nginx config

```bash
# Edit config
sudo nano /etc/nginx/conf.d/dataharbor.conf

# Test syntax
sudo nginx -t

# Reload (no downtime)
sudo systemctl reload nginx
```

#### Rotate logs manually

```bash
# Trigger logrotate for DataHarbor
sudo logrotate -f /etc/logrotate.d/dataharbor
```

#### Clear browser cache issues

```bash
# If frontend shows old version after update:
# 1. Hard refresh: Ctrl+Shift+R (Windows/Linux) or Cmd+Shift+R (Mac)
# 2. Clear browser cache
# 3. Open incognito/private window
# 4. Check browser console for cache errors
```

---

## Backup and Recovery

### Create Complete Backup

```bash
# Create timestamped backup
BACKUP_DIR=~/dataharbor-backups/backup-$(date +%Y%m%d-%H%M%S)
mkdir -p $BACKUP_DIR

# Backup all critical files
sudo cp -r /root/dataharbor/config/ $BACKUP_DIR/
sudo cp /etc/systemd/system/dataharbor-backend.service $BACKUP_DIR/
sudo cp /etc/nginx/conf.d/dataharbor.conf $BACKUP_DIR/
sudo cp /usr/share/dataharbor-frontend/config.json $BACKUP_DIR/

# Create archive
tar -czf ~/dataharbor-backups/dataharbor-backup-$(date +%Y%m%d-%H%M%S).tar.gz \
  -C $BACKUP_DIR .

# List backups
ls -lh ~/dataharbor-backups/

echo "Backup completed: $BACKUP_DIR"
```

### Restore from Backup

```bash
# List available backups
ls -lh ~/dataharbor-backups/

# Extract backup (replace TIMESTAMP with your backup timestamp)
BACKUP_FILE=~/dataharbor-backups/dataharbor-backup-TIMESTAMP.tar.gz
mkdir -p ~/restore-temp
tar -xzf $BACKUP_FILE -C ~/restore-temp/

# Review files before restoring
ls -la ~/restore-temp/

# Restore configuration files
sudo cp ~/restore-temp/backend-config-gsi-test-server.yaml /root/dataharbor/config/
sudo cp ~/restore-temp/dataharbor-backend.service /etc/systemd/system/
sudo cp ~/restore-temp/dataharbor.conf /etc/nginx/conf.d/
sudo cp ~/restore-temp/config.json /usr/share/dataharbor-frontend/

# Reload and restart services
sudo systemctl daemon-reload
sudo systemctl restart dataharbor-backend
sudo systemctl reload nginx

# Verify
sudo systemctl status dataharbor-backend nginx
curl -k https://localhost/api/health
```

---

## Security Best Practices

### SSL Certificate Management

```bash
# Check certificate expiration
openssl x509 -in /root/dataharbor/config/cert/server.crt -noout -dates

# Verify certificate and key match
openssl x509 -noout -modulus -in /root/dataharbor/config/cert/server.crt | openssl md5
openssl rsa -noout -modulus -in /root/dataharbor/config/cert/server.key | openssl md5
# Both should output the same hash

# Set proper permissions
sudo chmod 644 /root/dataharbor/config/cert/server.crt
sudo chmod 600 /root/dataharbor/config/cert/server.key
sudo chown root:root /root/dataharbor/config/cert/*
```

### Secure Configuration Files

```bash
# Restrict access to sensitive configs
sudo chmod 600 /root/dataharbor/config/backend-config-gsi-test-server.yaml

# Verify no secrets in world-readable files
sudo find /root/dataharbor -type f -perm /o=r -ls
```

### Update OIDC Secrets

```bash
# Edit backend config
sudo nano /root/dataharbor/config/backend-config-gsi-test-server.yaml

# Update:
# - auth.oidc.client_secret
# - auth.oidc.session_secret (should be random string)

# Restart backend
sudo systemctl restart dataharbor-backend
```

---

## Monitoring and Alerting

### Set Up Log Monitoring

```bash
# Watch for errors in real-time
sudo journalctl -u dataharbor-backend -f | grep -i error

# Count errors in last hour
sudo journalctl -u dataharbor-backend --since "1 hour ago" | grep -c ERROR

# Email on service failure (optional)
# Add to /etc/systemd/system/dataharbor-backend.service:
# [Unit]
# OnFailure=status-email@%n.service
```

### Performance Monitoring

```bash
# Monitor backend process
top -p $(pgrep dataharbor-backend)

# Check memory usage
ps aux | grep dataharbor-backend

# Monitor network connections
sudo ss -tnp | grep dataharbor-backend

# Check disk space for logs
df -h /var/log
du -sh /var/log/dataharbor/
```

---

## Support and Additional Resources

### Getting Help

**For GSI-specific issues:**

- XRootD connectivity issues → Contact GSI storage team
- Keycloak/OIDC configuration → Contact GSI identity management team  
- Network/firewall issues → Contact GSI network operations team

**For DataHarbor issues:**

- See main **[DEPLOYMENT.md](./DEPLOYMENT.md)** - General deployment guide
- See **[TROUBLESHOOTING.md](./TROUBLESHOOTING.md)** - Detailed troubleshooting
- See **[BACKEND_CONFIGURATION.md](./BACKEND_CONFIGURATION.md)** - Backend config reference
- See **[FRONTEND_CONFIGURATION.md](./FRONTEND_CONFIGURATION.md)** - Frontend config reference
- GitHub Issues: [https://github.com/AnarManafov/dataharbor/issues](https://github.com/AnarManafov/dataharbor/issues)

### Document Information

**Version**: 2.0 (HTTPS Production Configuration)  
**Last Updated**: October 2025  
**Tested On**: RHEL/AlmaLinux at GSI (punch2.gsi.de)  
**Configuration**: HTTPS on port 443, Backend on port 22000, XRootD on ports 80/1094

---

### Change Log

**Version 2.0 (October 2025)**:

- ✅ Updated to HTTPS configuration (port 443)
- ✅ Removed port 80 HTTP redirect (conflicts with XRootD)
- ✅ Changed from port 8080 to port 443 for frontend
- ✅ Added clear separation between initial setup and version updates
- ✅ Updated CORS configuration for HTTPS origins
- ✅ Improved troubleshooting section
- ✅ Added external access testing procedures
- ✅ Production-tested configuration

**Version 1.0 (Initial)**:

- Initial port 8080 HTTP configuration
- Basic setup instructions

---

**End of GSI Deployment Guide**
