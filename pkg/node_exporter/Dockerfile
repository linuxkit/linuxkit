FROM linuxkit/alpine:cba395fbc278daee841106801aba1e1bd7e0f2f7 as build

RUN apk add --no-cache go git musl-dev make

ENV GOPATH=/go PATH=$PATH:/go/bin
ENV GITREPO=github.com/prometheus/node_exporter
ENV COMMIT=v0.15.1

RUN go get -d ${GITREPO} \
    && cd /go/src/${GITREPO} \
    && git checkout ${COMMIT} \
    && CGO_ENABLED=0 make build \
    && mv node_exporter /bin/


FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build /bin/node_exporter /bin/node_exporter

ENTRYPOINT ["/bin/node_exporter", "--path.procfs",  "/host/proc", \
            "--path.sysfs",  "/host/sys", \
            "--collector.filesystem.ignored-mount-points", \
            "^/(sys|proc|dev|host|etc)($|/)"]
LABEL org.mobyproject.config='{"pid": "host", "binds": ["/proc:/host/proc", "/sys:/host/sys", "/:/rootfs"], "capabilities": ["all"]}'

