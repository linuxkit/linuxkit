# Use Debian testing-slim Qemu 4.2.0 until https://bugs.alpinelinux.org/issues/8131 is resolved.
FROM debian@sha256:80381085775a6e69f0f8d09dd9a39495063e348382eb8d4881ba6f142d24b8ab AS qemu
RUN apt-get update && apt-get install -y qemu-user-static

FROM linuxkit/alpine:7f3944798557de5518a56e3437d7ed982701f224 AS mirror

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
