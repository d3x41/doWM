pkgname=dowm
pkgver=1.0.0
pkgrel=1
pkgdesc="A beautiful tiling and floating x11 window manager"
arch=('x86_64')
url="https://github.com/BobdaProgrammer/doWM" 
license=('MIT')
depends=('xorg-server')
makedepends=('go')
source=("$pkgname::git+file://$PWD")
md5sums=('SKIP')

build() {
  cd "$srcdir/$pkgname"
  go build -o doWM main.go
}

package() {
  cd "$srcdir/$pkgname"

  # Install binary
  install -Dm755 doWM "$pkgdir/usr/bin/doWM"

  # Install session .desktop file
  install -Dm644 install/doWM.desktop "$pkgdir/usr/share/xsessions/doWM.desktop"
}
