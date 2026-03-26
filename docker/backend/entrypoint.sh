#!/bin/sh
# Entrypoint script for dataharbor-backend container
# Fixes ownership of bind-mounted log directory, then drops to non-root user.

set -e

# Ensure the log directory is writable by the dataharbor user
chown dataharbor:dataharbor /app/log

# Drop privileges and exec the backend binary
exec su-exec dataharbor /app/dataharbor-backend "$@"
