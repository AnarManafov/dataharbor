# ============================================
# XRootD Server - Development Dockerfile
# ============================================
# Full-featured development environment with:
# - Auto-generated self-signed TLS certificates
# - Pre-configured test users for user mapping demo
# - Test data generation
# - Verbose logging for debugging
#
# For production, use Dockerfile.prod instead
#
# Platform: linux/amd64 required (ARM64 packages unavailable from CERN/OSG)
#
# Usage:
#   docker build -f Dockerfile -t dataharbor-xrootd:dev ..
# ============================================

# Global build argument for XRootD version
ARG XROOTD_VERSION=6.1.1

# ---------------------------------------------
# Build stage: Install XRootD and dependencies
# ---------------------------------------------
FROM rockylinux/rockylinux:10 AS builder

# Install XRootD repository and base dependencies
# Using BuildKit cache mount for faster rebuilds
# - dnf-plugins-core provides `dnf config-manager` (not preinstalled on EL10)
# - CERN repo provides the core xrootd 6.x packages (xrootd, server, libs, scitokens)
# - OSG repo provides xrootd-multiuser; the 6.x-ABI build currently lives in the
#   `osg-upcoming-development` channel (enabled per-install below)
# - Stay on the 25-main series. An `osg/26-main` el10 series exists but is still an
#   empty bootstrap skeleton (verified 2026-08-27): its release/development/
#   upcoming-development/testing channels together ship only osg-release,
#   buildsys-macros and osg-ca-certs -- no xrootd-multiuser, and osg-upcoming and
#   osg-contrib 404 outright. Re-check before bumping to 26-main.
RUN --mount=type=cache,target=/var/cache/dnf,sharing=locked \
	dnf install -y epel-release dnf-plugins-core && \
	curl -L https://xrootd.web.cern.ch/xrootd.repo -o /etc/yum.repos.d/xrootd.repo && \
	dnf install -y https://repo.opensciencegrid.org/osg/25-main/osg-25-main-el10-release-latest.rpm && \
	dnf config-manager --set-enabled crb && \
	dnf clean all

# Install XRootD and plugins
# Pinned version for reproducible builds
# Note: xrootd-multiuser comes from the OSG `osg-upcoming-development` channel,
# which carries the build linked against the XRootD 6.x plugin ABI
# (libXrdMultiuser-6.so -> libXrdUtils.so.6). The OSG stable/main channels still
# ship the 5.x-ABI build, which will not load under XRootD 6.x. The core xrootd
# packages still resolve from the higher-priority CERN repo.
ARG XROOTD_VERSION
RUN --mount=type=cache,target=/var/cache/dnf,sharing=locked \
	dnf install -y --enablerepo=osg-upcoming-development \
	xrootd-${XROOTD_VERSION} \
	xrootd-server-${XROOTD_VERSION} \
	xrootd-libs-${XROOTD_VERSION} \
	xrootd-scitokens-${XROOTD_VERSION} \
	xrootd-multiuser \
	openssl \
	libcap \
	&& dnf clean all

# xrootd RPM now uses systemd-sysusers instead of RPM scriptlets to create
# the xrootd user/group. Run it explicitly so the user exists in /etc/passwd
# before the runtime stage copies these files from the builder.
RUN systemd-sysusers

# Grant capabilities for multiuser plugin to change filesystem UIDs/GIDs
RUN setcap 'cap_setuid,cap_setgid=+ep' /usr/bin/xrootd

RUN mkdir -p /usr/lib64/xrootd /usr/share/xrootd

# ---------------------------------------------
# Runtime stage: Development image
# ---------------------------------------------
FROM rockylinux/rockylinux:10

# Re-declare ARG for use in this stage
ARG XROOTD_VERSION

# Labels for container identification
LABEL org.opencontainers.image.title="DataHarbor XRootD Server (Development)" \
	org.opencontainers.image.description="XRootD server with test users and self-signed certs for development" \
	org.opencontainers.image.version="${XROOTD_VERSION}" \
	org.opencontainers.image.vendor="DataHarbor" \
	xrootd.version="${XROOTD_VERSION}" \
	xrootd.features="ztn,tls,scitokens,multiuser" \
	xrootd.mode="development"

# Minimal runtime dependencies + dos2unix for Windows/WSL compatibility
RUN --mount=type=cache,target=/var/cache/dnf,sharing=locked \
	dnf install -y openssl ca-certificates dos2unix && \
	dnf clean all

# Copy XRootD binaries and libraries from builder
# Note: xrdfs client binary is not included; use the full builder image for ad-hoc debugging:
#   docker run --rm dataharbor-xrootd:builder xrdfs <host>:1094 ls /
COPY --from=builder /usr/bin/xrootd /usr/bin/xrootd
COPY --from=builder /usr/lib64/libXrd*.so* /usr/lib64/
COPY --from=builder /usr/lib64/libSciTokens.so.* /usr/lib64/
COPY --from=builder /usr/lib64/xrootd /usr/lib64/
COPY --from=builder /usr/share/xrootd /usr/share/

# Create unversioned symlink for plugin loading (config references libXrdAccSciTokens.so)
# XRootD 6.x bumped the plugin ABI suffix from -5 to -6.
RUN ln -s /usr/lib64/libXrdAccSciTokens-6.so /usr/lib64/libXrdAccSciTokens.so

# Copy system users (xrootd user created by package install)
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /etc/shadow /etc/shadow

# Create test users for token-to-Unix-user mapping demonstration
RUN useradd -m -s /bin/bash -u 1001 testuser1 && \
	useradd -m -s /bin/bash -u 1002 testuser2 && \
	useradd -m -s /bin/bash -u 1003 amanafov

# Create directories with proper permissions for xrootd and test users
RUN mkdir -p \
	/etc/xrootd \
	/etc/grid-security/certificates \
	/var/log/xrootd \
	/var/spool/xrootd \
	/var/run/xrootd \
	/data \
	/data/testuser1 \
	/data/testuser2 \
	/data/amanafov && \
	chown -R xrootd:xrootd \
	/var/log/xrootd \
	/var/spool/xrootd \
	/var/run/xrootd \
	/etc/grid-security/certificates && \
	chmod 755 /data && \
	chown xrootd:xrootd /data && \
	chown testuser1:testuser1 /data/testuser1 && \
	chown testuser2:testuser2 /data/testuser2 && \
	chown amanafov:amanafov /data/amanafov && \
	chmod 700 /data/testuser1 /data/testuser2 /data/amanafov

# Copy configuration files
COPY docker/xrootd/configs/xrootd-dev.cfg /etc/xrootd/xrootd-dev.cfg
COPY docker/xrootd/configs/xrootd-prod.cfg /etc/xrootd/xrootd-prod.cfg
COPY docker/xrootd/configs/scitokens_dev.cfg /etc/xrootd/scitokens_dev.cfg
COPY docker/xrootd/configs/scitokens_prod.cfg /etc/xrootd/scitokens_prod.cfg
COPY docker/xrootd/configs/mapfile /etc/xrootd/mapfile

# Copy scripts
COPY docker/xrootd/scripts/setup-test-data.sh /usr/local/bin/setup-test-data.sh
COPY docker/xrootd/scripts/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
COPY docker/xrootd/scripts/docker-entrypoint-prod.sh /usr/local/bin/docker-entrypoint-prod.sh

# Convert line endings for all config files and scripts (fixes WSL/Windows CRLF issues)
RUN dos2unix /etc/xrootd/*.cfg \
	/etc/xrootd/mapfile \
	/usr/local/bin/setup-test-data.sh \
	/usr/local/bin/docker-entrypoint.sh \
	/usr/local/bin/docker-entrypoint-prod.sh && \
	chmod +x /usr/local/bin/setup-test-data.sh \
	/usr/local/bin/docker-entrypoint.sh \
	/usr/local/bin/docker-entrypoint-prod.sh && \
	chown -R xrootd:xrootd /etc/xrootd && \
	chmod 644 /etc/xrootd/*.cfg /etc/xrootd/mapfile

EXPOSE 1094

# Work around xrootd home directory issue: https://github.com/xrootd/xrootd/issues/365
WORKDIR /var/spool/xrootd

# Health check - verify XRootD port is responding
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
	CMD timeout 2 bash -c '</dev/tcp/localhost/1094' || exit 1

# Development entrypoint handles cert setup, test users, and test data
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]

# Default: use dev config
CMD ["/usr/bin/xrootd", "-c", "/etc/xrootd/xrootd-dev.cfg", "-n", "xrootd"]
