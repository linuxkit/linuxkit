FROM aarch64/alpine:3.5
RUN apk --no-cache add --update --upgrade --purge   \
  bsd-compat-headers \
  curl \
  gcc \
  git \
  libc-dev \
  linux-headers \
  util-linux-dev

COPY compile.sh /usr/bin/

ENTRYPOINT ["/usr/bin/compile.sh"]
