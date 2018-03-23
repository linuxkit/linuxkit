# This Dockerfile extracts the kernel headers from the kernel image
# and then compiles a simple hello world kernel module against them.
# In the last stage, it creates a package, which can be used for
# testing.

FROM linuxkit/kernel:4.14.29 AS ksrc

# Extract headers and compile module
FROM linuxkit/alpine:d307c8a386fa3f32cddda9409b9687e191cdd6f1 AS build
RUN apk add build-base libelf-dev

COPY --from=ksrc /kernel-dev.tar /
RUN tar xf kernel-dev.tar

WORKDIR /kmod
COPY ./src/* ./
RUN make all

# Package
FROM alpine:3.7
COPY --from=build /kmod/hello_world.ko /
COPY check.sh /check.sh
ENTRYPOINT ["/bin/sh", "/check.sh"]
