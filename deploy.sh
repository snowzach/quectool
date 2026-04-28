#!/bin/sh
# QuecTool deploy script. Runs on a workstation; pushes a release tarball
# to an adb-attached Quectel modem and runs the on-device installer.
#
#   ./deploy.sh                                    # download + deploy 'latest' release
#   ./deploy.sh v2.0.0                             # download + deploy a specific tag
#   ./deploy.sh ./quectool-v2.0.0-armv7.tar.gz     # use a local tarball (skip download)
#
# Requirements on the workstation: adb, curl, tar, sha256sum.
# Requirements on the modem: adb shell as root, /usrdata writable.

set -eu

REPO=${QUECTOOL_REPO:-snowzach/quectool}
TMPDIR=${TMPDIR:-/tmp}
WORK=$(mktemp -d "$TMPDIR/quectool-deploy.XXXXXX")
trap 'rm -rf "$WORK"' EXIT INT TERM

err()  { printf '%s\n' "$*" >&2; }
die()  { err "$*"; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || die "missing required tool: $1"; }

need adb
need curl
need tar
need sha256sum

ARG=${1:-latest}

# --- Resolve tarball ---------------------------------------------------------
if [ -f "$ARG" ]; then
    TARBALL=$ARG
    echo "==> using local tarball: $TARBALL"
else
    if [ "$ARG" = "latest" ]; then
        API_URL="https://api.github.com/repos/$REPO/releases/latest"
    else
        API_URL="https://api.github.com/repos/$REPO/releases/tags/$ARG"
    fi
    echo "==> resolving release: $ARG"

    META=$WORK/release.json
    if ! curl -fsSL "$API_URL" -o "$META"; then
        die "failed to fetch release metadata from $API_URL"
    fi

    # Extract asset URLs without depending on jq.
    TAR_URL=$(grep -Eo '"browser_download_url":[[:space:]]*"[^"]*armv7\.tar\.gz"' "$META" \
              | head -n 1 | sed 's/.*"\(https[^"]*\)"$/\1/')
    SHA_URL=$(grep -Eo '"browser_download_url":[[:space:]]*"[^"]*armv7\.tar\.gz\.sha256"' "$META" \
              | head -n 1 | sed 's/.*"\(https[^"]*\)"$/\1/')

    [ -n "$TAR_URL" ] || die "no armv7.tar.gz asset on release $ARG"

    TARBALL=$WORK/$(basename "$TAR_URL")
    SHAFILE=$WORK/$(basename "${SHA_URL:-$TAR_URL.sha256}")

    echo "==> downloading $TAR_URL"
    curl -fsSL "$TAR_URL" -o "$TARBALL"

    if [ -n "$SHA_URL" ]; then
        echo "==> verifying sha256"
        curl -fsSL "$SHA_URL" -o "$SHAFILE"
        ( cd "$WORK" && sha256sum -c "$(basename "$SHAFILE")" ) \
            || die "sha256 mismatch on $TARBALL"
    else
        echo "==> no .sha256 published; skipping checksum verification"
    fi
fi

# --- Push & run on modem -----------------------------------------------------
echo "==> waiting for adb device"
adb wait-for-device

REMOTE_TAR=/tmp/quectool-deploy.tar.gz
REMOTE_DIR=/tmp/quectool-deploy

echo "==> adb push -> $REMOTE_TAR"
adb push "$TARBALL" "$REMOTE_TAR" >/dev/null

echo "==> running install.sh on modem"
adb shell "set -e; rm -rf $REMOTE_DIR; mkdir -p $REMOTE_DIR; \
    cd $REMOTE_DIR && tar xzf $REMOTE_TAR && sh ./install.sh; \
    cd / && rm -rf $REMOTE_DIR $REMOTE_TAR" \
    || die "install.sh failed on modem (see output above)"

echo
echo "Deploy complete."
