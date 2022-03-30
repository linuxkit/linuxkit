FROM linuxkit/alpine:33063834cf72d563cd8703467836aaa2f2b5a300 AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    apk-tools \
    busybox \
    ca-certificates \
    musl \
    openssh-client \
    openssh-server \
    tini \
    util-linux \
    wireguard-tools \
    && true
RUN mv /out/etc/apk/repositories.upstream /out/etc/apk/repositories

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=mirror /out/ /
COPY etc/ /etc/
COPY usr/ /usr/
RUN mkdir -p /etc/ssh /root/.ssh && chmod 0700 /root/.ssh
CMD ["/sbin/tini", "-s", "/usr/bin/ssh.sh"]
