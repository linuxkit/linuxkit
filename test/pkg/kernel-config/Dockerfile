FROM linuxkit/alpine:34af9cb1990debd17fae6d4198c62ce3910d9908 AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out alpine-baselayout busybox musl bash

ADD https://raw.githubusercontent.com/docker/docker/master/contrib/check-config.sh /out/check-config.sh
ADD . ./out

FROM scratch
COPY --from=mirror /out /
ENTRYPOINT ["/bin/sh", "/check.sh"]
LABEL org.mobyproject.config='{"readonly": true, "binds": ["/lib/modules:/lib/modules", "/dev:/dev", "/sys:/sys"], "capabilities": ["all"]}'
