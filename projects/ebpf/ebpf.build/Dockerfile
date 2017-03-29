FROM alpine:3.5
RUN apk update && apk upgrade -a && \
  apk add --no-cache \
  argp-standalone \
  autoconf \
  automake \
  bison \
  build-base \
  clang \
  clang-dev \
  cmake \
  curl \
  flex-dev \
  fts-dev \
  gettext-dev \
  git \
  iperf \
  libedit-dev \
  libtool \
  llvm \
  llvm-dev \
  llvm-static \
  luajit-dev \
  m4 \
  python \
  zlib-dev \
  && true
WORKDIR /build
COPY . ./
COPY Dockerfile /
COPY error.h /usr/include/
COPY cdefs.h /usr/include/sys/
ENV ELFUTILS_VERSION=0.165
RUN curl -sSL -O https://fedorahosted.org/releases/e/l/elfutils/0.165/elfutils-$ELFUTILS_VERSION.tar.bz2
RUN cat elfutils-$ELFUTILS_VERSION.tar.bz2 | tar xjf - && \
  cd elfutils-$ELFUTILS_VERSION && \
  patch -p1 < ../100-musl-compat.patch && \
  patch -p0 < ../decl.patch && \
  patch -p0 < ../intl.patch
ENV BCC_COMMIT=d4fc95d92ec9bace9bd607dfd1833e9e06457486
RUN git clone https://github.com/iovisor/bcc.git && cd bcc && git checkout $BCC_COMMIT
RUN cd bcc && patch -p0 < ../bcc-gnuism.patch
ENV LJSYSCALL_COMMIT=0b266e8f4f751ae894299d24a2d40d16c6cf856f
RUN git clone https://github.com/justincormack/ljsyscall.git && cd ljsyscall && git checkout $LJSYSCALL_COMMIT
