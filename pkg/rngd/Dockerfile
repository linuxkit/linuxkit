FROM linuxkit/alpine:5f6db26ab7bf6a9c452a612e236cc7495408132b@sha256:d009afc85d0b005daf51c8f3026aa552ab997dc47cab43915e9dc761accae086 AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    tini
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache
RUN mkdir -p /out/dev /out/proc /out/sys

FROM linuxkit/alpine:dae8bcbc6e2cec0a1cc1958dddbc5d6bd3ccf9a0@sha256:02c251d54c4083a596ead8cae92144306b385db0ff961c95a3a620a4c69961ed AS build
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
LABEL org.mobyproject.config='{"capabilities": ["CAP_SYS_ADMIN"], "oomScoreAdj": -800, "readonly": true}'
