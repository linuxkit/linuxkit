FROM linuxkit/alpine:0c069d0fd7defddb6e03925fcd4915407db0c9e1 AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    git \
    go \
    musl-dev
# Hack to work around an issue wirh go on arm64 requiring gcc
RUN [ $(uname -m) = aarch64 ] && apk add --no-cache --initdb -p /out gcc || true
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
ENV GOPATH=/go PATH=$PATH:/go/bin
COPY --from=mirror /out/ /
COPY --from=mirror /go/bin/ /go/bin/
COPY /compile.sh /compile.sh
ENTRYPOINT ["/compile.sh"]
