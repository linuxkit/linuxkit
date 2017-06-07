FROM linuxkit/containerd:5749f2e9e65395cc6635229e8da0e0d484320ddf AS qemu
RUN apk add \
    qemu-aarch64 \
    qemu-arm \
    qemu-ppc64le

FROM linuxkit/alpine:451603daf499e3a40308dbf5571dcffed2343ffa AS mirror

RUN apk add --no-cache go musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin

COPY main.go /go/src/binfmt/
RUN go-compile.sh /go/src/binfmt

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=qemu usr/bin/qemu-* usr/bin/
COPY --from=mirror /go/bin/binfmt usr/bin/binfmt
COPY etc/binfmt.d/00_linuxkit.conf etc/binfmt.d/00_linuxkit.conf
CMD ["/usr/bin/binfmt", "-dir", "/etc/binfmt.d/", "-mount", "/binfmt_misc"]
LABEL org.mobyproject.config='{"binds": ["/proc/sys/fs/binfmt_misc:/binfmt_misc"], "readonly": true, "net": "new", "ipc": "new"}'
