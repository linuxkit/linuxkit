FROM mobylinux/alpine-build-go:78d9a136c632c66f0767e11ce3c661989d706f21

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
