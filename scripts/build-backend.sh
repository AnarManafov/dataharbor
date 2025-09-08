#!/bin/bash
# Build script for dataharbor backend with version injection
# Usage: ./scripts/build-backend.sh [output-binary-name]

set -e

# Get the script directory and project root
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"
APP_DIR="$PROJECT_ROOT/app"

# Change to app directory
cd "$APP_DIR"

# Get version from package.json (compatible with both GNU grep and macOS grep)
VERSION=$(grep -o '"version": *"[^"]*"' "$PROJECT_ROOT/package.json" | head -1 | sed 's/.*"\([^"]*\)".*/\1/' || echo "dev")

# Get git commit hash (short)
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Get build time in UTC
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Prepare ldflags
LDFLAGS="-s -w"
LDFLAGS="$LDFLAGS -X github.com/AnarManafov/dataharbor/app/config.Version=$VERSION"
LDFLAGS="$LDFLAGS -X github.com/AnarManafov/dataharbor/app/config.GitCommit=$GIT_COMMIT"
LDFLAGS="$LDFLAGS -X github.com/AnarManafov/dataharbor/app/config.BuildTime=$BUILD_TIME"

# Output binary name (default to 'app' which is Go's default, or use first argument)
OUTPUT_BINARY="${1:-app}"

echo "Building dataharbor-backend..."
echo "  Version: $VERSION"
echo "  Git Commit: $GIT_COMMIT"
echo "  Build Time: $BUILD_TIME"
echo "  Output: $OUTPUT_BINARY"
echo "  Static linking: CGO_ENABLED=0"
echo ""

# Build with static linking (no CGO)
CGO_ENABLED=0 go build -v -ldflags="$LDFLAGS" -o "$OUTPUT_BINARY" .

echo ""
echo "Build successful: $APP_DIR/$OUTPUT_BINARY"

# Verify static linking
if command -v file &> /dev/null; then
    echo ""
    echo "Binary information:"
    file "$OUTPUT_BINARY"
fi
