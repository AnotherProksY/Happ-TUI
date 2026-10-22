# Maintainer: Fazilov Kamil <k.fazilov.work@gmail.com>
# Based on the bushuray-git PKGBUILD by Keivan-sf <keyvan0082@gmail.com>

pkgname=happ-tui-git
pkgdesc="Xray TUI client compatible with Happ subscriptions (builds happ-tui, happ-tui-core and v2parser)"
pkgver=0.0.0.r0.g0000000
pkgrel=1
arch=("x86_64")
url="https://github.com/AnotherProksY/Happ-TUI"
license=("MIT")
depends=("xray-bin" "nftables")
makedepends=("go" "cargo" "git")
source=(
  "happ-tui::git+https://github.com/AnotherProksY/Happ-TUI.git"
  "v2parser::git+https://github.com/Keivan-sf/v2-uri-parser"
)
sha256sums=("SKIP" "SKIP")

pkgver() {
  cd "$srcdir/happ-tui"
  git describe --tags --long 2>/dev/null | sed 's/^v//;s/-/.r/;s/-/./'
}

prepare() {
  cd "$srcdir/happ-tui"
  go mod download

  cd "$srcdir/happ-tui/core"
  go mod download

  cd "$srcdir/v2parser"
  cargo fetch --locked
}

build() {
  cd "$srcdir/happ-tui"
  go build -trimpath -ldflags "-X github.com/AnotherProksY/Happ-TUI/global.Version=v$pkgver" -o happ-tui .

  cd "$srcdir/happ-tui/core"
  go build -trimpath -o happ-tui-core .

  cd "$srcdir/v2parser"
  cargo build --release --locked
}

package() {
  install -Dm755 "$srcdir/happ-tui/happ-tui" "$pkgdir/usr/bin/happ-tui"
  install -Dm755 "$srcdir/happ-tui/core/happ-tui-core" "$pkgdir/usr/bin/happ-tui-core"
  install -Dm755 "$srcdir/v2parser/target/release/v2parser" "$pkgdir/usr/bin/v2parser"
  install -Dm644 "$srcdir/happ-tui/LICENSE" "$pkgdir/usr/share/licenses/$pkgname/LICENSE"
}
