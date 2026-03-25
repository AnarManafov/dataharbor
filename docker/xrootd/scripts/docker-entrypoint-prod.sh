#!/bin/bash
# ============================================
# XRootD Production Entrypoint Script
# ============================================
# Validates production requirements:
# - TLS certificates mounted
# - Data directory mounted
# - User mapfile mounted
# - Proper permissions
#
# Does NOT create test users or data - production
# expects all configuration from host mounts.
# ============================================

set -e

# Colors for output (if terminal supports it)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m' # No Color
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_ok() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

echo ""
echo "========================================"
echo "  DataHarbor XRootD Server (Production)"
echo "========================================"
echo ""

# ==========================================
# Render configuration templates
# ==========================================
log_info "Rendering configuration from environment variables..."

# SciTokens config: substitute env vars (issuer/audience)
SCITOKENS_ISSUER="${SCITOKENS_ISSUER:-}"
SCITOKENS_AUDIENCE="${SCITOKENS_AUDIENCE:-$SCITOKENS_ISSUER}"
export SCITOKENS_ISSUER SCITOKENS_AUDIENCE

if [ -z "$SCITOKENS_ISSUER" ]; then
    log_error "SCITOKENS_ISSUER environment variable is required"
    log_error "Set it to your OIDC issuer URL (e.g., https://id.gsi.de/realms/wl)"
    exit 1
fi

SCITOKENS_TEMPLATE="/etc/xrootd/scitokens_prod.cfg"
SCITOKENS_RENDERED="/etc/xrootd/scitokens_rendered.cfg"
envsubst '${SCITOKENS_ISSUER} ${SCITOKENS_AUDIENCE}' < "$SCITOKENS_TEMPLATE" > "$SCITOKENS_RENDERED"
# Point the main config at the rendered file
sed -i "s|config=/etc/xrootd/scitokens_prod.cfg|config=/etc/xrootd/scitokens_rendered.cfg|" /etc/xrootd/xrootd-prod.cfg
chown xrootd:xrootd "$SCITOKENS_RENDERED"
log_ok "SciTokens config rendered (issuer: $SCITOKENS_ISSUER)"

# TLS CA verification: generate tlsca.cfg from env var
XROOTD_TLS_CA_VERIFY="${XROOTD_TLS_CA_VERIFY:-true}"
TLSCA_CFG="/etc/xrootd/tlsca.cfg"
if [ "$XROOTD_TLS_CA_VERIFY" = "false" ] || [ "$XROOTD_TLS_CA_VERIFY" = "0" ]; then
    echo "xrd.tlsca noverify" > "$TLSCA_CFG"
    log_warn "TLS CA verification DISABLED (testing mode)"
else
    echo "xrd.tlsca certdir /etc/grid-security/certificates" > "$TLSCA_CFG"
    log_ok "TLS CA verification enabled"
fi
chown xrootd:xrootd "$TLSCA_CFG"

# ==========================================
# Setup runtime directories
# ==========================================
log_info "Setting up runtime directories..."
mkdir -p /var/spool/xrootd /var/run/xrootd/certs /home/xrootd /var/log/xrootd

# Set ownership for XRootD runtime directories
chown -R xrootd:xrootd /var/spool/xrootd /home/xrootd /var/log/xrootd
chown xrootd:xrootd /var/run/xrootd
chmod 755 /var/spool/xrootd /var/run/xrootd /home/xrootd /var/log/xrootd

# Create grid-security directory if it doesn't exist (may be mounted read-only)
if [ ! -d /etc/grid-security/certificates ]; then
    mkdir -p /etc/grid-security/certificates 2>/dev/null || true
fi

# Make grid-security readable by xrootd user (skip if read-only mount)
chmod 755 /etc/grid-security /etc/grid-security/certificates 2>/dev/null || \
    log_warn "CA certificates directory is read-only (this is OK if mounted from host)"

log_ok "Runtime directories ready"

# ==========================================
# Copy and fix TLS certificate permissions
# ==========================================
# User-provided certs are mounted to /var/run/xrootd/certs-user (read-only)
# We copy them to /var/run/xrootd/certs with correct permissions
# This handles Docker Desktop (macOS/Windows) not preserving Unix file permissions
log_info "Setting up TLS certificates..."

USER_CERT="/var/run/xrootd/certs-user/hostcert.pem"
USER_KEY="/var/run/xrootd/certs-user/hostkey.pem"
CERT_FILE="/var/run/xrootd/certs/hostcert.pem"
KEY_FILE="/var/run/xrootd/certs/hostkey.pem"

if [ -f "$USER_CERT" ] && [ -f "$USER_KEY" ]; then
    cp "$USER_CERT" "$CERT_FILE"
    cp "$USER_KEY" "$KEY_FILE"
    chmod 644 "$CERT_FILE"
    chmod 600 "$KEY_FILE"
    chown xrootd:xrootd "$CERT_FILE" "$KEY_FILE"
    log_ok "TLS certificates copied with correct permissions"
fi

# ==========================================
# Validate TLS Certificates
# ==========================================
log_info "Validating TLS certificates..."

if [ ! -f "$CERT_FILE" ]; then
    log_error "TLS certificate not found at $CERT_FILE"
    log_error "Mount your certificate using XRD_CERT_PATH in .env"
    log_error "Example: XRD_CERT_PATH=/etc/ssl/certs/your-cert.pem"
    exit 1
fi

if [ ! -f "$KEY_FILE" ]; then
    log_error "TLS private key not found at $KEY_FILE"
    log_error "Mount your private key using XRD_KEY_PATH in .env"
    log_error "Example: XRD_KEY_PATH=/etc/ssl/private/your-key.pem"
    exit 1
fi

# Verify certificate is readable
if [ ! -r "$CERT_FILE" ]; then
    log_error "Certificate file is not readable"
    exit 1
fi

if [ ! -r "$KEY_FILE" ]; then
    log_error "Private key file is not readable"
    exit 1
fi

# Verify certificate validity
CERT_EXPIRY=$(openssl x509 -enddate -noout -in "$CERT_FILE" 2>/dev/null | cut -d= -f2)
if [ -n "$CERT_EXPIRY" ]; then
    CERT_EXPIRY_EPOCH=$(date -d "$CERT_EXPIRY" +%s 2>/dev/null || echo "0")
    NOW_EPOCH=$(date +%s)
    DAYS_LEFT=$(( (CERT_EXPIRY_EPOCH - NOW_EPOCH) / 86400 ))

    if [ "$DAYS_LEFT" -lt 0 ]; then
        log_error "TLS certificate has EXPIRED!"
        log_error "Expiry: $CERT_EXPIRY"
        exit 1
    elif [ "$DAYS_LEFT" -lt 30 ]; then
        log_warn "TLS certificate expires in $DAYS_LEFT days"
        log_warn "Expiry: $CERT_EXPIRY"
    else
        log_ok "TLS certificate valid for $DAYS_LEFT days"
    fi
else
    log_warn "Could not verify certificate expiry date"
fi

# Show certificate subject
CERT_SUBJECT=$(openssl x509 -subject -noout -in "$CERT_FILE" 2>/dev/null | sed 's/subject=//')
log_info "Certificate Subject: $CERT_SUBJECT"

log_ok "TLS certificates validated"

# ==========================================
# Validate User Mapfile
# ==========================================
log_info "Validating user mapfile..."

MAPFILE="/etc/xrootd/mapfile"
if [ ! -f "$MAPFILE" ]; then
    log_error "User mapfile not found at $MAPFILE"
    log_error "Mount your mapfile using XRD_MAPFILE_PATH in .env"
    log_error "Example: XRD_MAPFILE_PATH=/opt/xrootd/mapfile"
    exit 1
fi

if [ ! -r "$MAPFILE" ]; then
    log_error "Mapfile is not readable"
    exit 1
fi

# Validate JSON syntax
if command -v python3 &>/dev/null; then
    if ! python3 -c "import json; json.load(open('$MAPFILE'))" 2>/dev/null; then
        log_error "Mapfile is not valid JSON"
        log_error "Check syntax at: $MAPFILE"
        exit 1
    fi

    # Count mappings
    MAPPING_COUNT=$(python3 -c "import json; print(len(json.load(open('$MAPFILE'))))" 2>/dev/null || echo "?")
    if [ "$MAPPING_COUNT" = "0" ]; then
        log_warn "Mapfile is empty (no user mappings) — mount a mapfile via XRD_MAPFILE_PATH for production use"
    else
        log_ok "Mapfile valid with $MAPPING_COUNT user mappings"
    fi
else
    log_warn "python3 not available, skipping mapfile JSON validation"
fi

# Show first few mappings (without sensitive data)
log_info "Configured mappings (first 5):"
head -n 20 "$MAPFILE" | grep -E '"sub"|"result"' | head -n 10 | while read line; do
    echo "    $line"
done

# ==========================================
# Validate Data Directory
# ==========================================
log_info "Validating data directory..."

DATA_DIR="/data"
if [ ! -d "$DATA_DIR" ]; then
    log_error "Data directory not found at $DATA_DIR"
    log_error "Mount your data directory using XROOTD_DATA_DIR in .env"
    log_error "Example: XROOTD_DATA_DIR=/lustre/mydata"
    exit 1
fi

# Check if it's a mount point (not just an empty directory)
if [ "$(stat -c %d "$DATA_DIR")" = "$(stat -c %d "$DATA_DIR/..")" ]; then
    log_warn "Data directory may not be a mount point"
    log_warn "Ensure XROOTD_DATA_DIR is properly mounted"
fi

# Show filesystem info
log_info "Data directory info:"
FS_TYPE=$(df -T "$DATA_DIR" 2>/dev/null | tail -1 | awk '{print $2}')
FS_SIZE=$(df -h "$DATA_DIR" 2>/dev/null | tail -1 | awk '{print $2}')
FS_AVAIL=$(df -h "$DATA_DIR" 2>/dev/null | tail -1 | awk '{print $4}')
FS_USE=$(df -h "$DATA_DIR" 2>/dev/null | tail -1 | awk '{print $5}')

echo "    Filesystem: $FS_TYPE"
echo "    Size: $FS_SIZE"
echo "    Available: $FS_AVAIL"
echo "    Usage: $FS_USE"

# Lustre-specific checks
if [ "$FS_TYPE" = "lustre" ]; then
    log_ok "Lustre filesystem detected"

    # Check if we can read Lustre striping info
    if command -v lfs &>/dev/null; then
        STRIPE_INFO=$(lfs getstripe -d "$DATA_DIR" 2>/dev/null | head -3)
        if [ -n "$STRIPE_INFO" ]; then
            log_info "Lustre striping:"
            echo "$STRIPE_INFO" | while read line; do echo "    $line"; done
        fi
    fi
elif [ "$FS_TYPE" = "gpfs" ]; then
    log_ok "GPFS filesystem detected"
elif [ "$FS_TYPE" = "nfs" ] || [ "$FS_TYPE" = "nfs4" ]; then
    log_ok "NFS filesystem detected"
    log_warn "Ensure UIDs in mapfile match NFS server UIDs"
fi

log_ok "Data directory validated"

# ==========================================
# Validate CA Certificates (Optional)
# ==========================================
log_info "Checking CA certificates..."

CA_DIR="/etc/grid-security/certificates"
if [ -d "$CA_DIR" ] && [ "$(ls -A $CA_DIR 2>/dev/null)" ]; then
    CA_COUNT=$(find "$CA_DIR" -name "*.pem" -o -name "*.0" 2>/dev/null | wc -l)
    log_ok "CA certificates found: $CA_COUNT files"
else
    log_warn "No CA certificates in $CA_DIR"
    log_warn "TLS verification may fail unless using 'xrd.tlsca noverify'"
fi

# ==========================================
# Check XRootD User
# ==========================================
log_info "Checking xrootd user..."

if ! id -u xrootd >/dev/null 2>&1; then
    log_warn "Creating xrootd user..."
    useradd -r -s /bin/bash -d /var/spool/xrootd xrootd
fi

XROOTD_UID=$(id -u xrootd)
XROOTD_GID=$(id -g xrootd)
log_ok "XRootD user: uid=$XROOTD_UID gid=$XROOTD_GID"

# ==========================================
# Show Resource Limits
# ==========================================
log_info "Resource limits:"
echo "    Max open files: $(ulimit -n)"
echo "    Max processes: $(ulimit -u)"

# ==========================================
# Show Configuration
# ==========================================
log_info "XRootD configuration:"
echo "    Hostname: ${XROOTD_HOSTNAME:-$(hostname)}"
echo "    Data Directory: $DATA_DIR"
echo "    TLS: Enabled"
echo "    Protocol: ZTN"
echo "    Command: $@"

# ==========================================
# Start XRootD
# ==========================================
echo ""
echo "========================================"
echo "  Starting XRootD Production Service"
echo "========================================"
echo ""

# CRITICAL: Switch to xrootd user but PRESERVE capabilities
# The multiuser plugin requires CAP_SETUID and CAP_SETGID
# Using 'runuser' with '--' preserves capabilities set by setcap
exec runuser -u xrootd -- "$@"
