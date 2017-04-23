FROM linuxkit/c-compile:f52f485825c890d581e82a62af6906c1d33d8e5d@sha256:473fd283a090d3546e93915adacf00c69a23ff615c95c968c4a40e8723985853 AS rng-build

COPY . /

ENV pkgname=rng-tools pkgver=5
RUN curl -O -sSL http://downloads.sourceforge.net/project/gkernel/$pkgname/$pkgver/$pkgname-$pkgver.tar.gz
RUN sha256sum -c sha256sums
RUN zcat $pkgname-$pkgver.tar.gz | tar xf -

RUN cd $pkgname-$pkgver && for p in ../*.patch; do cat $p | patch -p1; done

RUN cd $pkgname-$pkgver && \
  export LIBS="-largp" && \
  LDFLAGS=-static ./configure \
    --prefix=/usr \
    --libexecdir=/usr/lib/rng-tools \
    --sysconfdir=/etc \
    --disable-silent-rules && \
  make && \
  make DESTDIR=/ install && \
  strip /usr/sbin/rngd
RUN mkdir -p /tmp/dev /tmp/proc /tmp/sys


FROM linuxkit/tini:6714d66b82b5397f497b2aa05764096ed1ffe7d7@sha256:ba594b96af6195737ce2df702196d7adea2cafde554e18940ee14ad575d27f3b
ENTRYPOINT []
WORKDIR /
COPY --from=rng-build usr/sbin/rngd usr/sbin/rngd
# Can't create directories, so copy empty dirs from previous context
COPY --from=rng-build tmp/dev dev
COPY --from=rng-build tmp/proc proc
COPY --from=rng-build tmp/sys sys
CMD ["/bin/tini", "/usr/sbin/rngd", "-f"]
