# We need the `fstrim` binary:
FROM linuxkit/alpine:33063834cf72d563cd8703467836aaa2f2b5a300 AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    util-linux

# Remove apk residuals
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

# We also need the Go binary which calls it:
RUN apk add --no-cache go musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin
# Hack to work around an issue with go on arm64 requiring gcc
RUN [ $(uname -m) = aarch64 ] && apk add --no-cache gcc || true

COPY .  /go/src/trim-after-delete
RUN go-compile.sh /go/src/trim-after-delete

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
COPY --from=mirror /go/bin/trim-after-delete /usr/bin/trim-after-delete
CMD ["/usr/bin/trim-after-delete", "--", "/sbin/fstrim", "/var/lib/docker"]
