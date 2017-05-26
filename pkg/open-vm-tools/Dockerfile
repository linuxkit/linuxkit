FROM linuxkit/alpine:dc7d4cdb5932f3e6ff62172c05627816c0d412fa@sha256:0239e6984a9932bfe1b44dc9be6f8e4c29dd7bdb431ba0b040b6d027b1091f08 AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    open-vm-tools 

# Remove apk residuals
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
CMD ["/usr/bin/vmtoolsd"]
LABEL org.mobyproject.config='{"net": "host"}'
