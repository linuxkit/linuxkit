FROM linuxkit/alpine:1b05307ae8152e3d38f79e297b0632697a30c65c as build

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
