FROM alpine:edge as tools
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
