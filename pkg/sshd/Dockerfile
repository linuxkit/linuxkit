FROM linuxkit/containerd:5749f2e9e65395cc6635229e8da0e0d484320ddf AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    musl \
    openssh-server \
    tini \
    util-linux \
    && true
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=mirror /out/ /
COPY etc/ /etc/
COPY usr/ /usr/
RUN mkdir -p /etc/ssh /root/.ssh && chmod 0700 /root/.ssh
CMD ["/sbin/tini", "/usr/bin/ssh.sh"]
LABEL org.mobyproject.config='{"pid": "host", "binds": ["/root/.ssh:/root/.ssh", "/etc/resolv.conf:/etc/resolv.conf"], "capabilities": ["all"]}'
