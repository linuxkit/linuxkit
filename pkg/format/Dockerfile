FROM linuxkit/alpine:e2391e0b164c57db9f6c4ae110ee84f766edc430 AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    e2fsprogs \
    e2fsprogs-extra \
    btrfs-progs \
    xfsprogs \
    musl \
    sfdisk \
    util-linux \
    blkid \
    && true
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM linuxkit/alpine:e2391e0b164c57db9f6c4ae110ee84f766edc430 AS build

RUN apk add --no-cache go musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin
# Hack to work around an issue with go on arm64 requiring gcc
RUN [ $(uname -m) = aarch64 ] && apk add --no-cache gcc || true

COPY format.go /go/src/format/
RUN go-compile.sh /go/src/format

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
COPY --from=build /go/bin/format usr/bin/format
CMD ["/usr/bin/format"]
