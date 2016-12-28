# Tag: b77cfc4ad0033d4366df830ed697afc7bab458a2
FROM mobylinux/alpine-build-c@sha256:53739ea6042cb0ac39cf6e262012c1c4224206b2c9b719569fe7efa3a381348c

ENV pkgname=rng-tools pkgver=5

COPY . .

RUN curl -O -sSL http://downloads.sourceforge.net/project/gkernel/$pkgname/$pkgver/$pkgname-$pkgver.tar.gz
RUN sha256sum -c sha256sums
RUN zcat $pkgname-$pkgver.tar.gz | tar xf -

WORKDIR $pkgname-$pkgver
RUN for p in ../*.patch; do cat $p | patch -p1; done

RUN export LIBS="-largp" && \
  LDFLAGS=-static ./configure \
    --prefix=/usr \
    --libexecdir=/usr/lib/rng-tools \
    --sysconfdir=/etc \
    --disable-silent-rules && \
  make && \
  make DESTDIR=/ install && \
  strip /usr/sbin/rngd

WORKDIR /rootfs

RUN mkdir -p dev proc sys usr/sbin bin

RUN cp -a /usr/sbin/rngd usr/sbin/
RUN cp -a /tini bin/

RUN printf 'FROM scratch\nCOPY . ./\nCMD ["/bin/tini", "/usr/sbin/rngd", "-f"]\n' > Dockerfile

CMD ["tar", "cf", "-", "."]
