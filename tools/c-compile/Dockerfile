FROM alpine:3.5
RUN \
  apk update && apk upgrade && \
  apk add \
  bsd-compat-headers \
  curl \
  gcc \
  git \
  libc-dev \
  linux-headers \
  util-linux-dev \
  && true

COPY compile.sh /usr/bin/

ENTRYPOINT ["/usr/bin/compile.sh"]
