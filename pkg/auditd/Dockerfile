FROM linuxkit/alpine:cba395fbc278daee841106801aba1e1bd7e0f2f7 AS mirror

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
