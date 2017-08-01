FROM docker:17.06.0-ce AS docker

# Nothing to do in here, just for COPY --from=docker below

FROM linuxkit/alpine:9bcf61f605ef0ce36cc94d59b8eac307862de6e1 AS build

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
