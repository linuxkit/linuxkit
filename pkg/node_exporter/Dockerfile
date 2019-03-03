FROM linuxkit/alpine:518c2ed0f398c5508969ac5e033607201fb419ed as build

RUN apk add --no-cache go git musl-dev make

ENV GOPATH=/go PATH=$PATH:/go/bin
ENV GITREPO=github.com/prometheus/node_exporter
ENV COMMIT=v0.16.0

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
