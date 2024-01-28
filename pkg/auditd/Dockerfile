FROM linuxkit/alpine:146f540f25cd92ec8ff0c5b0c98342a9a95e479e AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --initdb -p /out alpine-baselayout apk-tools audit busybox tini

# Remove apk residuals. We have a read-only rootfs, so apk is of no use.
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /

COPY auditd.conf /etc/audit
COPY audit.rules /etc/audit
COPY runaudit.sh /usr/bin

CMD ["/sbin/tini", "/usr/bin/runaudit.sh"]
