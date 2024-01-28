FROM linuxkit/alpine:a720d2b1b83556653abd52d61faa0fe020c2b161 AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM linuxkit/alpine:a720d2b1b83556653abd52d61faa0fe020c2b161 AS mirror2
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    acpid
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
COPY --from=mirror /out/ /
COPY --from=mirror2 /out/etc/acpi /etc/acpi

CMD ["/sbin/acpid", "-f", "-d"]
