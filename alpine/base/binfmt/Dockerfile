# Tag: 2c9434f1c4ff70b102f34a97d2df1a8363a11a65
FROM mobylinux/alpine-build-go@sha256:d528bbf7102e4209bd59ef030d41de9003ab8e42c303956f62b2df47f3e17849

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
