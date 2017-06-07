FROM linuxkit/containerd:5749f2e9e65395cc6635229e8da0e0d484320ddf AS mirror

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
