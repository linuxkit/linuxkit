FROM aarch64/alpine:3.5
RUN \
  apk update && apk upgrade && \
  apk add \
    argp-standalone \
    automake \
    bash \
    bsd-compat-headers \
    build-base \
    cmake \
    curl \
    gcc \
    git \
    libc-dev \
    linux-headers \
    make \
    musl-dev \
    patch \
    util-linux-dev \
    vim \
  && true

COPY compile.sh /usr/bin/

ENTRYPOINT ["/usr/bin/compile.sh"]
