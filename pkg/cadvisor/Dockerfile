FROM linuxkit/alpine:96ad1eb5ec262b4cd0eef574cdc0b225b502d9ee as build

RUN apk add --no-cache bash go git musl-dev linux-headers make

ENV GOPATH=/go PATH=$PATH:/go/bin
ENV GITREPO=github.com/google/cadvisor
ENV COMMIT=v0.28.1

ADD /static.patch /tmp/

RUN go get -d ${GITREPO} \
    && cd /go/src/${GITREPO} \
    && git checkout ${COMMIT} \
    && patch -p1 build/build.sh </tmp/static.patch \
    && make build \
    && mv cadvisor /usr/bin/


FROM linuxkit/alpine:96ad1eb5ec262b4cd0eef574cdc0b225b502d9ee AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    curl

# Remove apk residuals
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache


FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
COPY --from=build /usr/bin/cadvisor /usr/bin/cadvisor
COPY /waitfordocker.sh /usr/bin/waitfordocker.sh

ENTRYPOINT ["/usr/bin/waitfordocker.sh", "/usr/bin/cadvisor", "-logtostderr", "--disable_metrics=disk,tcp,udp"]
LABEL org.mobyproject.config='{"pid": "host", "binds": ["/var/lib/docker:/var/lib/docker:ro", "/var/run:/var/run", "/sys:/sys:ro", "/etc/resolv.conf:/etc/resolv.conf:ro"], "capabilities": ["all"]}'
