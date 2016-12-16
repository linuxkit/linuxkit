# Tag: 1ae7bf8ec49a6537a93fba0c90720c65fa1c6ece
FROM mobylinux/alpine-build-go@sha256:5e9aed92363c25349c2845b9be4a5285e0f56376b8b3ce92c7361bb59e6eeb2d

COPY *.go /go/src/cloud-config/

WORKDIR /go/src/cloud-config

RUN go install --ldflags '-extldflags "-fno-PIC"'

WORKDIR /rootfs

RUN mkdir -p usr/bin etc/sysctl.d cloud_sysctl proc sys

RUN cp /go/bin/cloud-config usr/bin
COPY qemu* usr/bin/
COPY cloud-sysctl.conf etc/sysctl.d/

RUN printf 'FROM scratch\nCOPY . ./\nCMD ["/usr/bin/cloud-config", "-configDir", "/etc/sysctl.d"]\n' > Dockerfile

CMD ["tar", "cf", "-", "."]
