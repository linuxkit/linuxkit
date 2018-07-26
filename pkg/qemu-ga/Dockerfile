FROM linuxkit/alpine:3683c9a66cd4da40bd7d6c7da599b2dcd738b559 AS build
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN mkdir -p /out/var/run
RUN apk add --no-cache --initdb -p /out \
    busybox \
    qemu-guest-agent \
    musl
FROM scratch
WORKDIR /
ENTRYPOINT []
COPY --from=build /out /
CMD ["/usr/bin/qemu-ga", "-p", "/dev/vport0p1"]
