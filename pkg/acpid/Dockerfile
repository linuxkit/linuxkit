FROM linuxkit/alpine:7aea0bb7faf70f35c6b7d2d244c91da1d4b3ea93 AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM linuxkit/alpine:7aea0bb7faf70f35c6b7d2d244c91da1d4b3ea93 AS mirror2
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    acpid
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
COPY --from=mirror /out/ /
COPY --from=mirror2 /out/etc/acpi /etc/acpi

CMD ["/sbin/acpid", "-f", "-d"]
