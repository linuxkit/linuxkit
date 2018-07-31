FROM alpine:3.7
RUN apk update && apk upgrade -a && \
  apk add --no-cache \
  argp-standalone \
  autoconf \
  automake \
  bison \
  build-base \
  clang \
  clang-dev \
  clang-static \
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

RUN ln -s /usr/lib/cmake/llvm5/ /usr/lib/cmake/llvm; \
    ln -s /usr/include/llvm5/llvm-c/ /usr/include/llvm-c; \
    ln -s /usr/include/llvm5/llvm/ /usr/include/llvm

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
ENV BCC_COMMIT=6972806729da00ecda0235abac61d66c8fad7fad
RUN git clone https://github.com/iovisor/bcc.git && cd bcc && git checkout $BCC_COMMIT
RUN cd bcc && patch -p0 < ../bcc-gnuism.patch && patch -p0 < ../bcc-lua.patch
ENV LJSYSCALL_COMMIT=e587f8c55aad3955dddab3a4fa6c1968037b5c6e
RUN git clone https://github.com/justincormack/ljsyscall.git && cd ljsyscall && git checkout $LJSYSCALL_COMMIT
