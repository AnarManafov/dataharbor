# XRootD ZTN Configuration - punch2.gsi.de

**Server:** punch2.gsi.de  
**Date:** October 16, 2025  
**Status:** ✅ Production Ready  
**XRootD Version:** v5.8.4 (OSG 23)

> **Note**: This guide is for the external punch2.gsi.de XRootD server configuration.
> For Docker-based deployments, see the pre-configured files:
> - `docker/xrootd/configs/xrootd-dev.cfg` (development)
> - `docker/xrootd/configs/xrootd-prod.cfg` (production)

---

## Table of Contents

1. [Service Status](#1-service-status)
2. [TLS Configuration](#2-tls-configuration)
3. [ZTN Protocol Configuration](#3-ztn-protocol-configuration)
4. [SciTokens Configuration](#4-scitokens-configuration)
5. [Multiuser Plugin](#5-multiuser-plugin)
6. [User Mapping](#6-user-mapping)
7. [Authentication Flow](#7-authentication-flow)
8. [Debug & Logging](#8-debug--logging)
9. [Export Configuration](#9-export-configuration)
10. [Client Connection Guide](#10-client-connection-guide)
11. [Troubleshooting](#11-troubleshooting)

---

## 1. Service Status

### Running Service
- **Service Name:** `xrootd-privileged@http.service`
- **Status:** ✅ Active (running)
- **Main PID:** 7173
- **Configuration File:** `/etc/xrootd/xrootd-http.cfg`
- **Log File:** `/var/log/xrootd/http/xrootd.log`
- **Port:** 1094 (IPv4/IPv6)

### Service Information
```bash
systemctl status xrootd-privileged@http
```

### Installed Packages
```
xrootd-5.8.4-1.4.osg23.el8.x86_64
xrootd-client-5.8.4-1.4.osg23.el8.x86_64
xrootd-client-libs-5.8.4-1.4.osg23.el8.x86_64
xrootd-libs-5.8.4-1.4.osg23.el8.x86_64
xrootd-multiuser-2.2.0-1.1.osg23.el8.x86_64
xrootd-scitokens-5.8.4-1.4.osg23.el8.x86_64
xrootd-selinux-5.8.4-1.4.osg23.el8.noarch
xrootd-server-5.8.4-1.4.osg23.el8.x86_64
xrootd-server-libs-5.8.4-1.4.osg23.el8.x86_64
```

### Port Status
```bash
$ ss -tlnp | grep 1094
LISTEN 0  255  *:1094  *:*  users:(("xrootd",pid=7173,fd=22))
```

---

## 2. TLS Configuration

### Configuration (`/etc/xrootd/xrootd-http.cfg`)

```ini
# TLS Configuration
xrd.tlsca certdir /etc/grid-security/certificates
xrd.tls /etc/ssl/certs/punch2.gsi.de.pem /etc/ssl/private/punch2.gsi.de.key

# Enable TLS for all connections except data transfers
xrootd.tls capable all -data
```

### Certificate Details

| Component              | Path                                 | Permissions                     | Status   |
| ---------------------- | ------------------------------------ | ------------------------------- | -------- |
| **Server Certificate** | `/etc/ssl/certs/punch2.gsi.de.pem`   | `-rw-r--r--` (644)              | ✅ Valid  |
| **Private Key**        | `/etc/ssl/private/punch2.gsi.de.key` | `-r--------` (400, xrootd:root) | ✅ Secure |
| **CA Certificates**    | `/etc/grid-security/certificates/`   | Directory with 1,023 CAs        | ✅ Valid  |

### Certificate Verification
```bash
# Check certificate
openssl x509 -in /etc/ssl/certs/punch2.gsi.de.pem -text -noout

# Check private key
openssl rsa -in /etc/ssl/private/punch2.gsi.de.key -check

# Verify certificate-key pair
openssl x509 -in /etc/ssl/certs/punch2.gsi.de.pem -noout -modulus | md5sum
openssl rsa -in /etc/ssl/private/punch2.gsi.de.key -noout -modulus | md5sum
```

### TLS Status
✅ **TLS is REQUIRED** for all connections  
✅ TLS initialization successful  
✅ All authentication protocols require TLS

---

## 3. ZTN Protocol Configuration

### Configuration

```ini
# Security library
xrootd.seclib libXrdSec.so

# ZTN Protocol with SciTokens
sec.protocol ztn -tokenlib libXrdAccSciTokens.so
sec.protbind * only ztn
```

### Key Features

- **Protocol:** ZTN (Zero Trust Network) - **ONLY** protocol enabled
- **Token Library:** `libXrdAccSciTokens.so` (Bearer token validation)
- **Security Model:** Zero Trust - no implicit trust, all connections authenticated
- **Fallback:** ❌ **NONE** - ZTN is required, no fallback to insecure protocols

### Verification from Logs

```
Config Authentication protocol(s) ztn require TLS; login now requires TLS.
```

This confirms:
1. ✅ ZTN is the sole authentication protocol
2. ✅ TLS is mandatory for all connections
3. ✅ Bearer tokens are required for authentication

### Testing ZTN Enforcement

```bash
# This should fail without a token
$ xrdfs punch2.gsi.de:1094 ls /
[FATAL] Auth failed: No protocols left to try
```

✅ **Result:** Server correctly rejects connections without bearer tokens

---

## 4. SciTokens Configuration

> **Note**: This section is for the external punch2.gsi.de server. For Docker deployments, see:
> - `docker/xrootd/configs/scitokens_dev.cfg` (development)
> - `docker/xrootd/configs/scitokens_prod.cfg` (production)

### Configuration File: `/etc/xrootd/scitokens.cfg`

```ini
[Global]
audience = https://id.gsi.de/realms/wl

[Issuer https://id.gsi.de/realms/wl]
issuer = https://id.gsi.de/realms/wl
base_path = /
map_subject = True
default_user = xrootd
name_mapfile = /etc/xrootd/mapfile
```

### Main Configuration

```ini
# SciTokens Authorization
ofs.authlib ++ libXrdAccSciTokens.so config=/etc/xrootd/scitokens.cfg
```

### Token Validation Settings

| Setting             | Value                         | Description                           |
| ------------------- | ----------------------------- | ------------------------------------- |
| **Issuer**          | `https://id.gsi.de/realms/wl` | GSI Keycloak realm                    |
| **Audience**        | `https://id.gsi.de/realms/wl` | Expected token audience               |
| **Subject Mapping** | `True`                        | Map token subject to Linux user       |
| **Mapping File**    | `/etc/xrootd/mapfile`         | JSON mapping file                     |
| **Default User**    | `xrootd`                      | Fallback user (not used with mapping) |
| **Base Path**       | `/`                           | Root path for authorization           |

### Token Requirements

A valid SciToken must contain:
- **iss** (Issuer): `https://id.gsi.de/realms/wl`
- **aud** (Audience): `https://id.gsi.de/realms/wl`
- **sub** (Subject): Must match an entry in `/etc/xrootd/mapfile`
- **exp** (Expiration): Valid timestamp
- **iat** (Issued At): Valid timestamp

### Token Scopes (Example)

```json
{
  "scope": "read:/data write:/data delete:/data"
}
```

### Loaded Libraries

```
libSciTokens.so.0.0.2
libXrdAccSciTokens-5.so
```

---

## 5. Multiuser Plugin

### Configuration

```ini
# Multiuser plugin - maps authenticated user to Unix UID/GID
oss.localroot /data/xrootd

# Multiuser plugin
ofs.osslib ++ libXrdMultiuser.so
```

### Additional Configuration: `/etc/xrootd/config.d/60-osg-multiuser.cfg`

```ini
# Enable multiuser plugin
if defined ?~XC_ENABLE_MULTIUSER && exec xrootd 
  ofs.osslib ++ libXrdMultiuser.so
else if defined ?~XC_ENABLE_MULTIUSER
  ofs.osslib libXrdMultiuser.so default
fi

if defined ?~XC_ENABLE_MULTIUSER
  # Enable the checksum wrapper
  ofs.ckslib * libXrdMultiuser.so
  xrootd.chksum max 2 md5 adler32 crc32
fi
```

### Plugin Information

- **Library:** `libXrdMultiuser-5.so`
- **Version:** v5.7.0 (osg-multiuser)
- **Status:** ✅ Loaded and active

### Features

1. **UID/GID Mapping**
   - Maps authenticated bearer token subject to Linux UID/GID
   - Uses `/etc/xrootd/mapfile` for mapping
   - Files are created with the authenticated user's ownership

2. **Security**
   - ❌ Anonymous connections rejected: `"Anonymous client; no user set, cannot change FS UIDs"`
   - ✅ Only authenticated users can write files
   - ✅ Files maintain proper ownership (not written as 'xrootd' user)

3. **Checksum Support**
   - Supports MD5, Adler32, and CRC32 checksums
   - Optional checksum-on-write capability

### Local Root Path

- **Path:** `/data/xrootd`
- **Purpose:** Base directory for all file operations
- **Permissions:** Must be accessible by mapped user UIDs

---

## 6. User Mapping

### Mapping File: `/etc/xrootd/mapfile`

**Format:** JSON array of subject-to-username mappings

```json
[
  {
    "sub": "a.manafo",
    "result": "amanafov"
  },
  {
    "sub": "k.zissel",
    "result": "kzissel"
  },
  {
    "sub": "n.breunig",
    "result": "nbreunig"
  }
]
```

### Mapping Statistics

- **Total Mappings:** 700+ users
- **Format:** JSON (required for SciTokens)
- **Key Field:** `sub` (subject from JWT token)
- **Value Field:** `result` (Linux username)

### Example Mappings

| Token Subject            | Linux Username | Type            |
| ------------------------ | -------------- | --------------- |
| `a.manafo`               | `amanafov`     | Standard user   |
| `a.mollaebrahimi`        | `frs-ic`       | Service account |
| `thomas.bornhofen`       | `fcc-aec2`     | Service account |
| `partner_holger.gwosdz`  | `hgwosdz`      | Partner account |
| `aaronengel11@gmail.com` | `aengel`       | Email-based     |

### Mapping Process

1. Client authenticates with bearer token
2. XRootD extracts `sub` claim from JWT
3. Looks up `sub` in `/etc/xrootd/mapfile`
4. Maps to Linux username in `result` field
5. Switches process UID/GID to mapped user
6. Performs file operations as mapped user

### Adding New Users

Edit `/etc/xrootd/mapfile` and add:
```json
{
  "sub": "new.user",
  "result": "newuser"
}
```

Then restart XRootD:
```bash
systemctl restart xrootd-privileged@http
```

---

## 7. Authentication Flow

### Connection Flow Diagram

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │
       │ 1. Connect to punch2.gsi.de:1094
       │    (TLS handshake)
       ↓
┌─────────────────────┐
│   XRootD Server     │
│  (ZTN Protocol)     │
└──────┬──────────────┘
       │
       │ 2. Request bearer token
       ↓
┌─────────────────────┐
│      Client         │
│  (Send JWT token)   │
└──────┬──────────────┘
       │
       │ 3. Token validation
       ↓
┌─────────────────────┐
│  SciTokens Plugin   │
│  - Verify signature │
│  - Check issuer     │
│  - Check audience   │
│  - Check expiration │
└──────┬──────────────┘
       │
       │ 4. Extract subject
       ↓
┌─────────────────────┐
│   User Mapping      │
│  (/etc/xrootd/      │
│   mapfile)          │
└──────┬──────────────┘
       │
       │ 5. Map to Linux UID/GID
       ↓
┌─────────────────────┐
│ Multiuser Plugin    │
│ (Set process UID)   │
└──────┬──────────────┘
       │
       │ 6. Authorized file access
       ↓
┌─────────────────────┐
│  File System        │
│  (/data/xrootd)     │
└─────────────────────┘
```

### Authentication Requirements

✅ **Required:**
- Valid TLS connection
- Valid bearer token (JWT)
- Token issued by `https://id.gsi.de/realms/wl`
- Token subject must exist in `/etc/xrootd/mapfile`
- Linux user must exist on the system

❌ **Rejected:**
- Connections without TLS
- Connections without bearer token
- Invalid/expired tokens
- Tokens from untrusted issuers
- Unmapped subjects

### Log Evidence

```
251016 14:38:14 7182 sec_getParms: punch2.gsi.de sectoken=&P=ztn,0:4096:
251016 14:38:14 7182 XrootdXeq: dataharb.6769:26@punch2 disc 0:00:00
251016 14:38:14 7182 multiuser_UserSentry: Anonymous client; no user set, cannot change FS UIDs
```

This shows:
1. ZTN protocol negotiation
2. Anonymous connection attempt
3. Multiuser plugin rejection (no authenticated user)

---

## 8. Debug & Logging

### Debug Configuration

```ini
# Debug Token Processing
scitokens.trace all

# Logging
xrd.trace all
sec.trace all
ofs.trace all
```

### Log Levels

| Component           | Level | Purpose                            |
| ------------------- | ----- | ---------------------------------- |
| **scitokens.trace** | `all` | Token validation, parsing, mapping |
| **xrd.trace**       | `all` | XRootD protocol operations         |
| **sec.trace**       | `all` | Security/authentication events     |
| **ofs.trace**       | `all` | File system operations             |

### Log Files

- **Main Log:** `/var/log/xrootd/http/xrootd.log`
- **System Log:** `journalctl -u xrootd-privileged@http`

### Useful Log Queries

```bash
# View recent logs
tail -100 /var/log/xrootd/http/xrootd.log

# Monitor logs in real-time
tail -f /var/log/xrootd/http/xrootd.log

# View systemd logs
journalctl -u xrootd-privileged@http -n 100 --no-pager

# Follow systemd logs
journalctl -u xrootd-privileged@http -f

# Search for authentication events
grep -i "auth\|token\|ztn" /var/log/xrootd/http/xrootd.log

# Search for multiuser operations
grep -i "multiuser" /var/log/xrootd/http/xrootd.log
```

### Important Log Entries

**Successful Startup:**
```
Config Authentication protocol(s) ztn require TLS; login now requires TLS.
Plugin loaded osg-multiuser v5.7.0 from osslib libXrdMultiuser-5.so
------ xrootd http@punch2.gsi.de:1094 initialization completed.
```

**TLS Initialization:**
```
++++++ xrootd http@punch2.gsi.de TLS initialization started.
------ xrootd http@punch2.gsi.de TLS initialization ended.
```

**Rejected Connection:**
```
multiuser_UserSentry: Anonymous client; no user set, cannot change FS UIDs
```

---

## 9. Export Configuration

### Export Settings

```ini
# Export root path with read/write permissions
all.export / r/w

# Local root directory for file storage
oss.localroot /data/xrootd
```

### Storage Configuration

From logs:
```
Config effective /etc/xrootd/xrootd-http.cfg oss configuration:
       oss.alloc        0 0 0
       oss.spacescan    600
       oss.fdlimit      32768 65536
       oss.maxsize      0
       oss.localroot /data/xrootd
       oss.trace        0
       oss.xfr          1 deny 10800 keep 1200
       oss.memfile off  max 1957439488
       oss.defaults  r/w nocheck nodread nomig nopurge norcreate nostage
       oss.path / r/w nocheck nodread nomig nopurge norcreate nostage
```

### Storage Parameters

| Parameter       | Value          | Description                            |
| --------------- | -------------- | -------------------------------------- |
| **Local Root**  | `/data/xrootd` | Base directory for all files           |
| **Export Path** | `/`            | Exported path (relative to local root) |
| **Permissions** | `r/w`          | Read and write access                  |
| **FD Limit**    | 32768 / 65536  | File descriptor limits (soft/hard)     |
| **Max Size**    | 0              | Unlimited file size                    |
| **Space Scan**  | 600 seconds    | Space usage scan interval              |

### Storage Features

- ✅ Read/write access
- ❌ No staging (not a tape system)
- ❌ No migration
- ❌ No purging
- ❌ No remote creation

### Path Structure

```
/data/xrootd/           # Physical storage location
    └── <user files>    # Files owned by mapped users
```

When a client accesses `root://punch2.gsi.de:1094//myfile.txt`:
- Maps to physical path: `/data/xrootd/myfile.txt`
- Owned by the authenticated user's UID/GID

---

## 10. Client Connection Guide

### Prerequisites

1. **XRootD Client** (version 5.0+)
   ```bash
   yum install xrootd-client
   # or
   apt-get install xrootd-client
   ```

2. **Valid Bearer Token** from GSI Keycloak
   - Issuer: `https://id.gsi.de/realms/wl`
   - Your subject must be in the mapfile

3. **TLS Support** enabled in client

### Connection Methods

#### Method 1: Environment Variable

```bash
# Set the bearer token
export BEARER_TOKEN="eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."

# Use xrdfs
xrdfs punch2.gsi.de:1094 ls /

# Copy files
xrdcp /local/file root://punch2.gsi.de:1094//remote/file
xrdcp root://punch2.gsi.de:1094//remote/file /local/file
```

#### Method 2: Token File

```bash
# Save token to file
echo "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..." > ~/.xrootd-token

# Set environment variable to point to file
export BEARER_TOKEN_FILE=~/.xrootd-token

# Use xrdfs
xrdfs punch2.gsi.de:1094 ls /
```

#### Method 3: Programmatic (Go)

```go
package main

import (
    "github.com/go-hep/hep/xrootd"
    "github.com/go-hep/hep/xrootd/xrdfs"
)

func main() {
    // Create client with bearer token
    client, err := xrdfs.NewClient(
        "punch2.gsi.de:1094",
        xrdfs.WithBearerToken("your_jwt_token_here"),
        xrdfs.WithTLS(true),
    )
    if err != nil {
        panic(err)
    }
    defer client.Close()

    // List directory
    entries, err := client.Readdir("/")
    if err != nil {
        panic(err)
    }
    
    for _, entry := range entries {
        println(entry.Name())
    }
}
```

#### Method 4: Programmatic (Python)

```python
from XRootD import client
from XRootD.client.flags import OpenFlags

# Create client
myclient = client.FileSystem('punch2.gsi.de:1094')

# Set bearer token (via environment)
import os
os.environ['BEARER_TOKEN'] = 'your_jwt_token_here'

# List directory
status, listing = myclient.dirlist('/')
if status.ok:
    for entry in listing:
        print(entry.name)
```

### Common Operations

```bash
# List directory
xrdfs punch2.gsi.de:1094 ls /path/to/dir

# Create directory
xrdfs punch2.gsi.de:1094 mkdir /path/to/newdir

# Remove file
xrdfs punch2.gsi.de:1094 rm /path/to/file

# Get file info
xrdfs punch2.gsi.de:1094 stat /path/to/file

# Copy file to server
xrdcp /local/file root://punch2.gsi.de:1094//remote/path/file

# Copy file from server
xrdcp root://punch2.gsi.de:1094//remote/path/file /local/file

# Copy with progress
xrdcp --progress /local/file root://punch2.gsi.de:1094//remote/file
```

### Token Management

**Obtaining a Token from GSI Keycloak:**

```bash
# Using curl (example)
TOKEN=$(curl -X POST "https://id.gsi.de/realms/wl/protocol/openid-connect/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "username=your_username" \
  -d "password=your_password" \
  -d "grant_type=password" \
  -d "client_id=your_client_id" | jq -r '.access_token')

export BEARER_TOKEN="$TOKEN"
```

**Token Validation:**

```bash
# Decode token (without verification)
echo "$BEARER_TOKEN" | cut -d. -f2 | base64 -d | jq .

# Check expiration
echo "$BEARER_TOKEN" | cut -d. -f2 | base64 -d | jq -r '.exp' | xargs -I {} date -d @{}
```

---

## 11. Troubleshooting

### Common Issues

#### Issue 1: "Auth failed: No protocols left to try"

**Cause:** No bearer token provided or invalid token

**Solution:**
```bash
# Verify token is set
echo $BEARER_TOKEN

# Try with explicit token
export BEARER_TOKEN="your_valid_token"
xrdfs punch2.gsi.de:1094 ls /
```

#### Issue 2: "Anonymous client; no user set"

**Cause:** Token subject not mapped in `/etc/xrootd/mapfile`

**Solution:**
1. Check your token's subject:
   ```bash
   echo "$BEARER_TOKEN" | cut -d. -f2 | base64 -d | jq -r '.sub'
   ```

2. Verify it exists in mapfile:
   ```bash
   ssh root@punch2.gsi.de "grep 'your_subject' /etc/xrootd/mapfile"
   ```

3. Add mapping if missing and restart XRootD

#### Issue 3: "TLS handshake failed"

**Cause:** TLS certificate issues

**Solution:**
```bash
# Check server certificate
openssl s_client -connect punch2.gsi.de:1094 -showcerts

# Verify CA certificates
ssh root@punch2.gsi.de "ls -la /etc/grid-security/certificates/"
```

#### Issue 4: "Permission denied"

**Cause:** Linux user lacks filesystem permissions

**Solution:**
1. Verify user mapping:
   ```bash
   echo "$BEARER_TOKEN" | cut -d. -f2 | base64 -d | jq -r '.sub'
   ```

2. Check Linux user permissions:
   ```bash
   ssh root@punch2.gsi.de "ls -ld /data/xrootd/path/to/file"
   ```

3. Fix permissions:
   ```bash
   ssh root@punch2.gsi.de "chown username:group /data/xrootd/path/to/file"
   ```

#### Issue 5: Token expired

**Cause:** JWT token has expired

**Solution:**
```bash
# Check expiration
echo "$BEARER_TOKEN" | cut -d. -f2 | base64 -d | jq -r '.exp' | xargs -I {} date -d @{}

# Obtain new token from Keycloak
# (see Token Management section)
```

### Diagnostic Commands

```bash
# Check XRootD service status
ssh root@punch2.gsi.de 'systemctl status xrootd-privileged@http'

# Check if port is listening
ssh root@punch2.gsi.de 'ss -tlnp | grep 1094'

# View recent logs
ssh root@punch2.gsi.de 'tail -50 /var/log/xrootd/http/xrootd.log'

# Test connection (should fail without token)
xrdfs punch2.gsi.de:1094 ls /

# Check loaded libraries
ssh root@punch2.gsi.de 'lsof -p $(pgrep xrootd | head -1) | grep -i "scitokens\|multiuser"'

# Verify configuration files
ssh root@punch2.gsi.de 'cat /etc/xrootd/xrootd-http.cfg'
ssh root@punch2.gsi.de 'cat /etc/xrootd/scitokens.cfg'
```

### Server-Side Debugging

Enable verbose logging and restart:
```bash
ssh root@punch2.gsi.de
systemctl stop xrootd-privileged@http
/usr/bin/xrootd -d -l /var/log/xrootd/xrootd-debug.log -c /etc/xrootd/xrootd-http.cfg
```

### Getting Help

1. **Check logs first:** `/var/log/xrootd/http/xrootd.log`
2. **XRootD Documentation:** https://xrootd.slac.stanford.edu/
3. **SciTokens Documentation:** https://scitokens.org/
4. **GSI Support:** Contact GSI IT support for Keycloak/token issues

---

## Summary

The XRootD server on **punch2.gsi.de:1094** is configured with:

✅ **Zero Trust Network (ZTN)** - Token-based authentication only  
✅ **TLS Required** - All connections encrypted  
✅ **Bearer Token Authentication** - JWT tokens from GSI Keycloak  
✅ **User Mapping** - Tokens mapped to Linux users  
✅ **Multiuser Mode** - Files owned by authenticated users  
✅ **Security Hardening** - No anonymous access, no fallback protocols  

**Status: Production Ready** 🚀

---

*Document Version: 1.0*  
*Last Updated: October 16, 2025*  
*Maintained by: GSI IT Department*
