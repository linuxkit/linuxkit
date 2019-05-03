FROM fedora:30 as qemu
RUN curl -OL https://kojipkgs.fedoraproject.org/packages/qemu/4.0.0/1.fc31/x86_64/qemu-user-static-4.0.0-1.fc31.x86_64.rpm && \
    rpm -ivh qemu-user-static-4.0.0-1.fc31.x86_64.rpm --nodeps && \
    mv /usr/bin/qemu-aarch64-static /usr/bin/qemu-aarch64 && \
    mv /usr/bin/qemu-arm-static /usr/bin/qemu-arm && \
    mv /usr/bin/qemu-ppc64le-static /usr/bin/qemu-ppc64le && \
    mv /usr/bin/qemu-s390x-static /usr/bin/qemu-s390x

FROM linuxkit/alpine:86cd4f51b49fb9a078b50201d892a3c7973d48ec AS mirror

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
CMD ["/usr/bin/binfmt"]
