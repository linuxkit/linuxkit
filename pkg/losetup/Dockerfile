FROM linuxkit/alpine:5ac048802293564a50414394e1b9896eac528e7d AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out  \
    alpine-baselayout \
    busybox

# Remove apk residuals
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=mirror /out/ /

COPY loopy.sh /usr/bin/loopy
RUN chmod +x /usr/bin/loopy

CMD ["/usr/bin/loopy"]
