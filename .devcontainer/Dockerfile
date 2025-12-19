# Development Container for DataHarbor
# Uses base devcontainer with features for Go, Node.js, and development tools
#
# Architecture:
# - Base: mcr.microsoft.com/devcontainers/base:ubuntu-24.04 (Ubuntu 24.04 LTS)
# - Go, Node.js, golangci-lint installed via devcontainer features
# - XRootD packages installed directly (no feature available)
#

FROM mcr.microsoft.com/devcontainers/base:ubuntu-24.04

# Install XRootD development packages (for go-hep/xrootd integration)
# These are project-specific and not available as devcontainer features
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get install -y --no-install-recommends \
    xrootd-client \
    libxrootd-client-dev \
    libxrootd-dev \
    && rm -rf /var/lib/apt/lists/*

# Go, Node.js, golangci-lint, and other tools are installed via features in devcontainer.json
# Default user is 'vscode' in MS devcontainer base image
