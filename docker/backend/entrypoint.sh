#!/bin/sh
# Entrypoint script for dataharbor-backend container
# Drops to non-root user before starting the backend.

set -e

# Drop privileges and exec the backend binary
exec su-exec dataharbor /app/dataharbor-backend "$@"
