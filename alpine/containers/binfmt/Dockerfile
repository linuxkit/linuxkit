# Tag: 1ae7bf8ec49a6537a93fba0c90720c65fa1c6ece
FROM mobylinux/alpine-build-go@sha256:5e9aed92363c25349c2845b9be4a5285e0f56376b8b3ce92c7361bb59e6eeb2d

COPY *.go /go/src/binfmt/

WORKDIR /go/src/binfmt

RUN go install --ldflags '-extldflags "-fno-PIC"'

WORKDIR /rootfs

RUN mkdir -p usr/bin binfmt_misc dev etc/binfmt.d proc sys

RUN cp /go/bin/binfmt usr/bin
COPY qemu* usr/bin/
COPY 00_moby.conf etc/binfmt.d/

RUN printf 'FROM scratch\nCOPY . ./\nCMD ["/usr/bin/binfmt", "-dir", "/etc/binfmt.d/", "-mount", "/binfmt_misc"]\n' > Dockerfile

CMD ["tar", "cf", "-", "."]
