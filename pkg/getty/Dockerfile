FROM linuxkit/alpine:cba395fbc278daee841106801aba1e1bd7e0f2f7 AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    apk-tools \
    busybox \
    ca-certificates \
    musl \
    tini \
    util-linux \
    wireguard-tools \
    && true
RUN mv /out/etc/apk/repositories.upstream /out/etc/apk/repositories

#
# We require a version of `setsid(1)` which supports the `-w`
# option, which is not available in all implementations (e.g. the
# `busybox` implementation does not support it). When this is run
# as part of a LinuxKit `init` image (rather than as a standalone
# container) we cannot guarantee which version of `setsid` will
# be present once the layers are combined, so we take a copy of
# our own, known good, version for use later.
RUN cp /out/usr/bin/setsid /out/usr/bin/setsid.getty
# we really do not want a rogue inittab here
RUN rm -rf /out/etc/inittab

FROM scratch
ENTRYPOINT ["/sbin/tini","-s","-v","--"]
WORKDIR /
COPY --from=mirror /out/ /
COPY usr/ /usr/
COPY etc/ /etc/
CMD ["/usr/bin/rungetty.sh"]
LABEL org.mobyproject.config='{"pid": "host", "net":"host", "binds": ["/etc/resolv.conf:/etc/resolv.conf", "/run:/run", "/tmp:/tmp", "/etc:/hostroot/etc", "/usr/bin/ctr:/usr/bin/ctr", "/usr/bin/runc:/usr/bin/runc", "/containers:/containers","/var/log:/var/log","/dev:/dev","/sys:/sys"], "capabilities": ["all"]}'
