FROM linuxkit/alpine:9bcf61f605ef0ce36cc94d59b8eac307862de6e1 AS build
RUN apk add --no-cache --initdb alpine-baselayout make gcc musl-dev

ADD usermode-helper.c .
RUN make usermode-helper

FROM linuxkit/alpine:9bcf61f605ef0ce36cc94d59b8eac307862de6e1 AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out alpine-baselayout busybox musl

# Remove apk residuals. We have a read-only rootfs, so apk is of no use.
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache /out/var/run

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build usermode-helper /sbin/
COPY --from=mirror /out/ /
COPY init /
COPY etc etc/
COPY bin bin/
RUN ln -s /run /var/
