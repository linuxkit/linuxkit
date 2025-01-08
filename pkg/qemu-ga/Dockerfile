FROM linuxkit/alpine:5ac048802293564a50414394e1b9896eac528e7d AS build
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
