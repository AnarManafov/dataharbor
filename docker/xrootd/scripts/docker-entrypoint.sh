#!/bin/bash
set -e

# Ensure runtime directories exist with correct permissions
# These may need to be recreated if they're on tmpfs (like /var/run)
echo "Setting up runtime directories..."
mkdir -p /var/spool/xrootd /var/run/xrootd /home/xrootd /var/log/xrootd /data
mkdir -p /etc/grid-security/certificates
# Note: /var/run/xrootd/certs is a read-only mount from cert-init, so exclude it from chown
chown -R xrootd:xrootd /var/spool/xrootd /home/xrootd /var/log/xrootd /data
chown xrootd:xrootd /var/run/xrootd
chmod 755 /var/spool/xrootd /var/run/xrootd /home/xrootd /var/log/xrootd /data
# Make grid-security readable by xrootd user
chmod 755 /etc/grid-security /etc/grid-security/certificates

# Create test users for multiuser plugin (for development) if they don't exist
# These users will be used when tokens are mapped via the mapfile
echo "Setting up test users for multiuser plugin..."
# Note: UIDs must match those in Dockerfile
if ! id -u testuser1 &>/dev/null; then
    useradd -u 1001 -m -s /bin/bash testuser1
fi
mkdir -p /data/testuser1
chown testuser1:testuser1 /data/testuser1
chmod 700 /data/testuser1  # Owner-only access

if ! id -u testuser2 &>/dev/null; then
    useradd -u 1002 -m -s /bin/bash testuser2
fi
mkdir -p /data/testuser2
chown testuser2:testuser2 /data/testuser2
chmod 700 /data/testuser2  # Owner-only access

if ! id -u amanafov &>/dev/null; then
    useradd -u 1003 -m -s /bin/bash amanafov
fi
mkdir -p /data/amanafov
chown amanafov:amanafov /data/amanafov
chmod 700 /data/amanafov  # Owner-only access

# Setup test data for user mapping demonstration
if [ -f "/usr/local/bin/setup-test-data.sh" ]; then
    echo "Setting up test data..."
    bash /usr/local/bin/setup-test-data.sh
fi

# Handle certificates: copy from read-only shared volume to writable location
# The shared-certs volume is mounted read-only, so we copy to /var/run/xrootd/certs
# where we can set proper ownership and permissions for xrootd user
CERT_DIR="/var/run/xrootd/certs"
SHARED_CERT_DIR="/var/run/xrootd/certs-shared"
mkdir -p "$CERT_DIR"

if [ -f "$SHARED_CERT_DIR/hostcert.pem" ] && [ -f "$SHARED_CERT_DIR/hostkey.pem" ]; then
    echo "Copying certificates from shared volume to writable location..."
    cp "$SHARED_CERT_DIR/hostcert.pem" "$CERT_DIR/hostcert.pem"
    cp "$SHARED_CERT_DIR/hostkey.pem" "$CERT_DIR/hostkey.pem"
    
    # Set ownership and permissions required by XRootD TLS
    chown xrootd:xrootd "$CERT_DIR/hostcert.pem" "$CERT_DIR/hostkey.pem"
    chmod 644 "$CERT_DIR/hostcert.pem"
    chmod 600 "$CERT_DIR/hostkey.pem"
    
    echo "Certificates copied and configured:"
    ls -la "$CERT_DIR/"
    
    # XRootD v5.8 may require cert and key in the same file
    # Create combined file for TLS configuration
    cat "$CERT_DIR/hostcert.pem" "$CERT_DIR/hostkey.pem" > "$CERT_DIR/hostcert_combined.pem"
    chown xrootd:xrootd "$CERT_DIR/hostcert_combined.pem"
    chmod 600 "$CERT_DIR/hostcert_combined.pem"
    
    # Set up CA cert directory for XRootD TLS
    CA_DIR="/var/run/xrootd/ca-certs"
    mkdir -p "$CA_DIR"
    cp "$CERT_DIR/hostcert.pem" "$CA_DIR/"
    chown -R xrootd:xrootd "$CA_DIR"
    chmod 755 "$CA_DIR"
    chmod 644 "$CA_DIR"/*
    echo "CA cert directory configured:"
    ls -la "$CA_DIR/"
    
    echo "[+] Certificates ready for XRootD with correct permissions"
else
    echo "ERROR: Certificates not found in $SHARED_CERT_DIR"
    echo "The cert-init service should have generated them."
    exit 1
fi

# Ensure xrootd user exists (should be created by package but verify)
if ! id -u xrootd >/dev/null 2>&1; then
    echo "Creating xrootd user..."
    useradd -r -s /bin/bash -d /var/spool/xrootd xrootd
fi

# CRITICAL: Switch to xrootd user but PRESERVE capabilities
# The multiuser plugin requires CAP_SETUID and CAP_SETGID to change filesystem UIDs
# Using 'runuser' with '--' preserves capabilities set by setcap
echo "Starting XRootD as user xrootd with preserved capabilities..."
exec runuser -u xrootd -- "$@"
