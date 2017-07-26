FROM linuxkit/alpine:488aa6f5dd2d8121a3c5c5c7a1ecf97c424b96ac AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    e2fsprogs \
    e2fsprogs-extra \
    btrfs-progs \
    xfsprogs \
    xfsprogs-extra \
    musl \
    sfdisk \
    util-linux \
    && true
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM linuxkit/alpine:488aa6f5dd2d8121a3c5c5c7a1ecf97c424b96ac AS build

RUN apk add --no-cache go musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin

COPY extend.go /go/src/extend/
RUN go-compile.sh /go/src/extend

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
COPY --from=build /go/bin/extend usr/bin/extend
CMD ["/usr/bin/extend"]
LABEL org.mobyproject.config='{"binds": ["/dev:/dev"], "capabilities": ["CAP_SYS_ADMIN", "CAP_MKNOD"], "net": "new", "ipc": "new"}'
