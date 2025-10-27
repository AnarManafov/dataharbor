# Frontend Dockerfile - Multi-stage build: Node.js build + Nginx serve

FROM node:24-alpine AS builder

WORKDIR /build

COPY web/package*.json ./
RUN npm ci --only=production

COPY web/ ./
RUN npm run build

FROM nginx:1.28-alpine

RUN apk add --no-cache ca-certificates tzdata

RUN rm /etc/nginx/conf.d/default.conf

COPY docker/frontend/nginx.conf /etc/nginx/conf.d/dataharbor.conf
COPY --from=builder /build/dist /usr/share/nginx/html

# Ensure nginx user exists (may already exist in base image)
RUN addgroup -g 101 nginx || true && \
    adduser -D -u 101 -G nginx nginx || true

RUN chown -R nginx:nginx /usr/share/nginx/html && \
    chown -R nginx:nginx /var/cache/nginx && \
    chown -R nginx:nginx /var/log/nginx && \
    chmod -R 755 /usr/share/nginx/html

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:80/ || exit 1

CMD ["nginx", "-g", "daemon off;"]
