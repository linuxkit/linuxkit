FROM linuxkit/alpine:cba395fbc278daee841106801aba1e1bd7e0f2f7 AS qemu
RUN apk add \
    qemu-aarch64 \
    qemu-arm \
    qemu-ppc64le

FROM linuxkit/alpine:cba395fbc278daee841106801aba1e1bd7e0f2f7 AS mirror

RUN apk add --no-cache go musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin

COPY main.go /go/src/binfmt/
RUN go-compile.sh /go/src/binfmt

RUN mkdir /binfmt_misc

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=qemu usr/bin/qemu-* usr/bin/
COPY --from=mirror /go/bin/binfmt usr/bin/binfmt
COPY --from=mirror /binfmt_misc /binfmt_misc/
COPY etc/binfmt.d/00_linuxkit.conf etc/binfmt.d/00_linuxkit.conf
CMD ["/usr/bin/binfmt", "-dir", "/etc/binfmt.d/", "-mount", "/binfmt_misc"]
LABEL org.mobyproject.config='{"binds": ["/proc/sys/fs/binfmt_misc:/binfmt_misc"], "readonly": true, "net": "new", "ipc": "new"}'
