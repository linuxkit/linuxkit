# This Dockerfile extracts the kernel headers from the kernel image
# and then compiles a simple hello world kernel module against them.
# In the last stage, it creates a package, which can be used for
# testing.

FROM linuxkit/kernel:6.6.71-819af9d59279506dd2994e7aea1cbbaaebfdb0a2 AS ksrc

# Extract headers and compile module
FROM linuxkit/kernel:6.6.71-builder AS build
RUN apk add build-base elfutils-dev

COPY --from=ksrc /kernel-dev.tar /
RUN tar xf kernel-dev.tar

WORKDIR /kmod
COPY ./src/* ./
RUN make all

# Package
FROM alpine:3.17
COPY --from=build /kmod/hello_world.ko /
COPY check.sh /check.sh
ENTRYPOINT ["/bin/sh", "/check.sh"]
