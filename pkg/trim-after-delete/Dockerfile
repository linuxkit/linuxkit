# We need the `fstrim` binary:
FROM linuxkit/alpine:3683c9a66cd4da40bd7d6c7da599b2dcd738b559 AS mirror
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

COPY .  /go/src/trim-after-delete
RUN go-compile.sh /go/src/trim-after-delete

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
COPY --from=mirror /go/bin/trim-after-delete /usr/bin/trim-after-delete
CMD ["/usr/bin/trim-after-delete", "--", "/sbin/fstrim", "/var/lib/docker"]
