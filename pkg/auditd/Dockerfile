FROM linuxkit/alpine:cdb4e4d12ada4071a6c6a60bf4c14d35171ffae8 AS build
RUN apk add abuild gcc git

ADD build.sh /
RUN adduser -D -G abuild builder && sudo -u builder /build.sh

FROM linuxkit/alpine:cdb4e4d12ada4071a6c6a60bf4c14d35171ffae8 AS mirror
COPY --from=build /home/builder/*apk /

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --initdb -p /out alpine-baselayout tini
RUN apk add --allow-untrusted -p /out /*apk

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

LABEL org.mobyproject.config='{"pid": "host", "binds": ["/var/log:/var/log"], "capabilities": ["CAP_AUDIT_CONTROL", "CAP_AUDIT_READ", "CAP_AUDIT_WRITE", "CAP_SYS_NICE"]}'
