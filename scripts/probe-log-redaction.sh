#!/usr/bin/env bash
# Log redaction probe (GH-82).
#
# Sends requests that carry a one-off sentinel "secret" as a bearer token, as
# cookies, in the query string and in a JSON body, then searches the backend
# log for it.
#
#   PASS          the requests were logged and the sentinel is nowhere in the log
#   FAIL          the sentinel is in the log (exit 1)
#   INCONCLUSIVE  the requests were not found in the log, so nothing was proven:
#                 debug logging is off, or this is the wrong log file (exit 2)
#
# The backend must run with server.debug: true and logging level debug (the
# development config does both). The probe is read-only: every request is
# rejected with 401 before it reaches a handler.
#
# Against the compose dev stack the backend logs to stdout only, so capture it
# first and go through the nginx gateway:
#   docker logs -f dataharbor-backend-dev > /tmp/backend.log 2>&1 &
#   LOG_PROBE_BASE_URL=https://localhost LOG_PROBE_LOG_FILE=/tmp/backend.log \
#     ./scripts/probe-log-redaction.sh
#
# Environment:
#   LOG_PROBE_BASE_URL   backend base URL        (default https://localhost:22000)
#   LOG_PROBE_LOG_FILE   backend log file        (default app/log/dataharbor_app.log)
#   LOG_PROBE_INSECURE   1 = accept a self-signed certificate (default 1)
#   LOG_PROBE_WAIT       seconds to wait for the log to be written (default 2)

set -euo pipefail

BASE_URL="${LOG_PROBE_BASE_URL:-https://localhost:22000}"
LOG_FILE="${LOG_PROBE_LOG_FILE:-app/log/dataharbor_app.log}"
INSECURE="${LOG_PROBE_INSECURE:-1}"
WAIT="${LOG_PROBE_WAIT:-2}"

if [ ! -r "$LOG_FILE" ]; then
  echo "INCONCLUSIVE: cannot read log file '$LOG_FILE' (set LOG_PROBE_LOG_FILE)"
  exit 2
fi

rand="$(od -An -N16 -tx1 /dev/urandom | tr -d ' \n')"
SENTINEL="LOGPROBESECRET${rand}"   # must never appear in the log
MARKER="logprobe-marker-${rand:0:12}"  # not a secret: must appear in the log

curl_opts=(--silent --show-error --output /dev/null --max-time 10 --write-out '%{http_code}')
[ "$INSECURE" = "1" ] && curl_opts+=(--insecure)

send() { # method path [json-body]
  local method="$1" path="$2" body="${3:-}" args=()
  if [ -n "$body" ]; then
    args+=(--header 'Content-Type: application/json' --data "$body")
  fi
  local status
  if ! status="$(curl "${curl_opts[@]}" --request "$method" \
      --header "Authorization: Bearer dh_cli_${SENTINEL}" \
      --header "Cookie: probe=${SENTINEL}; dataharbor-session=${SENTINEL}" \
      --header "X-Log-Probe: ${MARKER}" \
      ${args[@]+"${args[@]}"} \
      "${BASE_URL}${path}?access_token=${SENTINEL}&state=${SENTINEL}")"; then
    echo "INCONCLUSIVE: request to ${BASE_URL}${path} failed (is the backend running?)"
    exit 2
  fi
  echo "  ${method} ${path} -> HTTP ${status}"
}

echo "Probing ${BASE_URL} (log: ${LOG_FILE})"
send GET  /api/v1/xrd/hostname
send GET  /api/auth/user
send POST /api/v1/xrd/ls/paged \
  "{\"path\":\"/\",\"access_token\":\"${SENTINEL}\",\"nested\":{\"refresh_token\":\"${SENTINEL}\",\"client_secret\":\"${SENTINEL}\"}}"

sleep "$WAIT"

leaks="$(grep -c -F -- "$SENTINEL" "$LOG_FILE" || true)"
seen="$(grep -c -F -- "$MARKER" "$LOG_FILE" || true)"

if [ "$leaks" -gt 0 ]; then
  echo "FAIL: the sentinel appears on ${leaks} log line(s):"
  grep -n -F -- "$SENTINEL" "$LOG_FILE" | cut -c1-240
  exit 1
fi

if [ "$seen" -eq 0 ]; then
  echo "INCONCLUSIVE: the probe requests are not in '${LOG_FILE}'."
  echo "  Enable server.debug and debug-level logging, or point LOG_PROBE_LOG_FILE at the right file."
  exit 2
fi

echo "PASS: ${seen} log line(s) recorded the probe requests; none contains the sentinel."
