# Nginx Gateway - Reverse proxy for backend API and frontend SPA
#
# Multi-stage build with separate dev and prod targets:
#   - dev:  Proxies to Vite dev server (https://frontend:5173) with HMR support
#   - prod: Proxies to static nginx (http://frontend:80) - SELF-CONTAINED
#
# Usage:
#   docker build --target dev -t dataharbor-nginx:dev .
#   docker build --target prod -t dataharbor-nginx:latest .
#
# Or in docker-compose.yml:
#   build:
#     target: dev   # or prod

# ============================================
# Base stage - common setup for both dev/prod
# ============================================
FROM nginx:1.29-alpine AS base

RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    openssl \
    && rm -rf /var/cache/apk/*

# Remove default config
RUN rm /etc/nginx/conf.d/default.conf

# Install main nginx.conf with JSON log format (stdout/stderr only)
COPY docker/nginx/nginx.conf /etc/nginx/nginx.conf

# SSL certificates mounted at runtime
RUN mkdir -p /etc/nginx/ssl

# Ensure nginx user exists
RUN addgroup -g 101 nginx 2>/dev/null || true && \
    adduser -D -u 101 -G nginx nginx 2>/dev/null || true

RUN chown -R nginx:nginx /var/cache/nginx && \
    chown -R nginx:nginx /var/log/nginx && \
    chmod -R 755 /etc/nginx/conf.d

EXPOSE 80 443

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:80/health || exit 1

CMD ["nginx", "-g", "daemon off;"]

# ============================================
# Development stage
# - Proxies to Vite dev server (https://frontend:5173)
# - WebSocket support for Hot Module Reload
# ============================================
FROM base AS dev

COPY docker/nginx/conf.d/dataharbor-dev.conf /etc/nginx/conf.d/dataharbor.conf

# ============================================
# Production stage (default)
# - Proxies to static nginx (http://frontend:80)
# - Self-contained - no external config needed
# ============================================
FROM base AS prod

COPY docker/nginx/conf.d/dataharbor-prod.conf /etc/nginx/conf.d/dataharbor.conf
