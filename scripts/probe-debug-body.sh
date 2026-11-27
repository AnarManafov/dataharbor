#!/usr/bin/env bash
# Debug body logging probe (GH-83).
#
# Sends three unauthenticated requests to a running backend in debug mode and
# reads its log:
#   1. an upload chunk (PUT .../chunk) with a 200 KiB JSON body,
#   2. a file download (GET .../download),
#   3. an upload completion (POST .../complete) with a JSON body several times
#      the body log limit, a sentinel "secret" straddling the limit and another
#      one wholly past it.
#
#   PASS          the chunk and the download have no body log line, and the
#                 completion body is logged truncated with no fragment of the
#                 random part of either sentinel in the log lines this run wrote
#   FAIL          a transfer body was logged, the oversized body was logged
#                 whole or not at all, or a sentinel fragment is in the log (exit 1)
#   INCONCLUSIVE  the requests were not found in the log, so nothing was proven:
#                 debug logging is off, or this is the wrong log file (exit 2)
#
# The backend must run with server.debug: true and logging level debug (the
# development config does both). The probe is read-only: every request is
# rejected with 401 by the session check, after the debug middleware has logged
# it and before it reaches a handler.
#
# Against the compose dev stack the backend logs to stdout only, so capture it
# first and go through the nginx gateway:
#   docker logs -f dataharbor-backend-dev > /tmp/backend.log 2>&1 &
#   DEBUG_BODY_PROBE_BASE_URL=https://localhost DEBUG_BODY_PROBE_LOG_FILE=/tmp/backend.log \
#     ./scripts/probe-debug-body.sh
#
# Environment:
#   DEBUG_BODY_PROBE_BASE_URL  backend base URL     (default https://localhost:22000)
#   DEBUG_BODY_PROBE_LOG_FILE  backend log file     (default app/log/dataharbor_app.log)
#   DEBUG_BODY_PROBE_LIMIT     the backend's server.debug_body_log_limit (default 4096)
#   DEBUG_BODY_PROBE_INSECURE  1 = accept a self-signed certificate (default 1)
#   DEBUG_BODY_PROBE_WAIT      seconds to wait for the log to be written (default 2)

set -euo pipefail

BASE_URL="${DEBUG_BODY_PROBE_BASE_URL:-https://localhost:22000}"
LOG_FILE="${DEBUG_BODY_PROBE_LOG_FILE:-app/log/dataharbor_app.log}"
LIMIT="${DEBUG_BODY_PROBE_LIMIT:-4096}"
INSECURE="${DEBUG_BODY_PROBE_INSECURE:-1}"
WAIT="${DEBUG_BODY_PROBE_WAIT:-2}"

if [ ! -r "$LOG_FILE" ]; then
  echo "INCONCLUSIVE: cannot read log file '$LOG_FILE' (set DEBUG_BODY_PROBE_LOG_FILE)"
  exit 2
fi
case "$LIMIT" in
  ''|*[!0-9]*) echo "INCONCLUSIVE: DEBUG_BODY_PROBE_LIMIT must be a positive integer"; exit 2 ;;
esac

rnd() { od -An -N16 -tx1 /dev/urandom | tr -d ' \n'; }   # 32 hex characters
rand="$(rnd)"
MARKER="dbgprobe-${rand:0:12}"            # not a secret: must appear in the log
UPLOAD_ID="up_${rand}"
# Each sentinel is a fixed, readable prefix plus 32 hex characters of its own
# randomness. The prefix is the same in every run, so it proves nothing and is
# not checked; only the random part is secret. It is drawn fresh per run and
# apart from the marker and upload ID (which are meant to be logged), so a
# fragment of it can reach the log only through this run's request body.
STRADDLE_PREFIX="DBGPROBEcut"
STRADDLE_SECRET="$(rnd)"
PAST_SECRET="$(rnd)"
SENT_STRADDLE="${STRADDLE_PREFIX}${STRADDLE_SECRET}"   # straddles the limit: no fragment may appear
SENT_PAST="DBGPROBEpast${PAST_SECRET}"                 # wholly past the limit

# Remember where the log ends now, so the leak scan below reads only what this
# run appended.
log_start="$(wc -c < "$LOG_FILE" | tr -d ' ')"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

repeat() { # char count
  head -c "$2" /dev/zero | tr '\0' "$1"
}

# 1. Chunk body: JSON, so a logger keyed on content type alone would log it.
{ printf '{"probe":"%s","pad":"' "$MARKER"; repeat c $((200 * 1024)); printf '"}'; } > "$tmp/chunk.json"

# 3. Completion body. The sentinel is the password of a URL under a neutral
#    key: only its shape marks it as a credential, and a cut URL loses that
#    shape, so a truncating logger that redacts after cutting would leak it.
head='{"note":"'"$MARKER"'","pad":"'
mid='","link":"https://alice:'
# Put the limit in the middle of the sentinel's random part, so that both
# halves of the secret are long enough to show up in the leak scan.
pad=$((LIMIT - ${#head} - ${#mid} - ${#STRADDLE_PREFIX} - ${#STRADDLE_SECRET} / 2))
if [ "$pad" -lt 0 ]; then
  echo "INCONCLUSIVE: DEBUG_BODY_PROBE_LIMIT=${LIMIT} is too small for the probe body"
  exit 2
fi
{
  printf '%s' "$head"; repeat p "$pad"; printf '%s%s' "$mid" "$SENT_STRADDLE"
  printf '@example.org/x","later":{"client_secret":"%s"},"more":"' "$SENT_PAST"
  repeat q $((4 * LIMIT)); printf '"}'
} > "$tmp/complete.json"

curl_opts=(--silent --show-error --output /dev/null --max-time 20 --write-out '%{http_code}')
[ "$INSECURE" = "1" ] && curl_opts+=(--insecure)

send() { # name method path [body-file]
  local name="$1" method="$2" path="$3" file="${4:-}" args=()
  if [ -n "$file" ]; then
    args+=(--header 'Content-Type: application/json' --header 'Expect:' --data-binary "@${file}")
  fi
  local status
  if ! status="$(curl "${curl_opts[@]}" --request "$method" \
      --header "X-Debug-Body-Probe: ${MARKER}-${name}" \
      ${args[@]+"${args[@]}"} "${BASE_URL}${path}")"; then
    echo "INCONCLUSIVE: request to ${BASE_URL}${path} failed (is the backend running?)"
    exit 2
  fi
  echo "  ${method} ${path} -> HTTP ${status}"
}

echo "Probing ${BASE_URL} (log: ${LOG_FILE}, limit: ${LIMIT} bytes)"
send chunk    PUT  "/api/v1/xrd/upload/${UPLOAD_ID}/chunk?index=0" "$tmp/chunk.json"
send download GET  "/api/v1/xrd/download?path=/${MARKER}"
send complete POST "/api/v1/xrd/upload/${UPLOAD_ID}/complete" "$tmp/complete.json"

sleep "$WAIT"

# block NAME prints the log lines of the request tagged NAME: from its
# "Request: " line up to the next one. Concurrent traffic on the backend can
# interleave other lines; run the probe on a quiet backend.
block() {
  awk -v tag="${MARKER}-$1" '
    /Request: / { if (found) exit; n = 0 }
    { buf[n++] = $0; if (index($0, tag)) found = 1 }
    END { if (found) for (i = 0; i < n; i++) print buf[i] }
  ' "$LOG_FILE"
}

fail=0
seen=0
for name in chunk download complete; do
  b="$(block "$name")"
  if [ -z "$b" ] || ! grep -q 'Response Status' <<<"$b"; then
    echo "  ${name}: request not found in the log"
    continue
  fi
  seen=$((seen + 1))
  case "$name" in
    chunk|download)
      if grep -q -e 'Request Body:' -e 'Response Body:' <<<"$b"; then
        echo "  ${name}: FAIL, a transfer body was logged:"
        grep -e 'Request Body:' -e 'Response Body:' <<<"$b" | cut -c1-240
        fail=1
      else
        echo "  ${name}: no body log line"
      fi
      ;;
    complete)
      line="$(grep 'Request Body:' <<<"$b" | head -n1 || true)"
      if [ -z "$line" ]; then
        echo "  complete: FAIL, the request body was not logged"
        fail=1
      elif ! grep -q -F "$MARKER" <<<"$line" || ! grep -q -F '(truncated)' <<<"$line"; then
        echo "  complete: FAIL, the body was not logged truncated (is the backend limit ${LIMIT}?):"
        cut -c1-240 <<<"$line"
        fail=1
      elif [ "${#line}" -gt $((LIMIT + 1024)) ]; then
        echo "  complete: FAIL, the body log line is ${#line} bytes, more than the ${LIMIT}-byte limit allows"
        fail=1
      else
        echo "  complete: body logged truncated (${#line}-byte log line)"
      fi
      ;;
  esac
done

if [ "$seen" -eq 0 ]; then
  echo "INCONCLUSIVE: the probe requests are not in '${LOG_FILE}'."
  echo "  Enable server.debug and debug-level logging, or point DEBUG_BODY_PROBE_LOG_FILE at the right file."
  exit 2
fi

# No 8-character window of either sentinel's random part may be in what this
# run appended to the log. The windows lie wholly inside the random part, so a
# sentinel leaked by an earlier run, or the fixed prefix, cannot match. If the
# log shrank (rotated or truncated) since the start, scan the whole file.
log_now="$(wc -c < "$LOG_FILE" | tr -d ' ')"
if [ "$log_now" -ge "$log_start" ]; then
  tail -c +"$((log_start + 1))" "$LOG_FILE" > "$tmp/scan.log"
  scope="the log lines written by this run"
else
  cp "$LOG_FILE" "$tmp/scan.log"
  scope="the log (it shrank during the run, so all of it was scanned)"
fi
leaks=0
for s in "$STRADDLE_SECRET" "$PAST_SECRET"; do
  for ((i = 0; i + 8 <= ${#s}; i++)); do
    if grep -q -F -- "${s:i:8}" "$tmp/scan.log"; then
      echo "FAIL: fragment '${s:i:8}' of a sentinel is in ${scope}:"
      grep -F -- "${s:i:8}" "$tmp/scan.log" | cut -c1-240
      leaks=1
      break
    fi
  done
done

if [ "$fail" -ne 0 ] || [ "$leaks" -ne 0 ]; then
  exit 1
fi
if [ "$seen" -lt 3 ]; then
  echo "INCONCLUSIVE: only ${seen} of 3 probe requests were found in the log."
  exit 2
fi
echo "PASS: transfer bodies were not logged, the oversized body was truncated, no sentinel fragment is in the log."
