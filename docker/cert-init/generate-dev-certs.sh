#!/bin/sh
set -e

# Certificate paths
CERT_DIR="/certs"

# Nginx naming convention
NGINX_CERT="$CERT_DIR/server.crt"
NGINX_KEY="$CERT_DIR/server.key"

# XRootD naming convention
XROOTD_CERT="$CERT_DIR/hostcert.pem"
XROOTD_KEY="$CERT_DIR/hostkey.pem"
XROOTD_COMBINED="$CERT_DIR/hostcert_combined.pem"

echo "Certificate Init Container - Generating self-signed certificates for development..."

# Check if certificates already exist (volume might be reused)
if [ -f "$NGINX_CERT" ] && [ -f "$NGINX_KEY" ]; then
    echo "[+] Certificates already exist, skipping generation"
    ls -lh "$CERT_DIR"
    exit 0
fi

# Generate self-signed certificate (using nginx naming first)
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout "$NGINX_KEY" \
    -out "$NGINX_CERT" \
    -subj "/C=DE/ST=Hessen/L=Darmstadt/O=DataHarbor/OU=Development/CN=localhost" \
    -addext "subjectAltName=DNS:localhost,DNS:*.localhost,DNS:xrootd,DNS:nginx,IP:127.0.0.1"

# Create symlinks for XRootD naming convention
ln -sf server.crt "$XROOTD_CERT"
ln -sf server.key "$XROOTD_KEY"

# Create combined certificate file (cert + key in one file) for XRootD
cat "$NGINX_CERT" "$NGINX_KEY" > "$XROOTD_COMBINED"

# Set proper permissions for shared volume access
# Nginx needs to read cert and key separately with standard permissions
chmod 644 "$NGINX_CERT" "$NGINX_KEY"
# XRootD uses the combined file and TLS requires restrictive permissions (400 = read-only by owner)
chmod 400 "$XROOTD_COMBINED"

echo "[+] Self-signed certificates generated successfully"
echo "  Nginx format: server.crt, server.key"
echo "  XRootD format: hostcert.pem, hostkey.pem (symlinks)"
echo "  Combined: hostcert_combined.pem"
echo "  Permissions: 644 (readable by all containers)"
echo "  Valid for: 365 days"
echo "  Hostnames: localhost, *.localhost, xrootd, nginx, 127.0.0.1"

# Display all files
ls -lh "$CERT_DIR"

echo "[+] Certificate initialization complete - all services can now access certificates"
