FROM alpine:3.5
RUN \
  apk update && apk upgrade && \
  apk add \
  argp-standalone \
  curl \
  gcc \
  linux-headers \
  make \
  musl-dev \
  patch \
  && true

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
