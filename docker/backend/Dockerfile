# Backend Dockerfile - Multi-stage build for optimized production image

# These ARGs are automatically set by Docker Buildx for multi-platform builds
# TARGETOS defaults to "linux", TARGETARCH is set per platform (amd64, arm64, etc.)
ARG TARGETOS=linux
ARG TARGETARCH
ARG BUILDPLATFORM

# Use the native build platform for faster compilation with cross-compilation
FROM --platform=${BUILDPLATFORM} golang:1.25-alpine AS builder

# Re-declare ARGs after FROM to make them available in this stage
ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy dependency files first for better layer caching
COPY app/go.mod app/go.sum ./

# Download dependencies with cache mount for faster rebuilds
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY app/ ./

# Build static binary with cache mounts for Go build cache
# CGO_ENABLED=0 for portability, -ldflags="-s -w" strips debug info
# Cross-compile using TARGETOS and TARGETARCH on the native platform (much faster than QEMU)
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w" \
    -o dataharbor-backend \
    .

FROM alpine:3.22

# Install only essential runtime dependencies
# Note: xrootd-client is NOT needed - the Go backend uses native Go XRootD client library
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    && rm -rf /var/cache/apk/*

# Run as non-root user for security
RUN addgroup -g 1000 dataharbor && \
    adduser -D -u 1000 -G dataharbor dataharbor

WORKDIR /app

COPY --from=builder /build/dataharbor-backend /app/dataharbor-backend

RUN mkdir -p /app/config /app/log && \
    chown -R dataharbor:dataharbor /app

USER dataharbor

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/dataharbor-backend"]
CMD ["-config", "/app/config/application.yaml"]
