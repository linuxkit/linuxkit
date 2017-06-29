FROM linuxkit/alpine:bbc3bd5264ecc9bb366c571bb3eeaa670470ec5a AS mirror
RUN apk version
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    chrony
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
RUN mkdir -p /var/run/chrony && chown chrony.chrony /var/run/chrony && chmod 750 /var/run/chrony
CMD ["/usr/sbin/chronyd", "-d" ]
LABEL org.mobyproject.config='{"capabilities": ["CAP_SYS_TIME", "CAP_SETUID", "CAP_SETGID", "CAP_NET_BIND_SERVICE"]}'
