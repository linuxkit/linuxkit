FROM linuxkit/alpine:630ee558e4869672fae230c78364e367b8ea67a9 AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    tini
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache
RUN mkdir -p /out/dev /out/proc /out/sys

FROM linuxkit/alpine:630ee558e4869672fae230c78364e367b8ea67a9 AS build
RUN apk add \
    argp-standalone \
    automake \
    gcc \
    linux-headers \
    make \
    musl-dev \
    patch    

COPY . /

ENV pkgname=rng-tools pkgver=5

ADD http://downloads.sourceforge.net/project/gkernel/$pkgname/$pkgver/$pkgname-$pkgver.tar.gz $pkgname-$pkgver.tar.gz
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

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=mirror /out/ /
COPY --from=build usr/sbin/rngd usr/sbin/rngd
CMD ["/sbin/tini", "/usr/sbin/rngd", "-f"]
LABEL org.mobyproject.config='{"capabilities": ["CAP_SYS_ADMIN"], "oomScoreAdj": -800, "readonly": true, "net": "new", "ipc": "new"}'
