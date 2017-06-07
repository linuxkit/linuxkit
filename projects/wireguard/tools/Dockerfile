FROM linuxkit/containerd:5749f2e9e65395cc6635229e8da0e0d484320ddf as tools
RUN echo http://dl-cdn.alpinelinux.org/alpine/edge/testing >> /etc/apk/repositories
RUN \
  apk update && \
  apk upgrade -a && \
  apk add --no-cache wireguard-tools

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=tools /usr/bin/wg /usr/bin/wg-quick /usr/bin/
COPY --from=tools /usr/lib/libmnl.so* /lib/
