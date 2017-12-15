FROM linuxkit/alpine:d307c8a386fa3f32cddda9409b9687e191cdd6f1 AS build
RUN apk add abuild gcc git

ADD build.sh /
RUN adduser -D -G abuild builder && sudo -u builder /build.sh

FROM linuxkit/alpine:d307c8a386fa3f32cddda9409b9687e191cdd6f1 AS mirror
COPY --from=build /home/builder/*apk /

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --initdb -p /out alpine-baselayout busybox tini
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
