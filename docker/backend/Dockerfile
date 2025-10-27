# Backend Dockerfile - Multi-stage build for optimized production image

FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

COPY app/go.mod app/go.sum ./
RUN go mod download

COPY app/ ./

# Build static binary: CGO_ENABLED=0 for portability, -ldflags="-s -w" strips debug info
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o dataharbor-backend \
    .

FROM alpine:3.22

RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    xrootd-client \
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
