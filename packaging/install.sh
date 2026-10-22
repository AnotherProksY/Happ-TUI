#!/bin/sh
# Installs happ-tui into $PREFIX (default /usr/local). Run with sudo.
set -eu

PREFIX=${PREFIX:-/usr/local}
HERE=$(cd "$(dirname "$0")" && pwd)

if [ "$(id -u)" -ne 0 ]; then
	echo "run as root: sudo $0" >&2
	exit 1
fi

# a running core keeps the old binaries and may hold TUN rules; stop it so
# the new version is used on the next start
if [ -x "$PREFIX/bin/happ-tui" ] && pgrep -x happ-tui-core >/dev/null 2>&1; then
	echo "stopping running happ-tui (the VPN connection will be closed)"
	"$PREFIX/bin/happ-tui" die || true
fi

install -Dm755 "$HERE/bin/happ-tui" "$PREFIX/bin/happ-tui"
install -d "$PREFIX/lib/happ-tui"
install -m755 "$HERE/lib/happ-tui/happ-tui-core" "$HERE/lib/happ-tui/xray" "$HERE/lib/happ-tui/v2parser" "$PREFIX/lib/happ-tui/"
install -m644 "$HERE/lib/happ-tui/geoip.dat" "$HERE/lib/happ-tui/geosite.dat" "$PREFIX/lib/happ-tui/"
install -m755 "$HERE/uninstall.sh" "$PREFIX/lib/happ-tui/uninstall.sh"
install -d "$PREFIX/share/licenses/happ-tui" "$PREFIX/share/doc/happ-tui"
install -m644 "$HERE"/share/licenses/happ-tui/* "$PREFIX/share/licenses/happ-tui/"
install -m644 "$HERE/README.md" "$PREFIX/share/doc/happ-tui/README.md"

echo "installed happ-tui $("$PREFIX/bin/happ-tui" version | cut -d' ' -f2) into $PREFIX"
if ! command -v nft >/dev/null 2>&1; then
	echo "note: nft was not found; TUN mode will fall back to iptables and won't block IPv6"
fi
echo "run: happ-tui        (TUN mode: sudo $PREFIX/bin/happ-tui)"
echo "uninstall: sudo $PREFIX/lib/happ-tui/uninstall.sh"
