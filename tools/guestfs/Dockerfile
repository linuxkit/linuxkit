FROM debian:bullseye-slim as base

ARG TARGETARCH

ENV DEBIAN_FRONTEND=noninteractive

FROM base as syslinux-amd64
# syslinux package is only available on amd64; for arm64, use
# the limited subset of syslinux-common
ENV SYSLINUX="syslinux"

FROM base as syslinux-arm64
ENV SYSLINUX_arm64="syslinux-common"

FROM syslinux-${TARGETARCH} as syslinux
RUN apt-get update && \
    apt-get -yq upgrade && \
    apt-get install -yq libguestfs-tools \
        ${SYSLINUX} \
        linux-image-${TARGETARCH} \
        vim
