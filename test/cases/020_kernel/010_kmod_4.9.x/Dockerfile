# This Dockerfile extracts the kernel headers from the kernel image
# and then compiles a simple hello world kernel module against them.
# In the last stage, it creates a package, which can be used for
# testing.

FROM linuxkit/kernel:4.9.50 AS ksrc

# Extract headers and compile module
FROM linuxkit/alpine:f4f5b333fa1a8433334fcae996d1637173144a72 AS build
RUN apk add build-base

COPY --from=ksrc /kernel-dev.tar /
RUN tar xf kernel-dev.tar

WORKDIR /kmod
COPY ./src/* ./
RUN make all

# Package
FROM alpine:3.5
COPY --from=build /kmod/hello_world.ko /
COPY check.sh /check.sh
ENTRYPOINT ["/bin/sh", "/check.sh"]
