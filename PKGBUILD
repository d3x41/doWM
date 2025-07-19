# Maintainer: Sam Inglis <https://github.com/BobdaProgrammer>
pkgname=doWM
pkgrel=1
pkgdesc="A beautiful tiling and floating x11 window manager"
arch=('x86_64')
url="https://github.com/BobdaProgrammer/doWM" 
license=('MIT')
depends=('xorg-server')
makedepends=('go')
source=("doWM.desktop")
build() {
    cd "$srcdir"
    cd ..
    go build -o doWM
}

package() {
  # Install binary
  sudo install -Dm755 "../doWM" "/usr/local/bin/doWM"

  # Install .desktop session file
  sudo install -Dm644 "doWM.desktop" "/usr/share/xsessions/doWM.desktop"
}

# Optional: you can add sha256sums=('SKIP') for local testing
sha256sums=('SKIP')

