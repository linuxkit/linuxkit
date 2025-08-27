FROM linuxkit/alpine:6090baae063eb5023c9601966e88df831f789a70 as build

RUN apk add --no-cache go git musl-dev make curl gcc

ENV GOPATH=/go PATH=$PATH:/go/bin
ENV GITBASE=github.com/prometheus
ENV GITREPO=github.com/prometheus/node_exporter
ENV COMMIT=v1.8.2

RUN mkdir -p /go/src/${GITBASE} \
    && cd /go/src/${GITBASE} \
    && git clone https://${GITREPO}.git \
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
