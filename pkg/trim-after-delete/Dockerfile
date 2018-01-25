# We need the `fstrim` binary:
FROM linuxkit/alpine:cba395fbc278daee841106801aba1e1bd7e0f2f7 AS mirror
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
LABEL org.mobyproject.config='{"binds": ["/var/run:/var/run", "/var/lib/docker:/var/lib/docker"], "capabilities": ["CAP_SYS_ADMIN"]}'
