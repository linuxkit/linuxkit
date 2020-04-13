FROM linuxkit/alpine:bc528cf9d4065d2e09aa44ff76909b94cfe8d867 AS mirror

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
