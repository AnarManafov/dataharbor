#!/bin/bash
# ============================================
# Render test for the SciTokens configuration
# ============================================
# Exercises render-scitokens-config.sh outside of a container:
#   - claim mode (the default) renders username_claim = posix_username
#   - claim mode ignores a stray mapfile, with a warning
#   - mapfile mode renders map_subject / name_mapfile / default_user = ""
#   - the [Global] block matches what each entrypoint asks for (onmissing,
#     audience, base_path) - production MUST deny on a missing token
#   - mapfile mode fails fast on a missing, empty, non-array or invalid mapfile
#   - an unknown XRD_USER_MAPPING value fails fast
#
# Mapfile mode is opt-in, so nothing exercises it day to day. This test keeps it
# from rotting. Run from anywhere:
#   ./docker/xrootd/scripts/test-render-scitokens.sh
#
# Requires: bash, envsubst (gettext). python3 is optional (mapfile JSON check).
# ============================================

set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RENDER_SCRIPT="$SCRIPT_DIR/render-scitokens-config.sh"
TEMPLATE="$SCRIPT_DIR/../configs/scitokens.cfg.tmpl"

WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT

FAILURES=0
TESTS=0

pass() {
    TESTS=$((TESTS + 1))
    echo "  ok   - $1"
}

fail() {
    TESTS=$((TESTS + 1))
    FAILURES=$((FAILURES + 1))
    echo "  FAIL - $1"
    if [ -n "${2:-}" ]; then
        echo "$2" | sed 's/^/         /'
    fi
}

# render <output-file> <mapfile-path> [env assignments...] - runs the renderer in
# a subshell so an exit inside it does not kill this test run.
render() {
    local out="$1"
    local mapfile_path="$2"
    shift 2
    (
        set -e
        # Hermetic: the README invites running this script from a normal shell,
        # where an ambient XRD_USER_MAPPING or SCITOKENS_ONMISSING would decide
        # what the fixtures render and quietly invalidate every assertion.
        unset XRD_USER_MAPPING SCITOKENS_ONMISSING SCITOKENS_AUDIENCE SCITOKENS_BASE_PATH
        export SCITOKENS_TEMPLATE="$TEMPLATE"
        export SCITOKENS_RENDERED="$out"
        export XRD_MAPFILE="$mapfile_path"
        export SCITOKENS_ISSUER="https://id.gsi.de/realms/wl"
        for assignment in "$@"; do
            export "${assignment?}"
        done
        # shellcheck source=./render-scitokens-config.sh
        . "$RENDER_SCRIPT"
        render_scitokens_config
    ) >"$WORK_DIR/log" 2>&1
}

# issuer_block prints the [Issuer ...] section of a rendered config, comments and
# blank lines stripped, so assertions see only effective directives.
issuer_block() {
    sed -n '/^\[Issuer /,$p' "$1" | grep -v '^[[:space:]]*#' | grep -v '^[[:space:]]*$'
}

# global_block prints the [Global] section - everything before the first
# [Issuer ...] header - with comments and blank lines stripped. onmissing lives
# here, so without this helper nothing would notice a production config that
# silently switched from deny to passthrough.
global_block() {
    sed -n '1,/^\[Issuer /p' "$1" | grep -v '^\[Issuer ' | grep -v '^[[:space:]]*#' | grep -v '^[[:space:]]*$'
}

assert_contains() {
    local file="$1" needle="$2" label="$3"
    if grep -qF -- "$needle" "$file"; then
        pass "$label"
    else
        fail "$label" "$(cat "$file")"
    fi
}

# assert_not_contains requires the file to exist and be non-empty first: grep on
# a missing file (exit 2) or an empty one (exit 1) would otherwise "prove" the
# absence of anything, so a test whose only check is a negative one would pass
# even when the renderer produced nothing at all.
assert_not_contains() {
    local file="$1" needle="$2" label="$3"
    if [ ! -s "$file" ]; then
        fail "$label" "expected a non-empty file, got: $file"
    elif grep -qF -- "$needle" "$file"; then
        fail "$label" "$(cat "$file")"
    else
        pass "$label"
    fi
}

echo "Rendering SciTokens config in every supported mode"
echo ""

# --------------------------------------------
# 1. Claim mode (the default)
# --------------------------------------------
echo "claim mode (default, no XRD_USER_MAPPING set):"
: >"$WORK_DIR/no-mapfile"
rm -f "$WORK_DIR/no-mapfile"
if render "$WORK_DIR/claim.cfg" "$WORK_DIR/no-mapfile"; then
    pass "renders successfully"
    issuer_block "$WORK_DIR/claim.cfg" >"$WORK_DIR/claim.block"
    assert_contains "$WORK_DIR/claim.block" "username_claim = posix_username" "maps users from the posix_username claim"
    assert_not_contains "$WORK_DIR/claim.block" "name_mapfile" "no mapfile directive"
    assert_not_contains "$WORK_DIR/claim.block" "default_user" "no default_user directive"
    assert_contains "$WORK_DIR/claim.cfg" "issuer = https://id.gsi.de/realms/wl" "issuer substituted"
    assert_not_contains "$WORK_DIR/claim.cfg" '${' "every template variable substituted"
else
    fail "renders successfully" "$(cat "$WORK_DIR/log")"
fi
echo ""

# --------------------------------------------
# 2. Claim mode with a stray mapfile - warn, do not fail
# --------------------------------------------
echo "claim mode with a stray mapfile mounted:"
cat >"$WORK_DIR/stray-mapfile" <<'MAPFILE'
[{"sub": "a.manafov", "result": "manafov"}]
MAPFILE
if render "$WORK_DIR/stray.cfg" "$WORK_DIR/stray-mapfile" "XRD_USER_MAPPING=claim"; then
    pass "renders successfully (mapfile ignored)"
    assert_contains "$WORK_DIR/log" "IGNORED" "warns that the mapfile is ignored"
    issuer_block "$WORK_DIR/stray.cfg" >"$WORK_DIR/stray.block"
    assert_contains "$WORK_DIR/stray.block" "username_claim = posix_username" "still maps from the posix_username claim"
    assert_not_contains "$WORK_DIR/stray.block" "name_mapfile" "mapfile not wired into the config"
else
    fail "renders successfully (mapfile ignored)" "$(cat "$WORK_DIR/log")"
fi
echo ""

# --------------------------------------------
# 3. Mapfile mode
# --------------------------------------------
echo "mapfile mode (XRD_USER_MAPPING=mapfile):"
cat >"$WORK_DIR/mapfile" <<'MAPFILE'
[
  {"sub": "a.manafov", "result": "manafov"},
  {"sub": "testuser1", "result": "testuser1"}
]
MAPFILE
if render "$WORK_DIR/mapfile.cfg" "$WORK_DIR/mapfile" "XRD_USER_MAPPING=mapfile"; then
    pass "renders successfully"
    issuer_block "$WORK_DIR/mapfile.cfg" >"$WORK_DIR/mapfile.block"
    assert_contains "$WORK_DIR/mapfile.block" "map_subject    = true" "maps the sub claim"
    assert_contains "$WORK_DIR/mapfile.block" "name_mapfile   = $WORK_DIR/mapfile" "points at the mapfile"
    assert_contains "$WORK_DIR/mapfile.block" 'default_user   = ""' "denies unmapped tokens"
    assert_not_contains "$WORK_DIR/mapfile.block" "username_claim" "no claim directive"
    # Counted without python3, which the production image does not ship.
    assert_contains "$WORK_DIR/log" "static mapfile (2 entries)" "counts the mapfile entries"
else
    fail "renders successfully" "$(cat "$WORK_DIR/log")"
fi
echo ""

# --------------------------------------------
# 4. The [Global] block each entrypoint asks for
# --------------------------------------------
# onmissing is the authorization switch. It is now an env-substituted value, so
# nothing but an explicit assertion stops a stray 'passthrough' from shipping.
# Both entrypoints leave it at the renderer's `deny`; the dev entrypoint's only
# real override is base_path=/data. The second case therefore sets a non-default
# onmissing EXPLICITLY, so the substitution path itself stays covered.
echo "production and development entrypoint defaults (both deny):"
if render "$WORK_DIR/prod.cfg" "$WORK_DIR/no-mapfile"; then
    pass "renders successfully"
    global_block "$WORK_DIR/prod.cfg" >"$WORK_DIR/prod.global"
    assert_contains "$WORK_DIR/prod.global" "onmissing = deny" "denies requests without a valid token"
    assert_not_contains "$WORK_DIR/prod.global" "passthrough" "never falls back to passthrough"
    assert_contains "$WORK_DIR/prod.global" "audience = https://id.gsi.de/realms/wl" "audience defaults to the issuer"
    issuer_block "$WORK_DIR/prod.cfg" >"$WORK_DIR/prod.block"
    assert_contains "$WORK_DIR/prod.block" "base_path = /" "exports / (oss.localroot)"
else
    fail "renders successfully" "$(cat "$WORK_DIR/log")"
fi
echo ""

echo "development entrypoint env (base_path=/data) with an explicit onmissing override:"
if render "$WORK_DIR/dev.cfg" "$WORK_DIR/no-mapfile" \
    "SCITOKENS_BASE_PATH=/data" "SCITOKENS_ONMISSING=passthrough" \
    "SCITOKENS_AUDIENCE=https://dataharbor.example"; then
    pass "renders successfully"
    issuer_block "$WORK_DIR/dev.cfg" >"$WORK_DIR/dev.block"
    assert_contains "$WORK_DIR/dev.block" "base_path = /data" "exports /data (the dev entrypoint's only override)"
    global_block "$WORK_DIR/dev.cfg" >"$WORK_DIR/dev.global"
    assert_contains "$WORK_DIR/dev.global" "onmissing = passthrough" "a non-default onmissing is substituted, not ignored"
    assert_contains "$WORK_DIR/dev.global" "audience = https://dataharbor.example" "audience override honoured"
else
    fail "renders successfully" "$(cat "$WORK_DIR/log")"
fi
echo ""

# --------------------------------------------
# 5. Failure modes
# --------------------------------------------
echo "failure modes:"

if render "$WORK_DIR/bad.cfg" "$WORK_DIR/mapfile" "XRD_USER_MAPPING=hybrid"; then
    fail "unknown XRD_USER_MAPPING value exits non-zero" "$(cat "$WORK_DIR/log")"
else
    pass "unknown XRD_USER_MAPPING value exits non-zero"
    assert_contains "$WORK_DIR/log" "Invalid XRD_USER_MAPPING" "reports the invalid value"
fi

# onmissing is an authorization switch, so a typo must fail loudly rather than
# leave the plugin on a default the operator never chose.
if render "$WORK_DIR/bad.cfg" "$WORK_DIR/no-mapfile" "SCITOKENS_ONMISSING=passthru"; then
    fail "unknown SCITOKENS_ONMISSING value exits non-zero" "$(cat "$WORK_DIR/log")"
else
    pass "unknown SCITOKENS_ONMISSING value exits non-zero"
    assert_contains "$WORK_DIR/log" "Invalid SCITOKENS_ONMISSING" "reports the invalid value"
fi

if render "$WORK_DIR/allow.cfg" "$WORK_DIR/no-mapfile" "SCITOKENS_ONMISSING=allow_public"; then
    pass "allow_public is accepted"
    assert_contains "$WORK_DIR/log" "does not deny requests that carry no valid token" "warns about the non-default onmissing"
else
    fail "allow_public is accepted" "$(cat "$WORK_DIR/log")"
fi

# base_path is client-visible, not a container directory; a relative value is
# always a mistake.
if render "$WORK_DIR/bad.cfg" "$WORK_DIR/no-mapfile" "SCITOKENS_BASE_PATH=data"; then
    fail "relative SCITOKENS_BASE_PATH exits non-zero" "$(cat "$WORK_DIR/log")"
else
    pass "relative SCITOKENS_BASE_PATH exits non-zero"
    assert_contains "$WORK_DIR/log" "must be an absolute path" "reports the relative path"
fi

if render "$WORK_DIR/bad.cfg" "$WORK_DIR/does-not-exist" "XRD_USER_MAPPING=mapfile"; then
    fail "missing mapfile exits non-zero" "$(cat "$WORK_DIR/log")"
else
    pass "missing mapfile exits non-zero"
fi

# The mapfile checks below must hold WITHOUT python3: the production image ships
# no interpreter, so validation gated on one would never run where it matters.
echo 'not json' >"$WORK_DIR/broken-mapfile"
if render "$WORK_DIR/bad.cfg" "$WORK_DIR/broken-mapfile" "XRD_USER_MAPPING=mapfile"; then
    fail "invalid mapfile JSON exits non-zero" "$(cat "$WORK_DIR/log")"
else
    pass "invalid mapfile JSON exits non-zero"
fi

: >"$WORK_DIR/zero-byte-mapfile"
if render "$WORK_DIR/bad.cfg" "$WORK_DIR/zero-byte-mapfile" "XRD_USER_MAPPING=mapfile"; then
    fail "zero-byte mapfile exits non-zero" "$(cat "$WORK_DIR/log")"
else
    pass "zero-byte mapfile exits non-zero"
    assert_contains "$WORK_DIR/log" "User mapfile is empty" "reports the empty mapfile"
fi

# '[]' is the shape an operator gets from a placeholder: it renders a perfectly
# valid config that denies 100% of users, so it must be rejected outright.
echo '[]' >"$WORK_DIR/empty-array-mapfile"
if render "$WORK_DIR/bad.cfg" "$WORK_DIR/empty-array-mapfile" "XRD_USER_MAPPING=mapfile"; then
    fail "empty mapfile array exits non-zero" "$(cat "$WORK_DIR/log")"
else
    pass "empty mapfile array exits non-zero"
    assert_contains "$WORK_DIR/log" "User mapfile is empty" "reports the empty mapfile"
fi

# A JSON object parses fine but is not a rule list - len() on it would report a
# bogus, reassuring entry count.
echo '{"a.manafov": "manafov"}' >"$WORK_DIR/object-mapfile"
if render "$WORK_DIR/bad.cfg" "$WORK_DIR/object-mapfile" "XRD_USER_MAPPING=mapfile"; then
    fail "non-array mapfile exits non-zero" "$(cat "$WORK_DIR/log")"
else
    pass "non-array mapfile exits non-zero"
fi

# A path containing a shell/Python metacharacter must be diagnosed on its merits,
# never interpolated into an interpreter.
mkdir -p "$WORK_DIR/quoted"
cat >"$WORK_DIR/quoted/o'brien.json" <<'MAPFILE'
[{"sub": "o.brien", "result": "obrien"}]
MAPFILE
if render "$WORK_DIR/quoted.cfg" "$WORK_DIR/quoted/o'brien.json" "XRD_USER_MAPPING=mapfile"; then
    pass "a quote in the mapfile path is not misreported as invalid JSON"
else
    fail "a quote in the mapfile path is not misreported as invalid JSON" "$(cat "$WORK_DIR/log")"
fi

if (
    set -e
    export SCITOKENS_TEMPLATE="$TEMPLATE"
    export SCITOKENS_RENDERED="$WORK_DIR/bad.cfg"
    export SCITOKENS_ISSUER=""
    . "$RENDER_SCRIPT"
    render_scitokens_config
) >"$WORK_DIR/log" 2>&1; then
    fail "missing SCITOKENS_ISSUER exits non-zero" "$(cat "$WORK_DIR/log")"
else
    pass "missing SCITOKENS_ISSUER exits non-zero"
fi

echo ""
if [ "$FAILURES" -gt 0 ]; then
    echo "$FAILURES of $TESTS checks FAILED"
    exit 1
fi
echo "all $TESTS checks passed"
