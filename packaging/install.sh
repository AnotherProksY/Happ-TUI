#!/bin/sh
# Installs happ-tui into $PREFIX (default /usr/local). Run with sudo.
#
#   sudo ./install.sh                       install, the core runs as you
#   sudo ./install.sh --service [--user N]  also run the core as a system
#                                           service for user N (default: the
#                                           user running sudo), so TUN mode
#                                           and the CLI work without sudo
set -eu

PREFIX=${PREFIX:-/usr/local}
HERE=$(cd "$(dirname "$0")" && pwd)
LIBDIR=$PREFIX/libexec/happ-tui
# where v0.3 and earlier put the core
OLD_LIBDIR=$PREFIX/lib/happ-tui
UNIT=/etc/systemd/system/happ-tui-core.service
SERVICE=no
SERVICE_USER=${SUDO_USER:-}

usage() {
	sed -n '2,9p' "$0" | sed 's/^# \{0,1\}//' >&2
	exit 2
}

while [ $# -gt 0 ]; do
	case $1 in
	--service) SERVICE=yes ;;
	--user)
		[ $# -ge 2 ] || usage
		SERVICE_USER=$2
		shift
		;;
	*) usage ;;
	esac
	shift
done

if [ "$(id -u)" -ne 0 ]; then
	echo "run as root: sudo $0" >&2
	exit 1
fi
if [ "$SERVICE" = yes ]; then
	if [ -z "$SERVICE_USER" ] || [ "$SERVICE_USER" = root ] || ! id -u "$SERVICE_USER" >/dev/null 2>&1; then
		echo "--service needs the user the VPN is for: --user NAME" >&2
		exit 2
	fi
	command -v systemctl >/dev/null 2>&1 || { echo "--service needs systemd" >&2; exit 1; }
fi

# Cores before v0.4 listen on TCP, which `happ-tui die` no longer speaks;
# SIGTERM makes any core disconnect and remove its TUN rules before exiting.
stop_old_cores() {
	pkill -TERM -x 'happ-tui-core|bushuray-core' 2>/dev/null || return 0
	for _ in 1 2 3 4 5 6 7 8 9 10; do
		pgrep -x 'happ-tui-core|bushuray-core' >/dev/null 2>&1 || return 0
		sleep 0.5
	done
	echo "a core didn't stop in time" >&2
}

# a running core keeps the old binaries and may hold TUN rules; stop it so
# the new version is used on the next start. bushuray-core counts too: it
# listens on the same port, and happ-tui would keep talking to it.
SERVICE_WAS_ACTIVE=no
if [ -f "$UNIT" ] && systemctl is-active --quiet happ-tui-core; then
	echo "stopping the happ-tui-core service (the VPN connection will be closed)"
	systemctl stop happ-tui-core
	SERVICE_WAS_ACTIVE=yes
fi
if pgrep -x 'happ-tui-core|bushuray-core' >/dev/null 2>&1; then
	echo "stopping the running core (the VPN connection will be closed)"
	"$HERE/bin/happ-tui" die >/dev/null 2>&1 || true
	stop_old_cores
fi

install -Dm755 "$HERE/bin/happ-tui" "$PREFIX/bin/happ-tui"
install -d "$LIBDIR"
install -m755 "$HERE/libexec/happ-tui/happ-tui-core" "$HERE/libexec/happ-tui/xray" "$HERE/libexec/happ-tui/v2parser" "$LIBDIR/"
install -m644 "$HERE/libexec/happ-tui/geoip.dat" "$HERE/libexec/happ-tui/geosite.dat" "$HERE/libexec/happ-tui/happ-tui-core.service" "$LIBDIR/"
install -m755 "$HERE/uninstall.sh" "$LIBDIR/uninstall.sh"
rm -rf "$OLD_LIBDIR"
# files keep the labels of where they were unpacked if they were moved; the
# service must run from usr_t, see happ-tui-core.service
if command -v restorecon >/dev/null 2>&1; then
	restorecon -RF "$LIBDIR" "$PREFIX/bin/happ-tui" 2>/dev/null || true
fi
install -d "$PREFIX/share/licenses/happ-tui" "$PREFIX/share/doc/happ-tui"
install -m644 "$HERE"/share/licenses/happ-tui/* "$PREFIX/share/licenses/happ-tui/"
install -m644 "$HERE/README.md" "$PREFIX/share/doc/happ-tui/README.md"

# xray and v2parser run as this user when the core is root
if ! id -u happ-tui-xray >/dev/null 2>&1; then
	NOLOGIN=$(command -v nologin || echo /usr/sbin/nologin)
	useradd --system --user-group --no-create-home --home-dir / --shell "$NOLOGIN" happ-tui-xray
fi

echo "installed happ-tui $("$PREFIX/bin/happ-tui" version | cut -d' ' -f2) into $PREFIX"
if ! command -v nft >/dev/null 2>&1; then
	echo "note: nft was not found; TUN mode will fall back to iptables and won't block IPv6"
fi

if [ "$SERVICE" = yes ]; then
	# the service keeps its own copy in /etc/happ-tui and /var/lib/happ-tui;
	# the files in the home directory stay for when the service is off
	if ! "$LIBDIR/happ-tui-core" migrate --user "$SERVICE_USER"; then
		echo "could not copy the config and groups of $SERVICE_USER; the service was not installed" >&2
		exit 1
	fi
	sed -e "s|@LIBDIR@|$LIBDIR|g" -e "s|@USER@|$SERVICE_USER|g" "$LIBDIR/happ-tui-core.service" > "$UNIT"
	chmod 644 "$UNIT"
	systemctl daemon-reload
	systemctl enable --now happ-tui-core
	echo "the happ-tui-core service runs for $SERVICE_USER"
	echo "run: happ-tui   (TUN mode works without sudo)"
	echo "logs: happ-tui logs, or: journalctl -u happ-tui-core"
else
	if [ "$SERVICE_WAS_ACTIVE" = yes ]; then
		systemctl start happ-tui-core
	fi
	echo "run: happ-tui        (TUN mode: sudo $PREFIX/bin/happ-tui, or install with --service)"
fi
echo "uninstall: sudo $LIBDIR/uninstall.sh"
