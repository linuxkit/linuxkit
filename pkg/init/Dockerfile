FROM linuxkit/alpine:630ee558e4869672fae230c78364e367b8ea67a9 AS build
RUN apk add --no-cache --initdb alpine-baselayout make gcc musl-dev

ADD usermode-helper.c .
RUN make usermode-helper

FROM linuxkit/alpine:630ee558e4869672fae230c78364e367b8ea67a9 AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out alpine-baselayout busybox musl

# Remove apk residuals. We have a read-only rootfs, so apk is of no use.
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build usermode-helper /sbin/
COPY --from=mirror /out/ /
COPY init /
COPY etc etc/
COPY bin bin/
