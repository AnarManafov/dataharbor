# DataHarbor RPM Packaging

This directory contains the RPM packaging infrastructure for DataHarbor backend and frontend.

## Directory Structure

```text
packaging/
├── build_rpm.py                          # RPM build script
├── generate_changelog.py                 # Changelog generator
├── dataharbor-backend.spec              # Backend RPM spec file
├── dataharbor-frontend.spec             # Frontend RPM spec file
├── systemd/
│   └── dataharbor-backend.service       # SystemD service template
├── nginx/
│   ├── nginx-http-simple.conf           # Simple HTTP config
│   ├── nginx-https-proxy.conf           # HTTPS with reverse proxy (recommended)
│   └── nginx-gsi.conf                   # GSI-specific configuration
├── config/
│   ├── application.yaml.example         # Backend config example
│   └── config.json.example              # Frontend config example
└── README.md                            # This file
```

## Building RPM Packages

### Prerequisites

- Python 3
- `rpmbuild` installed
- `npm` (for frontend build)
- `go` (for backend build)

### Build Both Packages

```bash
# From repository root
python3 packaging/build_rpm.py

# Or specify version
python3 packaging/build_rpm.py -v 1.2.0
```

### Build Specific Package

```bash
# Backend only
python3 packaging/build_rpm.py --backend

# Frontend only
python3 packaging/build_rpm.py --frontend
```

### Output Location

Built RPMs are placed in: `/tmp/all-rpms/`

```bash
ls -la /tmp/all-rpms/
# dataharbor-backend-1.2.0-1.el8.x86_64.rpm
# dataharbor-frontend-1.2.0-1.el8.noarch.rpm
```

## What Gets Installed

### Backend Package (`dataharbor-backend`)

| File/Directory   | Location                                             | Description            |
| ---------------- | ---------------------------------------------------- | ---------------------- |
| Binary           | `/usr/local/bin/dataharbor-backend`                  | Main executable        |
| SystemD Service  | `/usr/lib/systemd/system/dataharbor-backend.service` | Service file           |
| Config Example   | `/etc/dataharbor/application.yaml.example`           | Configuration template |
| Config Directory | `/etc/dataharbor/`                                   | Configuration location |
| Log Directory    | `/var/log/dataharbor/`                               | Default log location   |
| Documentation    | `/usr/share/doc/dataharbor-backend/`                 | Quick reference        |

### Frontend Package (`dataharbor-frontend`)

| File/Directory       | Location                                             | Description              |
| -------------------- | ---------------------------------------------------- | ------------------------ |
| Frontend Files       | `/usr/share/dataharbor-frontend/`                    | Vue.js SPA files         |
| Config Example       | `/usr/share/dataharbor-frontend/config.json.example` | Frontend config template |
| Nginx Templates      | `/etc/dataharbor-frontend/nginx/templates/`          | Multiple nginx configs   |
| Default Nginx Config | `/etc/dataharbor-frontend/nginx/nginx.conf`          | Simple HTTP config       |

## Post-Installation Setup

### Backend Setup (Quick)

```bash
# 1. Create config from example
sudo cp /etc/dataharbor/application.yaml.example /etc/dataharbor/application.yaml

# 2. Edit config (update secrets, SSL paths, OIDC settings)
sudo nano /etc/dataharbor/application.yaml

# 3. Enable and start service
sudo systemctl enable --now dataharbor-backend

# 4. Check status
sudo systemctl status dataharbor-backend
```

### Frontend Setup (Quick)

```bash
# 1. Choose nginx template and copy
sudo cp /etc/dataharbor-frontend/nginx/templates/nginx-https-proxy.conf \
        /etc/nginx/conf.d/dataharbor.conf

# 2. Edit for your environment
sudo nano /etc/nginx/conf.d/dataharbor.conf

# 3. Create frontend config
sudo cp /usr/share/dataharbor-frontend/config.json.example \
        /usr/share/dataharbor-frontend/config.json

# 4. Edit frontend config
sudo nano /usr/share/dataharbor-frontend/config.json

# 5. Test and reload nginx
sudo nginx -t && sudo systemctl reload nginx
```

## Nginx Configuration Templates

### 1. Simple HTTP (`nginx-http-simple.conf`)

**Use case:** Development, testing, non-production environments

**Features:**

- HTTP on port 80
- No SSL
- No reverse proxy
- Frontend accesses backend directly

**Frontend config.json:**

```json
{
  "apiBaseUrl": "https://backend-server:8081/api"
}
```

### 2. HTTPS with Reverse Proxy (`nginx-https-proxy.conf`)

**Use case:** Production deployments (recommended)

**Features:**

- HTTPS on port 443
- SSL/TLS enabled
- Reverse proxy to backend
- Security headers
- HTTP to HTTPS redirect

**Frontend config.json:**

```json
{
  "apiBaseUrl": "/api"
}
```

**Customization needed:**

- Replace `your-hostname.example.com` with actual domain
- Update SSL certificate paths
- Adjust backend port if not 8081

### 3. GSI-Specific (`nginx-gsi.conf`)

**Use case:** GSI servers with XRootD on port 80

**Features:**

- HTTPS on port 443 only (no port 80)
- Backend on port 22000
- SSL with GEANT CA certificates
- Reverse proxy to backend

**Frontend config.json:**

```json
{
  "apiBaseUrl": "/api"
}
```

**Customization needed:**

- Replace `punch2.gsi.de` with actual server hostname
- Update SSL certificate paths
- Verify backend port (22000)

## SystemD Service Features

The included systemd service file provides:

- **Automatic restart** on failure
- **Environment variable support** for config path
- **Security hardening** (NoNewPrivileges, PrivateTmp)
- **Resource limits** configured
- **Journal logging** integration

### Override Config Path

Create a drop-in file:

```bash
sudo systemctl edit dataharbor-backend
```

Add:

```ini
[Service]
Environment="CONFIG_FILE=/custom/path/config.yaml"
```

## Customization Examples

### Using Custom Config Location

```bash
# Option 1: Override with drop-in file
sudo systemctl edit dataharbor-backend
# Add: Environment="CONFIG_FILE=/opt/dataharbor/config.yaml"

# Option 2: Copy and modify service file
sudo cp /usr/lib/systemd/system/dataharbor-backend.service \
        /etc/systemd/system/dataharbor-backend.service
sudo nano /etc/systemd/system/dataharbor-backend.service
# Edit ExecStart line

sudo systemctl daemon-reload
sudo systemctl restart dataharbor-backend
```

### Using Multiple Instances

```bash
# Copy service file with instance name
sudo cp /usr/lib/systemd/system/dataharbor-backend.service \
        /etc/systemd/system/dataharbor-backend@instance1.service

# Edit to use different config
sudo nano /etc/systemd/system/dataharbor-backend@instance1.service

# Start instance
sudo systemctl enable --now dataharbor-backend@instance1
```

## Documentation References

- [DEPLOYMENT.md](../docs/DEPLOYMENT.md) - General deployment guide
- [DEPLOYMENT_GSI.md](../docs/DEPLOYMENT_GSI.md) - GSI-specific deployment
- [BACKEND_CONFIGURATION.md](../docs/BACKEND_CONFIGURATION.md) - Backend config reference
- [FRONTEND_CONFIGURATION.md](../docs/FRONTEND_CONFIGURATION.md) - Frontend config reference

## Troubleshooting

### RPM Build Fails

```bash
# Check build directory
ls -la ~/rpmbuild/

# Check for missing dependencies
rpm -q rpmbuild rpm-build

# View detailed build logs
less ~/rpmbuild/BUILD/*.log
```

### Service Won't Start

```bash
# Check service status
sudo systemctl status dataharbor-backend

# View logs
sudo journalctl -u dataharbor-backend -n 100

# Check config file
sudo cat /etc/dataharbor/application.yaml

# Test config manually
/usr/local/bin/dataharbor-backend --config=/etc/dataharbor/application.yaml
```

### Nginx Config Issues

```bash
# Test nginx config
sudo nginx -t

# Check nginx error log
sudo tail -f /var/log/nginx/dataharbor-frontend-error.log

# Verify templates exist
ls -la /etc/dataharbor-frontend/nginx/templates/
```

## Updating Packages

When updating to a new version:

1. **Backend**: RPM update preserves `/etc/dataharbor/` configs
2. **Frontend**: May need to re-copy `config.json` after update
3. **Service files**: Automatically updated by RPM

```bash
# Update backend
sudo rpm -Uvh dataharbor-backend-*.rpm
sudo systemctl daemon-reload
sudo systemctl restart dataharbor-backend

# Update frontend
sudo rpm -Uvh dataharbor-frontend-*.rpm
# Re-copy config if needed
sudo cp /path/to/your/config.json /usr/share/dataharbor-frontend/
sudo systemctl reload nginx
```

## Version Management

The version is managed in `package.json` at repository root:

```json
{
  "version": "1.2.0"
}
```

This version is automatically used by `build_rpm.py` unless overridden with `-v` flag.

## Contributing

When adding new templates or modifying packaging:

1. Update appropriate spec file
2. Add template to `packaging/nginx/` or `packaging/config/`
3. Update `build_rpm.py` to copy new files
4. Test RPM build locally
5. Update this README

## License

GPL-3.0 - See [LICENSE](../LICENSE) file for details
