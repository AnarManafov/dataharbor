# Development Container for DataHarbor
# Based on Microsoft's official devcontainer Go 1.25 image (Go 1.25.3)
#
# Why MS devcontainer instead of golang:1.25?
# - Pre-configured 'vscode' user with sudo access
# - Includes common dev tools (git, curl, wget, vim, jq, etc.)
# - Pre-installed Go tools (gopls, delve, golangci-lint, staticcheck)
# - Node.js/npm support via features
# - Oh My Zsh and nvm pre-configured
# - Optimized for VS Code Dev Container features
#
# What's included by default in mcr.microsoft.com/devcontainers/go:1.25-bookworm:
# - Go 1.25.3
# - gopls (Go language server)
# - delve (debugger)
# - golangci-lint (latest)
# - staticcheck, revive, go-outline, gopkgs
# - git, curl, wget, vim, less, jq, htop, zsh
# - make, gcc, g++, build-essential
# - Docker-in-Docker support
# - Non-root 'vscode' user with sudo

FROM mcr.microsoft.com/devcontainers/go:1.25-bookworm

# Avoid warnings by switching to noninteractive
ENV DEBIAN_FRONTEND=noninteractive

# Install additional development tools
# Already included: git, curl, wget, vim, jq, make, gcc, build-essential, golangci-lint, gopls, delve
# We add: XRootD development libraries and utilities for backend XRD integration

# Install XRootD development packages (for go-hep/xrootd integration)
RUN apt-get update && apt-get install -y --no-install-recommends \
    xrootd-client \
    libxrootd-client-dev \
    libxrootd-dev \
    && rm -rf /var/lib/apt/lists/*

# Node.js and npm will be installed via devcontainer features
# This allows flexible version management through devcontainer.json

# Install Go tools specific to DataHarbor (project-specific versions installed via postCreateCommand)
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest \
    && go install golang.org/x/tools/gopls@latest \
    && go install github.com/go-delve/delve/cmd/dlv@latest

# Verify installations (useful for debugging)
RUN echo "=== Installed Tool Versions ===" \
    && go version \
    && golangci-lint version \
    && make --version \
    && git --version \
    && xrdfs --version 2>&1 | head -n1 || echo "XRootD client installed" \
    && echo "=========================="

# Switch back to dialog for any ad-hoc use of apt-get
ENV DEBIAN_FRONTEND=dialog

# Default user is already 'vscode' in MS devcontainer image
# WORKDIR is already set to /workspaces in MS devcontainer image
