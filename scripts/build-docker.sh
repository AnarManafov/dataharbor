#!/bin/bash
# =============================================================================
# DataHarbor Docker Image Build Script
# =============================================================================
# Builds and pushes Docker images to GHCR with "dev" tag for local development
#
# Architecture Strategy:
#   - backend, frontend, nginx: Multi-arch (linux/amd64,linux/arm64)
#     These can be tested on both Mac (ARM64) and Linux (AMD64)
#   - xrootd: AMD64 only (CERN/OSG packages not available for ARM64)
#
# Prerequisites:
#   - Docker with buildx support
#   - GitHub Personal Access Token with packages:write scope
#
# Usage:
#   ./scripts/build-docker.sh              # Build all images
#   ./scripts/build-docker.sh backend      # Build only backend
#   ./scripts/build-docker.sh frontend     # Build only frontend
#   ./scripts/build-docker.sh nginx        # Build only nginx
#   ./scripts/build-docker.sh xrootd       # Build only xrootd
#   ./scripts/build-docker.sh --no-push    # Build without pushing
#   ./scripts/build-docker.sh --amd64-only # Force AMD64 only for all images
#
# Environment Variables:
#   GITHUB_TOKEN    - GitHub PAT with packages:write (required for push)
#   GITHUB_USER     - GitHub username (default: anarmanafov)
#   IMAGE_TAG       - Custom tag (default: dev)
# =============================================================================

# Configuration
REGISTRY="ghcr.io"
IMAGE_OWNER="${GITHUB_USER:-anarmanafov}"
IMAGE_TAG="${IMAGE_TAG:-dev}"
PLATFORM_MULTI="linux/amd64,linux/arm64"
PLATFORM_AMD64="linux/amd64"
AMD64_ONLY="false"

# Script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Images that support multi-arch (arm64 + amd64)
# xrootd is NOT in this list because CERN/OSG packages are x86_64 only
MULTI_ARCH_IMAGES=("backend" "frontend" "nginx")

# Enable strict mode after variable declarations
set -euo pipefail

# Helper function to get dockerfile for an image
get_dockerfile() {
    local name=$1
    case "$name" in
        backend)  echo "docker/backend/Dockerfile" ;;
        frontend) echo "docker/frontend/Dockerfile" ;;
        nginx)    echo "docker/nginx/Dockerfile" ;;
        xrootd)   echo "docker/xrootd/Dockerfile" ;;
        *)        echo "" ;;
    esac
}

# Helper function to check if image name is valid
is_valid_image() {
    local name=$1
    case "$name" in
        backend|frontend|nginx|xrootd) return 0 ;;
        *) return 1 ;;
    esac
}

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_usage() {
    cat << EOF
Usage: $(basename "$0") [OPTIONS] [IMAGE...]

Build and push DataHarbor Docker images to GHCR.

OPTIONS:
    --no-push       Build images without pushing to registry
    --tag TAG       Use custom tag (default: dev)
    --amd64-only    Force AMD64-only builds for all images
    --help, -h      Show this help message

IMAGES:
    backend         Build backend image (multi-arch: amd64 + arm64)
    frontend        Build frontend image (multi-arch: amd64 + arm64)
    nginx           Build nginx image (multi-arch: amd64 + arm64)
    xrootd          Build xrootd image (amd64 only - no ARM64 packages)
    (none)          Build all images

EXAMPLES:
    $(basename "$0")                    # Build and push all images (multi-arch where supported)
    $(basename "$0") backend frontend   # Build and push backend and frontend
    $(basename "$0") --no-push          # Build all without pushing
    $(basename "$0") --tag test backend # Build backend with 'test' tag
    $(basename "$0") --amd64-only       # Build all images for amd64 only

ENVIRONMENT:
    GITHUB_TOKEN    GitHub PAT with packages:write scope (required for push)
    GITHUB_USER     GitHub username (default: anarmanafov)
    IMAGE_TAG       Default tag (default: dev)

ARCHITECTURE NOTES:
    - backend, frontend, nginx: Multi-arch (linux/amd64 + linux/arm64)
      Go cross-compiles natively, Node.js/nginx have ARM64 support
    - xrootd: AMD64 only (CERN/OSG XRootD packages are x86_64 only)
EOF
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi
    
    # Check buildx
    if ! docker buildx version &> /dev/null; then
        log_error "Docker buildx is not available"
        exit 1
    fi
    
    # Ensure buildx builder exists
    if ! docker buildx inspect dataharbor-builder &> /dev/null; then
        log_info "Creating buildx builder..."
        docker buildx create --name dataharbor-builder --use --bootstrap
    else
        docker buildx use dataharbor-builder
    fi
    
    log_success "Prerequisites OK"
}

login_to_registry() {
    if [[ -z "${GITHUB_TOKEN:-}" ]]; then
        log_error "GITHUB_TOKEN environment variable is not set"
        log_info "Create a PAT at: https://github.com/settings/tokens"
        log_info "Required scope: packages:write"
        exit 1
    fi
    
    log_info "Logging in to ${REGISTRY}..."
    echo "${GITHUB_TOKEN}" | docker login "${REGISTRY}" -u "${IMAGE_OWNER}" --password-stdin
    log_success "Logged in to ${REGISTRY}"
}

build_image() {
    local name=$1
    local dockerfile=$2
    local full_image="${REGISTRY}/${IMAGE_OWNER}/dataharbor-${name}:${IMAGE_TAG}"
    
    # Determine platform based on image type
    local platform
    if [[ "${AMD64_ONLY}" == "true" ]]; then
        platform="${PLATFORM_AMD64}"
    elif [[ " ${MULTI_ARCH_IMAGES[*]} " =~ " ${name} " ]]; then
        platform="${PLATFORM_MULTI}"
    else
        platform="${PLATFORM_AMD64}"
    fi
    
    log_info "Building ${name}..."
    log_info "  Dockerfile: ${dockerfile}"
    log_info "  Image: ${full_image}"
    log_info "  Platform: ${platform}"
    
    local push_flag=""
    if [[ "${DO_PUSH}" == "true" ]]; then
        push_flag="--push"
    else
        # For multi-arch builds without push, we can't use --load
        # --load only works for single platform
        if [[ "${platform}" == *","* ]]; then
            log_warn "Multi-arch build without push - image will be in build cache only"
            push_flag=""
        else
            push_flag="--load"
        fi
    fi
    
    docker buildx build \
        --platform "${platform}" \
        --file "${PROJECT_ROOT}/${dockerfile}" \
        --tag "${full_image}" \
        --build-arg VERSION="${IMAGE_TAG}" \
        --build-arg BUILD_DATE="$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
        --build-arg VCS_REF="$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')" \
        ${push_flag} \
        "${PROJECT_ROOT}"
    
    if [[ "${DO_PUSH}" == "true" ]]; then
        log_success "Built and pushed: ${full_image} (${platform})"
    else
        log_success "Built: ${full_image} (${platform})"
    fi
}

# Main script
main() {
    local images_to_build=()
    DO_PUSH="true"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --no-push)
                DO_PUSH="false"
                shift
                ;;
            --tag)
                IMAGE_TAG="$2"
                shift 2
                ;;
            --amd64-only)
                AMD64_ONLY="true"
                shift
                ;;
            -h|--help)
                print_usage
                exit 0
                ;;
            backend|frontend|nginx|xrootd)
                images_to_build+=("$1")
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                print_usage
                exit 1
                ;;
        esac
    done
    
    # Default to all images if none specified
    if [[ ${#images_to_build[@]} -eq 0 ]]; then
        images_to_build=("backend" "frontend" "nginx" "xrootd")
    fi
    
    # Determine platform display string
    local platform_display
    if [[ "${AMD64_ONLY}" == "true" ]]; then
        platform_display="${PLATFORM_AMD64} (forced)"
    else
        platform_display="multi-arch (amd64+arm64) / amd64-only for xrootd"
    fi
    
    echo ""
    echo "=========================================="
    echo "  DataHarbor Docker Image Builder"
    echo "=========================================="
    echo "  Registry:  ${REGISTRY}"
    echo "  Owner:     ${IMAGE_OWNER}"
    echo "  Tag:       ${IMAGE_TAG}"
    echo "  Platform:  ${platform_display}"
    echo "  Push:      ${DO_PUSH}"
    echo "  Images:    ${images_to_build[*]}"
    echo "=========================================="
    echo ""
    
    check_prerequisites
    
    if [[ "${DO_PUSH}" == "true" ]]; then
        login_to_registry
    fi
    
    # Build images
    local failed=()
    for name in "${images_to_build[@]}"; do
        local dockerfile=$(get_dockerfile "$name")
        if [[ -n "$dockerfile" ]]; then
            if ! build_image "$name" "$dockerfile"; then
                failed+=("$name")
            fi
        else
            log_warn "Unknown image: $name"
        fi
    done
    
    echo ""
    echo "=========================================="
    echo "  Build Summary"
    echo "=========================================="
    
    if [[ ${#failed[@]} -eq 0 ]]; then
        log_success "All images built successfully!"
        echo ""
        echo "Pull commands:"
        for name in "${images_to_build[@]}"; do
            echo "  docker pull ${REGISTRY}/${IMAGE_OWNER}/dataharbor-${name}:${IMAGE_TAG}"
        done
    else
        log_error "Failed to build: ${failed[*]}"
        exit 1
    fi
    
    if [[ "${DO_PUSH}" == "true" ]]; then
        echo ""
        echo "Deploy with:"
        echo "  VERSION=${IMAGE_TAG} docker compose -f docker/docker-compose.deploy.yml up -d"
    fi
}

main "$@"
