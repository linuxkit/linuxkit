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
COPY . .
WORKDIR /build
ENV ELFUTILS_VERSION=0.165
RUN curl -sSL -O https://fedorahosted.org/releases/e/l/elfutils/0.165/elfutils-$ELFUTILS_VERSION.tar.bz2
ENV BCC_COMMIT=eb6cdda7739bfd9b6bce85b2dc4edef798590c21
RUN git clone https://github.com/iovisor/bcc.git && cd bcc && git checkout $BCC_COMMIT
ENV LJSYSCALL_COMMIT=0b266e8f4f751ae894299d24a2d40d16c6cf856f
RUN git clone https://github.com/justincormack/ljsyscall.git && cd ljsyscall && git checkout $LJSYSCALL_COMMIT
