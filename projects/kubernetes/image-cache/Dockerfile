FROM docker:17.06.0-ce AS docker

# Nothing to do in here, just for COPY --from=docker below

FROM linuxkit/alpine:ad35b6ddbc70faa07e59a9d7dee7707c08122e8d AS build

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox

# Remove apk residuals. We have a read-only rootfs, so apk is of no use.
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

RUN rmdir /out/var/run && ln -nfs /run /out/var/run

FROM scratch
WORKDIR /
COPY --from=build /out /
COPY --from=docker /usr/local/bin/docker /usr/local/bin/docker
COPY *.tar /images/
ENTRYPOINT [ "/bin/sh", "-c" ]
CMD [ "for image in /images/*.tar ; do docker image load -i $image && rm -f $image ; done" ]
LABEL org.mobyproject.config='{"binds": ["/var/run:/var/run"]}'
