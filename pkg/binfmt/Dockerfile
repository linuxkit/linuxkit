FROM linuxkit/alpine:5f6db26ab7bf6a9c452a612e236cc7495408132b@sha256:d009afc85d0b005daf51c8f3026aa552ab997dc47cab43915e9dc761accae086 AS qemu
RUN apk add \
    qemu-aarch64 \
    qemu-arm \
    qemu-ppc64le

FROM linuxkit/go-compile:4513068d9a7e919e4ec42e2d7ee879ff5b95b7f5@sha256:bdfadbe3e4ec699ca45b67453662321ec270f2d1a1dbdbf09625776d3ebd68c5 AS build
COPY main.go main.go
RUN echo "gofmt..." && test -z $(gofmt -s -l main.go) && \
    echo "go vet..." && test -z $(GOOS=linux go tool vet -printf=false main.go) && \
    echo "golint..." && test -z $(golint main.go)
RUN go build -o usr/bin/binfmt -buildmode pie -ldflags "-s -w -extldflags -static" main.go

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=qemu usr/bin/qemu-* usr/bin/
COPY --from=build usr/bin/binfmt usr/bin/binfmt
COPY etc/binfmt.d/00_linuxkit.conf etc/binfmt.d/00_linuxkit.conf
CMD ["/usr/bin/binfmt", "-dir", "/etc/binfmt.d/", "-mount", "/binfmt_misc"]
LABEL org.mobyproject.config='{"binds": ["/proc/sys/fs/binfmt_misc:/binfmt_misc"], "readonly": true}'
