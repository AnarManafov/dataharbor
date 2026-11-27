#!/usr/bin/env bash
# Graceful shutdown probe (GH-84).
#
# Starts a rate-limited download, sends SIGTERM to the backend while it runs, and
# checks that the backend refuses new connections but lets the download finish.
#
#   PASS          the download completed in full after the stop was requested, and
#                 new connections were refused (and, if a log file is given, both
#                 shutdown lines are in it)
#   FAIL          the download was cut, new connections were still accepted, or a
#                 shutdown log line is missing (exit 1)
#   INCONCLUSIVE  nothing was proven: missing settings, no --stop-backend flag, the
#                 download ended before the stop, the backend had nothing in flight
#                 at the signal, or the stop command failed (exit 2)
#
# This probe is NOT read-only: it runs SHUTDOWN_PROBE_STOP_CMD, which stops the
# backend. It refuses to do that unless it is given --stop-backend. Restart the
# backend afterwards.
#
# The download must still be running when the stop arrives and must be able to
# finish within the backend's server.shutdown_timeout (30s by default). Socket
# and proxy buffers swallow the first 10-20 MB at once, so use a file of about
# 100 MB at 5m (5 MB/s, 20 seconds). A smaller file still passes, but the
# backend's download handler will already have returned when the signal arrives.
#
# Point the probe at nginx, as a browser would. A backend container reached
# directly through a published port lost the data buffered in Docker's port
# proxy when the container stopped (seen on Docker Desktop for macOS); the
# backend cannot prevent that.
#
#   export SHUTDOWN_PROBE_BASE_URL=https://localhost
#   read -rs SHUTDOWN_PROBE_COOKIE && export SHUTDOWN_PROBE_COOKIE
#   SHUTDOWN_PROBE_FILE=/data/big.bin SHUTDOWN_PROBE_RATE=5m \
#   SHUTDOWN_PROBE_STOP_CMD='docker stop -t 40 dataharbor-backend' \
#   SHUTDOWN_PROBE_INSECURE=1 \
#     ./scripts/probe-graceful-shutdown.sh --stop-backend
#
# Environment:
#   SHUTDOWN_PROBE_BASE_URL  backend or nginx URL                        (required)
#   SHUTDOWN_PROBE_COOKIE    value of the dataharbor-session cookie; a leading
#                            "dataharbor-session=" is accepted           (required)
#   SHUTDOWN_PROBE_FILE      XRootD path of the file to download         (required)
#   SHUTDOWN_PROBE_STOP_CMD  command that sends SIGTERM to the backend, e.g.
#                            'docker stop -t 40 dataharbor-backend' or
#                            'kill -TERM <pid>'                          (required)
#   SHUTDOWN_PROBE_RATE      curl --limit-rate for the download (default 1m)
#   SHUTDOWN_PROBE_LOG_FILE  backend log to search for the two shutdown lines; it
#                            must already be receiving the log when the probe starts
#   SHUTDOWN_PROBE_INSECURE  1 = accept a self-signed certificate (default 0)
#
# The cookie is passed to curl on stdin and is never printed.

set -euo pipefail

BASE_URL="${SHUTDOWN_PROBE_BASE_URL:-}"
COOKIE="${SHUTDOWN_PROBE_COOKIE:-}"
FILE="${SHUTDOWN_PROBE_FILE:-}"
STOP_CMD="${SHUTDOWN_PROBE_STOP_CMD:-}"
RATE="${SHUTDOWN_PROBE_RATE:-1m}"
LOG_FILE="${SHUTDOWN_PROBE_LOG_FILE:-}"
INSECURE="${SHUTDOWN_PROBE_INSECURE:-0}"

stop_backend=0
for arg in "$@"; do
  case "$arg" in
    --stop-backend) stop_backend=1 ;;
    -h|--help) sed -n '2,/^set -euo/p' "$0" | sed '$d' | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "INCONCLUSIVE: unknown argument '$arg' (usage: $0 --stop-backend)"; exit 2 ;;
  esac
done

missing=""
[ -n "$BASE_URL" ] || missing="$missing SHUTDOWN_PROBE_BASE_URL"
[ -n "$COOKIE" ] || missing="$missing SHUTDOWN_PROBE_COOKIE"
[ -n "$FILE" ] || missing="$missing SHUTDOWN_PROBE_FILE"
[ -n "$STOP_CMD" ] || missing="$missing SHUTDOWN_PROBE_STOP_CMD"
if [ -n "$missing" ]; then
  echo "INCONCLUSIVE: missing settings:${missing}"
  echo "  See the header of $0 for usage."
  exit 2
fi

if [ "$stop_backend" -ne 1 ]; then
  echo "INCONCLUSIVE: this probe stops the backend and is not read-only; nothing was done."
  echo "  It would download ${FILE} from ${BASE_URL} at ${RATE}, then run:"
  echo "    ${STOP_CMD}"
  echo "  Run again with --stop-backend (make: SHUTDOWN_PROBE_FLAGS=--stop-backend) to proceed."
  exit 2
fi

BASE_URL="${BASE_URL%/}"
# The value itself may end in "=" (base64 padding), so only the exact name counts.
case "$COOKIE" in
  dataharbor-session=*) ;;
  *) COOKIE="dataharbor-session=${COOKIE}" ;;
esac

tmp="$(mktemp -d)"
dl_pid=""
stop_pid=""
cleanup() {
  [ -n "$dl_pid" ] && kill "$dl_pid" 2>/dev/null || true
  rm -rf "$tmp"
}
trap cleanup EXIT

tls_opts=()
[ "$INSECURE" = "1" ] && tls_opts+=(--insecure)

# curl config on stdin keeps the cookie out of the process list and the output.
cookie_cfg() {
  local v="${COOKIE//\\/\\\\}"
  v="${v//\"/\\\"}"
  printf 'header = "Cookie: %s"\n' "$v"
}

echo "Probing ${BASE_URL}: downloading ${FILE} at ${RATE}"

cookie_cfg | curl -K - --silent --show-error --fail \
  ${tls_opts[@]+"${tls_opts[@]}"} \
  --limit-rate "$RATE" --get --data-urlencode "path=${FILE}" \
  --dump-header "$tmp/headers" --output "$tmp/body" \
  "${BASE_URL}/api/v1/xrd/download" 2>"$tmp/dl.err" &
dl_pid=$!

sleep 3
if ! kill -0 "$dl_pid" 2>/dev/null; then
  rc=0
  wait "$dl_pid" || rc=$?
  dl_pid=""
  if [ "$rc" -ne 0 ]; then
    echo "INCONCLUSIVE: the download failed before the stop (curl exit ${rc}); the backend was not stopped."
    sed 's/^/  /' "$tmp/dl.err" | head -5
    echo "  Check SHUTDOWN_PROBE_BASE_URL, the cookie and SHUTDOWN_PROBE_FILE."
  else
    echo "INCONCLUSIVE: the download finished within 3 seconds; the backend was not stopped."
    echo "  Use a larger file or a lower SHUTDOWN_PROBE_RATE (now ${RATE})."
  fi
  exit 2
fi

# Only log lines written from here on count; an earlier shutdown is in the file too.
log_start=0
if [ -n "$LOG_FILE" ] && [ -r "$LOG_FILE" ]; then
  log_start="$(wc -l <"$LOG_FILE" | tr -d ' ')"
fi

echo "  download running; stopping the backend: ${STOP_CMD}"
bash -c "$STOP_CMD" >"$tmp/stop.out" 2>&1 &
stop_pid=$!

sleep 1
health_rc=0
health_status="$(curl --silent --output /dev/null --max-time 5 --write-out '%{http_code}' \
  ${tls_opts[@]+"${tls_opts[@]}"} "${BASE_URL}/api/health")" || health_rc=$?

exit_code=0
failures=0
handler_running=0
handler_idle=0
case "$health_rc:$health_status" in
  0:200|0:2??)
    echo "  FAIL: new connections are still accepted (GET /api/health -> HTTP ${health_status})"
    failures=$((failures + 1)) ;;
  0:502|0:503|0:504)
    echo "  new request refused by the gateway (HTTP ${health_status})" ;;
  0:*)
    echo "  new request got HTTP ${health_status}; treated as not served by the backend" ;;
  *)
    echo "  new request failed to connect (curl exit ${health_rc})" ;;
esac

dl_rc=0
wait "$dl_pid" || dl_rc=$?
dl_pid=""

stop_rc=0
wait "$stop_pid" || stop_rc=$?
stop_pid=""
if [ "$stop_rc" -ne 0 ]; then
  echo "INCONCLUSIVE: the stop command exited with status ${stop_rc}:"
  sed 's/^/  /' "$tmp/stop.out" | head -5
  exit 2
fi

got="$(wc -c <"$tmp/body" | tr -d ' ')"
want="$(tr -d '\r' <"$tmp/headers" | awk -F': *' 'tolower($1)=="content-length"{v=$2} END{print v}')"

if [ "$dl_rc" -ne 0 ]; then
  echo "  FAIL: the download was cut (curl exit ${dl_rc}, ${got} bytes received)"
  sed 's/^/    /' "$tmp/dl.err" | head -3
  echo "  The file must be downloadable at ${RATE} within server.shutdown_timeout;"
  echo "  use a smaller file or a higher SHUTDOWN_PROBE_RATE, or raise the timeout."
  failures=$((failures + 1))
elif [ -z "$want" ]; then
  echo "INCONCLUSIVE: the response has no Content-Length header, so completeness cannot be checked (${got} bytes received)."
  exit 2
elif [ "$got" != "$want" ]; then
  echo "  FAIL: the download is incomplete: ${got} of ${want} bytes"
  failures=$((failures + 1))
else
  echo "  download completed after the stop request: ${got} of ${want} bytes"
fi

if [ -n "$LOG_FILE" ]; then
  if [ ! -r "$LOG_FILE" ]; then
    echo "INCONCLUSIVE: cannot read log file '${LOG_FILE}'"
    exit 2
  fi
  for line in "Shutdown started" "Shutdown complete"; do
    if tail -n "+$((log_start + 1))" "$LOG_FILE" | grep -q -F -- "$line"; then
      echo "  log has '${line}'"
    else
      echo "  FAIL: '${line}' not found in ${LOG_FILE}"
      failures=$((failures + 1))
    fi
  done
  # The backend's own count shows whether the handler was still running when the
  # signal arrived. Zero is not a failure: the handler had already written the
  # rest into the socket buffer, and the backend then waits for that to be delivered.
  in_flight="$(tail -n "+$((log_start + 1))" "$LOG_FILE" | grep -F -- "Shutdown started" | tail -1 \
    | sed -n 's/.*"inFlight": *\([0-9][0-9]*\).*/\1/p')"
  if [ -n "$in_flight" ] && [ "$in_flight" -ge 1 ]; then
    handler_running=1
    echo "  backend reported ${in_flight} request(s) in flight when the shutdown started"
  else
    echo "  backend reported no request in flight when the shutdown started"
    handler_idle=1
  fi
fi

if [ "$failures" -gt 0 ]; then
  echo "FAIL: ${failures} check(s) failed."
elif [ "$handler_idle" -eq 1 ]; then
  echo "INCONCLUSIVE: the download completed, but the backend had nothing in flight when the"
  echo "  signal arrived: socket and proxy buffers had already taken the whole file, so the"
  echo "  drain was not exercised. Use a larger file, about 100 MB at SHUTDOWN_PROBE_RATE=5m."
  exit_code=2
elif [ "$handler_running" -eq 1 ]; then
  echo "PASS: the download drained after SIGTERM and new connections were refused."
else
  echo "PASS (client side only): the download completed after SIGTERM and new connections"
  echo "  were refused. Without SHUTDOWN_PROBE_LOG_FILE the probe cannot tell whether the"
  echo "  backend still had the download in flight when the signal arrived."
fi
echo
echo "Manual follow-up:"
echo "  - A clean drain exits with status 0, a cut one with 1. Check it with"
echo "      docker inspect -f '{{.State.ExitCode}}' <container>"
echo "    or  systemctl show -p ExecMainStatus dataharbor-backend"
echo "  - The probe stopped the backend: start it again."
[ "$failures" -eq 0 ] || exit 1
exit "$exit_code"
