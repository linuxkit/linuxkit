FROM linuxkit/alpine:c608d404c1cb36cce0c7d9303e30b52c9d81ccf0 AS mirror

RUN apk add --no-cache go musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin

COPY main.go /go/src/sysfs/
RUN go-compile.sh /go/src/sysfs

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /go/bin/sysfs /usr/bin/sysfs
COPY etc/ /etc/
CMD ["/usr/bin/sysfs"]
LABEL org.mobyproject.config='{"net": "new", "ipc": "new"}'
