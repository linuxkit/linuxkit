FROM linuxkit/alpine:ad35b6ddbc70faa07e59a9d7dee7707c08122e8d AS mirror

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
