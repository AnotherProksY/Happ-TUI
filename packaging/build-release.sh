#!/bin/sh
# Builds dist/happ-tui-<version>-linux-x86_64.tar.gz with happ-tui, its core,
# xray and v2parser. Needs go, cargo, git, curl, unzip and sha256sum.
# Set V2PARSER_TARGET=x86_64-unknown-linux-musl (with that rustup target
# installed) for a static v2parser that runs on any distribution.
set -eu

VERSION=${VERSION:-0.3.0}
XRAY_VERSION=26.3.27
XRAY_SHA256=23cd9af937744d97776ee35ecad4972cf4b2109d1e0fe6be9930467608f7c8ae
V2PARSER_REPO=https://github.com/Keivan-sf/v2-uri-parser
V2PARSER_COMMIT=ce439bafab136f52bde5d7366685543ad3f63a58
V2PARSER_TARGET=${V2PARSER_TARGET:-}

if [ "$(uname -m)" != "x86_64" ]; then
	echo "only x86_64 is supported" >&2
	exit 1
fi

ROOT=$(cd "$(dirname "$0")/.." && pwd)
NAME=happ-tui-$VERSION
WORK=$ROOT/dist/build
STAGE=$WORK/$NAME
LIB=$STAGE/libexec/happ-tui

rm -rf "$STAGE"
mkdir -p "$STAGE/bin" "$LIB" "$STAGE/share/licenses/happ-tui"

echo "==> happ-tui $VERSION"
cd "$ROOT"
CGO_ENABLED=0 go build -trimpath \
	-ldflags "-s -w -X github.com/AnotherProksY/Happ-TUI/global.Version=v$VERSION" \
	-o "$STAGE/bin/happ-tui" .
cd "$ROOT/core"
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$LIB/happ-tui-core" .

echo "==> v2parser $V2PARSER_COMMIT"
if [ ! -d "$WORK/v2parser/.git" ]; then
	git clone -q "$V2PARSER_REPO" "$WORK/v2parser"
fi
cd "$WORK/v2parser"
git fetch -q origin
git checkout -q "$V2PARSER_COMMIT"
if [ -n "$V2PARSER_TARGET" ]; then
	cargo build -q --release --locked --target "$V2PARSER_TARGET"
	install -m755 "target/$V2PARSER_TARGET/release/v2parser" "$LIB/v2parser"
else
	cargo build -q --release --locked
	install -m755 target/release/v2parser "$LIB/v2parser"
fi

echo "==> xray $XRAY_VERSION"
ZIP=$WORK/Xray-linux-64-$XRAY_VERSION.zip
if [ ! -f "$ZIP" ]; then
	curl -fsSL -o "$ZIP" "https://github.com/XTLS/Xray-core/releases/download/v$XRAY_VERSION/Xray-linux-64.zip"
fi
echo "$XRAY_SHA256  $ZIP" | sha256sum -c --quiet
rm -rf "$WORK/xray" && mkdir "$WORK/xray"
unzip -q "$ZIP" -d "$WORK/xray"
install -m755 "$WORK/xray/xray" "$LIB/xray"
install -m644 "$WORK/xray/geoip.dat" "$WORK/xray/geosite.dat" "$LIB/"
install -m644 "$WORK/xray/LICENSE" "$STAGE/share/licenses/happ-tui/LICENSE.xray"

install -m644 "$ROOT/LICENSE" "$STAGE/share/licenses/happ-tui/LICENSE"
install -m644 "$ROOT/README.md" "$STAGE/README.md"
install -m755 "$ROOT/packaging/install.sh" "$ROOT/packaging/uninstall.sh" "$STAGE/"
install -m644 "$ROOT/packaging/happ-tui-core.service" "$LIB/"

ARCHIVE=$ROOT/dist/$NAME-linux-x86_64.tar.gz
tar -C "$WORK" --owner=0 --group=0 -czf "$ARCHIVE" "$NAME"
cd "$ROOT/dist"
sha256sum "$(basename "$ARCHIVE")" > "$(basename "$ARCHIVE").sha256"
echo "==> $ARCHIVE"
