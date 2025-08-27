# Use Debian testing Qemu 4.2.0 until https://bugs.alpinelinux.org/issues/8131 is resolved.
FROM debian@sha256:731dd1380d6a8d170a695dbeb17fe0eade0e1c29f654cf0a3a07f372191c3f4b AS qemu
RUN apt-get update && apt-get install -y qemu-user-static && \
    mv /usr/bin/qemu-aarch64-static /usr/bin/qemu-aarch64 && \
    mv /usr/bin/qemu-arm-static /usr/bin/qemu-arm && \
    mv /usr/bin/qemu-ppc64le-static /usr/bin/qemu-ppc64le && \
    mv /usr/bin/qemu-s390x-static /usr/bin/qemu-s390x && \
    mv /usr/bin/qemu-riscv64-static /usr/bin/qemu-riscv64 && \
    mv /usr/bin/qemu-loongarch64-static /usr/bin/qemu-loongarch64 && \
    rm /usr/bin/qemu-*-static

FROM linuxkit/alpine:6090baae063eb5023c9601966e88df831f789a70 AS mirror

RUN apk add --no-cache go musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin
ARG GOPKGVERSION
ENV ldflags="-X main.Version=$GOPKGVERSION"

COPY . /go/src/binfmt/
RUN go-compile.sh /go/src/binfmt

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=qemu usr/bin/qemu-* usr/bin/
COPY --from=mirror /go/bin/binfmt usr/bin/binfmt
COPY etc/binfmt.d/00_linuxkit.conf etc/binfmt.d/00_linuxkit.conf
CMD ["/usr/bin/binfmt"]
