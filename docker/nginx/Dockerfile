# Nginx Gateway - Reverse proxy for backend API and frontend SPA

FROM nginx:1.28-alpine

RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    openssl \
    && rm -rf /var/cache/apk/*

RUN rm /etc/nginx/conf.d/default.conf

COPY docker/nginx/nginx.conf /etc/nginx/conf.d/dataharbor.conf

# SSL certificates mounted at runtime from cert-init (dev) or host (prod)
RUN mkdir -p /etc/nginx/ssl

# Ensure nginx user exists (may already exist in base image)
RUN addgroup -g 101 nginx 2>/dev/null || true && \
    adduser -D -u 101 -G nginx nginx 2>/dev/null || true

RUN chown -R nginx:nginx /var/cache/nginx && \
    chown -R nginx:nginx /var/log/nginx && \
    chmod -R 755 /etc/nginx/conf.d

EXPOSE 80 443

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:80/health || exit 1

CMD ["nginx", "-g", "daemon off;"]
