FROM linuxkit/alpine:a720d2b1b83556653abd52d61faa0fe020c2b161 AS mirror
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
