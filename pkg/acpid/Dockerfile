FROM linuxkit/alpine:d497d5be5b78bfd78268f3466cc4080aee5d7389 AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM linuxkit/alpine:d497d5be5b78bfd78268f3466cc4080aee5d7389 AS mirror2
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    acpid
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
COPY --from=mirror /out/ /
COPY --from=mirror2 /out/etc/acpi /etc/acpi
COPY --chown=0:0 --chmod=0755 power-button /etc/acpi/PWRF/00000080

CMD ["/sbin/acpid", "-f", "-d"]
