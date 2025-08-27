FROM linuxkit/alpine:6090baae063eb5023c9601966e88df831f789a70 as builder


# checkout and compile containerd
# Update `FROM` in `pkg/containerd/Dockerfile`, `pkg/init/Dockerfile` and
# `test/pkg/containerd/Dockerfile` when changing this.
ENV CONTAINERD_REPO=https://github.com/containerd/containerd.git

# this commit is post-v2.0.3 and includes the fix for the blkdiscard issue
# which is necessary for the root tests to pass when running on Alpine.
# Once it is in semver, likely v2.0.4, we can switch to that.
# See PR https://github.com/containerd/containerd/pull/11330
ENV CONTAINERD_COMMIT=f35b7dae5f1f4f1277e4c39b89acb0a8ea669c48
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
