# Tag: d3f266a146a79f35d3bebf252cce62eee25fcfa9
FROM mobylinux/alpine-build-go@sha256:1eca9f912cfa4f59ad817acad76744516b999395c080bc01a50b3a2b3a9a3f5c

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
