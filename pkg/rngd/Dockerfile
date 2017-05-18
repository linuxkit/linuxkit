FROM linuxkit/alpine:6336329f15b4166514782eaa555cf0ffd35c519c@sha256:f6c2ce92910b1d6e4e5557850a554f4a3ae9f66c1e89ad86a24d6c6e550f165e AS build
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
RUN mkdir -p /tmp/dev /tmp/proc /tmp/sys


FROM linuxkit/tini:cb32c9b3ceb16505e1d62919cf28c8b52bf6d57e@sha256:1645296b3e155f8cf672f71f8d20b274bf38ee94c39dd1b58f7b18e0163b00b8
ENTRYPOINT []
WORKDIR /
COPY --from=build usr/sbin/rngd usr/sbin/rngd
# Can't create directories, so copy empty dirs from previous context
COPY --from=build tmp/dev dev
COPY --from=build tmp/proc proc
COPY --from=build tmp/sys sys
CMD ["/bin/tini", "/usr/sbin/rngd", "-f"]
