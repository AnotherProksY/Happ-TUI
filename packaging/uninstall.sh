#!/bin/sh
# Removes happ-tui from $PREFIX (default /usr/local). Run with sudo.
#
#   sudo ./uninstall.sh          keep your config and groups
#   sudo ./uninstall.sh --purge  also delete the system service's copy in
#                                /etc/happ-tui and /var/lib/happ-tui
#
# Files in your home directory (~/.config/happ-tui, ~/.local/share/happ-tui,
# ~/.local/state/happ-tui) are never removed.
set -eu

PREFIX=${PREFIX:-/usr/local}
UNIT=/etc/systemd/system/happ-tui-core.service
PURGE=no

case ${1:-} in
"") ;;
--purge) PURGE=yes ;;
*)
	sed -n '2,10p' "$0" | sed 's/^# \{0,1\}//' >&2
	exit 2
	;;
esac

if [ "$(id -u)" -ne 0 ]; then
	echo "run as root: sudo $0" >&2
	exit 1
fi

if [ -f "$UNIT" ]; then
	systemctl disable --now happ-tui-core || true
	rm -f "$UNIT"
	systemctl daemon-reload
fi
if [ -x "$PREFIX/bin/happ-tui" ] && pgrep -x 'happ-tui-core|bushuray-core' >/dev/null 2>&1; then
	"$PREFIX/bin/happ-tui" die || true
fi

rm -f "$PREFIX/bin/happ-tui"
rm -rf "$PREFIX/lib/happ-tui" "$PREFIX/share/licenses/happ-tui" "$PREFIX/share/doc/happ-tui"
if id -u happ-tui-xray >/dev/null 2>&1; then
	userdel happ-tui-xray || true
fi
echo "happ-tui removed from $PREFIX"

if [ "$PURGE" = yes ]; then
	rm -rf /etc/happ-tui /var/lib/happ-tui
	echo "removed /etc/happ-tui and /var/lib/happ-tui"
elif [ -e /etc/happ-tui ] || [ -e /var/lib/happ-tui ]; then
	echo "kept /etc/happ-tui and /var/lib/happ-tui; they contain your subscription links,"
	echo "delete them with: sudo rm -rf /etc/happ-tui /var/lib/happ-tui"
fi
echo "kept: ~/.config/happ-tui, ~/.local/share/happ-tui, ~/.local/state/happ-tui"
