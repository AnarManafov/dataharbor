# XRootD ZTN Protocol Configuration Guide

**Purpose**: Enable OAuth token authentication on native XRootD protocol (port 1094)

**Date**: October 2025

---

## Problem

The XRootD server currently only has HTTP authentication configured. The native protocol (port 1094) has **no authentication** (missing `sec.protocol` directive), causing "Anonymous client" errors when the DataHarbor backend attempts to connect.

---

## Solution: Enable ZTN (Zero-Trust Networking) Protocol

ZTN enables token-based authentication on the native XRootD protocol using the same SciTokens infrastructure already configured for HTTP.

**Requirements**:

- TLS encryption (mandatory for ZTN)
- Existing scitokens.cfg and mapfile (already configured ✓)
- TLS certificates (self-signed or real certificates)

---

## Step 1: Verify/Create TLS Certificates

### Option A: Using Self-Signed Certificates (Development/Testing)

If you don't have real certificates, create self-signed ones:

```bash
# Create self-signed certificate valid for 365 days
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout /etc/xrootd/hostkey.pem \
  -out /etc/xrootd/hostcert.pem \
  -subj "/C=DE/ST=Hesse/L=Darmstadt/O=GSI/CN=punch2.gsi.de"

# Set correct permissions
chown xrootd:xrootd /etc/xrootd/hostcert.pem /etc/xrootd/hostkey.pem
chmod 644 /etc/xrootd/hostcert.pem
chmod 600 /etc/xrootd/hostkey.pem

# Create CA certificate directory (use self-signed cert as CA)
mkdir -p /etc/grid-security/certificates
cp /etc/xrootd/hostcert.pem /etc/grid-security/certificates/
c_rehash /etc/grid-security/certificates/
```

**Note**: Self-signed certificates work fine for ZTN protocol. Clients connecting to port 1094 will need to trust this certificate or disable TLS verification.

### Option B: Using Real Certificates (Production)

If you have real certificates from your GSI CA or Let's Encrypt:

```bash
# Verify certificates exist
ls -la /etc/xrootd/hostcert.pem /etc/xrootd/hostkey.pem
ls -la /etc/grid-security/certificates/

# Set correct permissions
chown xrootd:xrootd /etc/xrootd/hostcert.pem /etc/xrootd/hostkey.pem
chmod 644 /etc/xrootd/hostcert.pem
chmod 600 /etc/xrootd/hostkey.pem
```

---

## Step 2: Update XRootD Configuration

Edit `/etc/xrootd/xrootd-http.cfg` and add the following lines:

```bash
# ============================================
# TLS Configuration (Required for ZTN)
# ============================================
xrd.tlsca certdir /etc/grid-security/certificates
xrd.tls /etc/xrootd/hostcert.pem /etc/xrootd/hostkey.pem

# ============================================
# ZTN Protocol for Native Port 1094
# ============================================
sec.protocol ztn -tokenlib libXrdAccSciTokens.so
sec.protbind * only ztn
```

**Important**: Keep all existing configuration directives:
- `ofs.authorize`
- `ofs.authlib ++ libXrdAccSciTokens.so config=/etc/xrootd/scitokens.cfg`
- `ofs.osslib ++ libXrdMultiuser.so`
- `http.header2cgi Authorization authz`

### Complete Configuration Example

Here's how `/etc/xrootd/xrootd-http.cfg` should look after changes:

```bash
# Export filesystem
all.export / r/w

# HTTP Protocol (port 80)
xrd.protocol XrdHttp:80 libXrdHttp.so
http.header2cgi Authorization authz

# TLS Configuration (Required for ZTN)
xrd.tlsca certdir /etc/grid-security/certificates
xrd.tls /etc/xrootd/hostcert.pem /etc/xrootd/hostkey.pem

# Native Protocol Authentication (ZTN for port 1094)
sec.protocol ztn -tokenlib libXrdAccSciTokens.so
sec.protbind * only ztn

# Authorization and Token Validation
ofs.authorize
ofs.authlib ++ libXrdAccSciTokens.so config=/etc/xrootd/scitokens.cfg

# Multiuser Support
ofs.osslib ++ libXrdMultiuser.so

# Debug Token Processing
scitokens.trace all

# Include additional config files
continue /etc/xrootd/config.d/
```

---

## Step 3: Verify Configuration Files (No Changes Needed)

These files should **remain unchanged**:

### `/etc/xrootd/scitokens.cfg`
```ini
[Global]
audience = ...

[Issuer wl]
issuer = https://id.gsi.de/realms/wl
base_path = /
map_subject = True
default_user = xrootd
name_mapfile = /etc/xrootd/mapfile
```

### `/etc/xrootd/mapfile`
```json
[
  {"sub": "user1", "result": "mappeduser1"},
  {"sub": "user2", "result": "mappeduser2"},
  ...
]
```

All 200+ existing user mappings will work with ZTN automatically.

**Note**: Keep actual usernames/mappings confidential.

---

## Step 4: Restart XRootD Service

```bash
# Restart XRootD
systemctl restart xrootd@http

# Or if using different service name:
# systemctl restart xrootd

# Check service status
systemctl status xrootd@http
```

---

## Step 5: Verify Configuration

### 5.1. Check TLS is Active

```bash
# Verify port 1094 is listening
netstat -tlnp | grep 1094

# Test TLS connection
openssl s_client -connect localhost:1094 -showcerts
```

Expected: TLS handshake succeeds, shows certificate details.

### 5.2. Check XRootD Logs

```bash
tail -f /var/log/xrootd/xrootd.log
```

Look for:
- `sec.protocol ztn` initialization messages
- TLS certificate loaded successfully
- No error messages about missing libraries

---

## How ZTN Token Discovery Works

When a client connects to port 1094 with ZTN, XRootD looks for tokens in this order:

1. **`BEARER_TOKEN`** environment variable
2. **`BEARER_TOKEN_FILE`** environment variable → reads file contents  
3. **`$XDG_RUNTIME_DIR/bt_u{euid}`** (if XDG_RUNTIME_DIR is set)
4. **`/tmp/bt_u{euid}`** (fallback)

The **DataHarbor backend** will be updated to provide tokens via one of these methods.

## References

- XRootD ZTN Protocol Documentation: CERN Indico Event 1483930
- XRootD Security Configuration: https://xrootd.web.cern.ch/doc/dev56/sec_config.htm
- SciTokens Library: https://github.com/scitokens/xrootd-scitokens

---

**Configuration prepared for**: punch2.gsi.de (140.181.3.31)  
**XRootD Version**: 5.x (verify with `xrootd -v`)  
**Date**: October 2025
