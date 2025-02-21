FROM linuxkit/alpine:35b33c6b03c40e51046c3b053dd131a68a26c37a as builder


# checkout and compile containerd
# Update `FROM` in `pkg/containerd/Dockerfile`, `pkg/init/Dockerfile` and
# `test/pkg/containerd/Dockerfile` when changing this.
ENV CONTAINERD_REPO=https://github.com/containerd/containerd.git

# this is a commit post-v2.0.2 that contains a fix necesary for tests to pass.
# it should be released with v2.0.3, which can be used once that release is available.
ENV CONTAINERD_COMMIT=3904a5761fadbb999d3fc1a7f7ab7cc6f6d7d850
ENV GOPATH=/go
RUN apk add go git
RUN mkdir -p $GOPATH/src/github.com/containerd && \
  cd $GOPATH/src/github.com/containerd && \
  git clone https://github.com/containerd/containerd.git && \
  cd $GOPATH/src/github.com/containerd/containerd && \
  git checkout $CONTAINERD_COMMIT
RUN apk add --no-cache btrfs-progs-dev gcc libc-dev linux-headers make libseccomp-dev
WORKDIR $GOPATH/src/github.com/containerd/containerd
RUN make binaries EXTRA_FLAGS="-buildmode pie" EXTRA_LDFLAGS='-extldflags "-fno-PIC -static"' BUILDTAGS="static_build no_devmapper"

RUN cp bin/containerd bin/ctr bin/containerd-shim-runc-v2 /usr/bin/
RUN strip /usr/bin/containerd /usr/bin/ctr /usr/bin/containerd-shim-runc-v2

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=builder /usr/bin/containerd /usr/bin/ctr /usr/bin/containerd-shim-runc-v2 /usr/bin/
COPY --from=builder /go/src/github.com/containerd/containerd /go/src/github.com/containerd/containerd
