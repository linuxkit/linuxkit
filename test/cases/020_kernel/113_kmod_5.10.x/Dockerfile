# This Dockerfile extracts the kernel headers from the kernel image
# and then compiles a simple hello world kernel module against them.
# In the last stage, it creates a package, which can be used for
# testing.

FROM linuxkit/kernel:5.10.104 AS ksrc

# Extract headers and compile module
FROM linuxkit/kernel:5.10.104-builder AS build
RUN apk add build-base elfutils-dev

COPY --from=ksrc /kernel-dev.tar /
RUN tar xf kernel-dev.tar

WORKDIR /kmod
COPY ./src/* ./
RUN make all

# Package
FROM alpine:3.13
COPY --from=build /kmod/hello_world.ko /
COPY check.sh /check.sh
ENTRYPOINT ["/bin/sh", "/check.sh"]
