#!/bin/sh
# QuecTool on-modem uninstaller.
#
#   sh uninstall.sh           # stop + disable, leave /usrdata/quectool intact
#   sh uninstall.sh --purge   # also delete /usrdata/quectool entirely
#
# Run on the modem as root.

set -e

if [ "$(id -u)" != "0" ]; then
    echo "uninstall.sh: must run as root" >&2
    exit 1
fi

PURGE=0
if [ "$1" = "--purge" ]; then
    PURGE=1
fi

UNIT_DST=/lib/systemd/system/quectool.service
WANTS_LINK=/lib/systemd/system/multi-user.target.wants/quectool.service
INSTALL_DIR=/usrdata/quectool

remount() {
    mount -o "remount,$1" / 2>/dev/null || true
}

echo "==> stopping service"
systemctl stop quectool 2>/dev/null || true

echo "==> remounting / read-write"
remount rw

echo "==> removing systemd unit"
rm -f "$WANTS_LINK"
rm -f "$UNIT_DST"
systemctl daemon-reload 2>/dev/null || true

echo "==> remounting / read-only"
remount ro

if [ "$PURGE" = "1" ]; then
    echo "==> purging $INSTALL_DIR"
    rm -rf "$INSTALL_DIR"
else
    echo "==> leaving $INSTALL_DIR (config + host_key preserved). Pass --purge to remove."
fi

echo "QuecTool uninstalled."
