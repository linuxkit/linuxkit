FROM linuxkit/alpine:146f540f25cd92ec8ff0c5b0c98342a9a95e479e as build

RUN apk add --no-cache bash go git musl-dev linux-headers make patch
# Hack to work around an issue with go on arm64 requiring gcc
RUN [ $(uname -m) = aarch64 ] && apk add --no-cache gcc || true

ENV GOPATH=/go PATH=$PATH:/go/bin
ENV GITBASE=github.com/google
ENV GITREPO=github.com/google/cadvisor
ENV COMMIT=v0.36.0

ADD /static.patch /tmp/

RUN mkdir -p /go/src/${GITBASE} \
    && cd /go/src/${GITBASE} \
    && git clone https://${GITREPO}.git \
    && cd /go/src/${GITREPO} \
    && git checkout ${COMMIT} \
    && patch -p1 build/build.sh </tmp/static.patch \
    && make build \
    && mv cadvisor /usr/bin/


FROM linuxkit/alpine:146f540f25cd92ec8ff0c5b0c98342a9a95e479e AS mirror

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
