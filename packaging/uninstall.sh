#!/bin/sh
# Removes happ-tui from $PREFIX (default /usr/local). Run with sudo.
# Your config and groups in ~/.config/happ-tui and ~/.local/share/happ-tui are kept.
set -eu

PREFIX=${PREFIX:-/usr/local}

if [ "$(id -u)" -ne 0 ]; then
	echo "run as root: sudo $0" >&2
	exit 1
fi

if [ -x "$PREFIX/bin/happ-tui" ] && pgrep -x 'happ-tui-core|bushuray-core' >/dev/null 2>&1; then
	"$PREFIX/bin/happ-tui" die || true
fi

rm -f "$PREFIX/bin/happ-tui"
rm -rf "$PREFIX/lib/happ-tui" "$PREFIX/share/licenses/happ-tui" "$PREFIX/share/doc/happ-tui"
echo "happ-tui removed from $PREFIX"
echo "kept: ~/.config/happ-tui, ~/.local/share/happ-tui, ~/.local/state/happ-tui"
