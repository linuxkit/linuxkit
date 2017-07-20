FROM linuxkit/alpine:9bcf61f605ef0ce36cc94d59b8eac307862de6e1 AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    ca-certificates \
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
LABEL org.mobyproject.config='{"pid": "host", "binds": ["/root/.ssh:/root/.ssh", "/etc/resolv.conf:/etc/resolv.conf", "/run:/run", "/tmp:/tmp", "/etc:/hostroot/etc", "/usr/bin/ctr:/usr/bin/ctr", "/usr/bin/runc:/usr/bin/runc", "/var:/var","/containers:/containers","/dev:/dev","/sys:/sys"], "capabilities": ["all"]}'
