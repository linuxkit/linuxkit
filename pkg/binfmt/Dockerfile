# Use Debian testing Qemu 4.2.0 until https://bugs.alpinelinux.org/issues/8131 is resolved.
FROM debian@sha256:79f148e13b4c596d4ad7fd617aba3630e37cf04f04538699341ed1267bd61a23 AS qemu
RUN apt-get update && apt-get install -y qemu-user-static && \
    mv /usr/bin/qemu-aarch64-static /usr/bin/qemu-aarch64 && \
    mv /usr/bin/qemu-arm-static /usr/bin/qemu-arm && \
    mv /usr/bin/qemu-ppc64le-static /usr/bin/qemu-ppc64le && \
    mv /usr/bin/qemu-s390x-static /usr/bin/qemu-s390x && \
    mv /usr/bin/qemu-riscv64-static /usr/bin/qemu-riscv64 && \
    rm /usr/bin/qemu-*-static

FROM linuxkit/alpine:33063834cf72d563cd8703467836aaa2f2b5a300 AS mirror

RUN apk add --no-cache go musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin

COPY . /go/src/binfmt/
RUN go-compile.sh /go/src/binfmt

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=qemu usr/bin/qemu-* usr/bin/
COPY --from=mirror /go/bin/binfmt usr/bin/binfmt
COPY etc/binfmt.d/00_linuxkit.conf etc/binfmt.d/00_linuxkit.conf
CMD ["/usr/bin/binfmt"]
