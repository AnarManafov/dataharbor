#!/bin/bash

# Example SSL Certificate Setup Script for DataHarbor
# Copy and customize this script for your specific environment

# Configuration - Modify these paths for your setup
PKM_WORKSPACE_PATH="${PKM_WORKSPACE_PATH:-$HOME/Documents/workspace/pkm}"
CERT_SUBPATH="${CERT_SUBPATH:-docs/gsi/dataharbor/test/cert}"

# Certificate paths in your PKM
CERT_DIR="$PKM_WORKSPACE_PATH/$CERT_SUBPATH"
SSL_KEY="$CERT_DIR/server.key"
SSL_CERT="$CERT_DIR/server.crt"

echo "🔍 Checking for SSL certificates in PKM workspace..."
echo "Looking in: $CERT_DIR"

if [[ -f "$SSL_KEY" && -f "$SSL_CERT" ]]; then
    echo "✅ Certificates found!"
    echo "🔐 Setting environment variables..."
    
    export VITE_SSL_KEY="$SSL_KEY"
    export VITE_SSL_CERT="$SSL_CERT"
    
    echo "Environment variables set:"
    echo "  VITE_SSL_KEY=$VITE_SSL_KEY"
    echo "  VITE_SSL_CERT=$VITE_SSL_CERT"
    echo
    echo "🚀 You can now run:"
    echo "  npm run dev"
    echo "  npm run sandbox"
    echo
    echo "💡 To make this permanent, add the export commands to your ~/.bashrc or ~/.zshrc"
else
    echo "❌ Certificates not found in expected location."
    echo "Please check the PKM_WORKSPACE_PATH in this script."
    echo
    echo "Expected files:"
    echo "  $SSL_KEY"
    echo "  $SSL_CERT"
fi
