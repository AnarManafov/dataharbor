#!/bin/bash
# ============================================
# SciTokens configuration renderer
# ============================================
# Shared by docker-entrypoint.sh (development) and docker-entrypoint-prod.sh.
# Source this file, then call render_scitokens_config.
#
# It renders configs/scitokens.cfg.tmpl into $SCITOKENS_RENDERED and selects the
# token-to-Unix-user mapping from XRD_USER_MAPPING:
#
#   claim   (default) -> username_claim = posix_username
#                        XRootD takes the Unix user straight from the token.
#   mapfile (opt-in)  -> map_subject / name_mapfile / default_user = ""
#                        XRootD maps the token 'sub' through a static JSON file.
#
# Both modes fail closed: in claim mode the SciTokens plugin rejects a token
# whose posix_username claim is missing, empty or unsafe; in mapfile mode an
# unmapped 'sub' hits default_user = "" and is denied.
#
# Inputs (environment):
#   XRD_USER_MAPPING    claim | mapfile                    (default: claim)
#   SCITOKENS_ISSUER    OIDC issuer URL                    (required)
#   SCITOKENS_AUDIENCE  expected 'aud' claim               (default: issuer)
#   SCITOKENS_ONMISSING deny | passthrough | allow_public  (default: deny)
#   SCITOKENS_BASE_PATH absolute path prefix for the issuer (default: /)
#
# Paths (overridable, mainly for the render test):
#   SCITOKENS_TEMPLATE  template to render
#   SCITOKENS_RENDERED  output file
#   XRD_MAPFILE         mapfile location inside the container
#
# Exits non-zero on an unknown XRD_USER_MAPPING or SCITOKENS_ONMISSING value, a
# relative SCITOKENS_BASE_PATH, a missing SCITOKENS_ISSUER, or (mapfile mode
# only) a missing, unreadable, empty or malformed mapfile.
# ============================================

SCITOKENS_TEMPLATE="${SCITOKENS_TEMPLATE:-/etc/xrootd/scitokens.cfg.tmpl}"
SCITOKENS_RENDERED="${SCITOKENS_RENDERED:-/etc/xrootd/scitokens_rendered.cfg}"
XRD_MAPFILE="${XRD_MAPFILE:-/etc/xrootd/mapfile}"

# The development entrypoint has no logging helpers of its own; provide plain
# fallbacks so this script behaves the same in both entrypoints.
# `declare -F` matches shell FUNCTIONS only - `type` would also be satisfied by
# a same-named executable on PATH (this file lives in /usr/local/bin), a builtin
# or an alias.
if ! declare -F log_info >/dev/null 2>&1; then
    log_info() { echo "[INFO] $1"; }
fi
if ! declare -F log_ok >/dev/null 2>&1; then
    log_ok() { echo "[OK] $1"; }
fi
if ! declare -F log_warn >/dev/null 2>&1; then
    log_warn() { echo "[WARN] $1"; }
fi
if ! declare -F log_error >/dev/null 2>&1; then
    log_error() { echo "[ERROR] $1"; }
fi

# validate_user_mapfile checks that the mounted mapfile exists, is readable and
# looks like a non-empty JSON array of rules, and sets MAPFILE_ENTRY_COUNT to
# the number of mappings found. Mapfile mode only - a broken, empty or
# non-array mapfile locks every user out, so fail fast.
#
# The checks are deliberately interpreter-free: the production image
# (docker/xrootd/Dockerfile.prod) ships no python3, so validation gated on it
# would never run exactly where it matters most. python3 is used only as an
# extra, stricter JSON check when it happens to be installed.
#
# Runs in the caller's shell (not a subshell) so its exit and log output work.
validate_user_mapfile() {
    local content
    local py_err

    if [ ! -f "$XRD_MAPFILE" ]; then
        log_error "User mapfile not found at $XRD_MAPFILE"
        log_error "Mount it with XRD_MAPFILE_PATH, e.g.:"
        log_error "  XRD_MAPFILE_PATH=/opt/xrootd/mapfile docker compose \\"
        log_error "    -f docker-compose.prod.yml -f docker-compose.mapfile.yml up -d"
        exit 1
    fi

    if [ ! -r "$XRD_MAPFILE" ]; then
        log_error "User mapfile is not readable: $XRD_MAPFILE"
        exit 1
    fi

    content=$(tr -d '[:space:]' <"$XRD_MAPFILE")

    if [ -z "$content" ] || [ "$content" = "[]" ]; then
        log_error "User mapfile is empty: $XRD_MAPFILE"
        log_error "In mapfile mode an empty mapfile denies EVERY token."
        log_error 'Add at least one rule, e.g. [{"sub": "a.manafov", "result": "manafov"}]'
        exit 1
    fi

    case "$content" in
    \[*\]) ;;
    *)
        log_error "User mapfile is not a JSON array of rules: $XRD_MAPFILE"
        log_error 'Expected e.g. [{"sub": "a.manafov", "result": "manafov"}]'
        exit 1
        ;;
    esac

    # Optional stricter check. The mapfile path is passed as an ARGUMENT, never
    # interpolated into the Python source: a path containing a quote would
    # otherwise be misreported as invalid JSON - or executed.
    if command -v python3 >/dev/null 2>&1; then
        if ! py_err=$(python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); sys.exit(0 if isinstance(d, list) else 1)' "$XRD_MAPFILE" 2>&1); then
            log_error "User mapfile is not a valid JSON array: $XRD_MAPFILE"
            if [ -n "$py_err" ]; then
                log_error "  $(echo "$py_err" | tail -n 1)"
            fi
            exit 1
        fi
    fi

    MAPFILE_ENTRY_COUNT=$(grep -o '"result"[[:space:]]*:' "$XRD_MAPFILE" | wc -l | tr -d '[:space:]')
}

# render_scitokens_config renders the SciTokens template for the selected
# user-mapping mode and logs the mode that will be in effect.
render_scitokens_config() {
    local mode="${XRD_USER_MAPPING:-claim}"
    local mapfile_content

    if [ -z "${SCITOKENS_ISSUER:-}" ]; then
        log_error "SCITOKENS_ISSUER environment variable is required"
        log_error "Set it to your OIDC issuer URL (e.g., https://id.gsi.de/realms/wl)"
        exit 1
    fi

    SCITOKENS_AUDIENCE="${SCITOKENS_AUDIENCE:-$SCITOKENS_ISSUER}"
    SCITOKENS_ONMISSING="${SCITOKENS_ONMISSING:-deny}"
    SCITOKENS_BASE_PATH="${SCITOKENS_BASE_PATH:-/}"

    # onmissing decides what happens to a request carrying no valid token, so a
    # typo must never reach the rendered config: the plugin would fall back to
    # its own default rather than the value the operator believes is in effect.
    # Allowlist taken from the SciTokens plugin's own parser.
    case "$SCITOKENS_ONMISSING" in
    deny | passthrough | allow_public) ;;
    *)
        log_error "Invalid SCITOKENS_ONMISSING value: '$SCITOKENS_ONMISSING'"
        log_error "Supported values: deny (default), passthrough, allow_public"
        exit 1
        ;;
    esac

    # base_path is the path prefix the issuer may access as the CLIENT sees it,
    # not a directory inside the container. Production exports / and maps it to
    # /data with oss.localroot, so a well-meant SCITOKENS_BASE_PATH=/data there
    # denies every path. A relative value is always a mistake.
    case "$SCITOKENS_BASE_PATH" in
    /*) ;;
    *)
        log_error "SCITOKENS_BASE_PATH must be an absolute path, got: '$SCITOKENS_BASE_PATH'"
        log_error "It is the client-visible path prefix for this issuer (e.g. / or /data)"
        exit 1
        ;;
    esac

    case "$mode" in
    claim)
        # username_claim implies map_subject and makes default_user irrelevant.
        # A token without a usable posix_username is rejected outright - a
        # mapfile cannot rescue it, which is why the two modes are exclusive.
        SCITOKENS_USER_MAPPING_BLOCK="username_claim = posix_username"

        # Cosmetic warning only. Never let a mapfile this mode deliberately
        # ignores abort the entrypoint under `set -e` - an unreadable file
        # (SELinux denial, root-squashed NFS source) must not restart-loop the
        # container, so guard the read and swallow any failure.
        mapfile_content=""
        if [ -f "$XRD_MAPFILE" ] && [ -r "$XRD_MAPFILE" ]; then
            mapfile_content=$(tr -d '[:space:]' <"$XRD_MAPFILE" 2>/dev/null || true)
        fi
        if [ -n "$mapfile_content" ]; then
            log_warn "A user mapfile is present at $XRD_MAPFILE but XRD_USER_MAPPING=claim"
            log_warn "The mapfile is IGNORED. Set XRD_USER_MAPPING=mapfile to use it."
        fi

        log_ok "User mapping: posix_username claim"
        ;;
    mapfile)
        validate_user_mapfile
        SCITOKENS_USER_MAPPING_BLOCK="map_subject    = true
name_mapfile   = ${XRD_MAPFILE}
default_user   = \"\""

        if [ "$MAPFILE_ENTRY_COUNT" = "0" ]; then
            log_warn "User mapping: static mapfile, but no \"result\" entries were found in $XRD_MAPFILE"
            log_warn "The mapping may be unusable - an unmapped 'sub' is denied (default_user = \"\")"
        else
            log_ok "User mapping: static mapfile ($MAPFILE_ENTRY_COUNT entries)"
        fi
        ;;
    *)
        log_error "Invalid XRD_USER_MAPPING value: '$mode'"
        log_error "Supported values: claim (default), mapfile"
        exit 1
        ;;
    esac

    export SCITOKENS_ONMISSING SCITOKENS_AUDIENCE SCITOKENS_ISSUER \
        SCITOKENS_BASE_PATH SCITOKENS_USER_MAPPING_BLOCK

    if [ ! -f "$SCITOKENS_TEMPLATE" ]; then
        log_error "SciTokens template not found at $SCITOKENS_TEMPLATE"
        exit 1
    fi

    envsubst '${SCITOKENS_ONMISSING} ${SCITOKENS_AUDIENCE} ${SCITOKENS_ISSUER} ${SCITOKENS_BASE_PATH} ${SCITOKENS_USER_MAPPING_BLOCK}' \
        <"$SCITOKENS_TEMPLATE" >"$SCITOKENS_RENDERED"

    chmod 644 "$SCITOKENS_RENDERED"
    chown xrootd:xrootd "$SCITOKENS_RENDERED" 2>/dev/null || true

    log_ok "SciTokens config rendered to $SCITOKENS_RENDERED"
    log_info "  issuer:    $SCITOKENS_ISSUER"
    log_info "  audience:  $SCITOKENS_AUDIENCE"
    log_info "  base_path: $SCITOKENS_BASE_PATH"
    log_info "  onmissing: $SCITOKENS_ONMISSING"

    # Never let a non-default onmissing slip by behind a clean [OK] line.
    if [ "$SCITOKENS_ONMISSING" != "deny" ]; then
        log_warn "SCITOKENS_ONMISSING=$SCITOKENS_ONMISSING - this plugin does not deny requests that carry no valid token"
        log_warn "Production must use deny. Unset SCITOKENS_ONMISSING to get it."
    fi
}
