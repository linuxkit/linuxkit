# Tag: 36aecb5cf4738737634140eec9abebe1f6559a39
FROM mobylinux/alpine-build-c@sha256:d66b9625abc831f28f8c584991a9cb6975e85d3bb3d3768474b592f1cf32a3a6

ARG KERNEL_VERSION=4.9.1

ENV KERNEL_SOURCE=https://www.kernel.org/pub/linux/kernel/v4.x/linux-${KERNEL_VERSION}.tar.xz

RUN curl -fsSL -o linux-${KERNEL_VERSION}.tar.xz ${KERNEL_SOURCE}

RUN cat linux-${KERNEL_VERSION}.tar.xz | tar --absolute-names -xJ &&  mv /linux-${KERNEL_VERSION} /linux

COPY kernel_config /linux/arch/x86/configs/x86_64_defconfig
COPY kernel_config.debug /linux/debug_config

ARG DEBUG=0

RUN if [ $DEBUG -ne "0" ]; then \
    sed -i 's/CONFIG_PANIC_ON_OOPS=y/# CONFIG_PANIC_ON_OOPS is not set/' /linux/arch/x86/configs/x86_64_defconfig; \
    cat /linux/debug_config >> /linux/arch/x86/configs/x86_64_defconfig; \
    fi

# Apply local patches
COPY patches /patches
RUN cd /linux && \
    set -e && for patch in /patches/*.patch; do \
        echo "Applying $patch"; \
        patch -p1 < "$patch"; \
    done

WORKDIR /linux
RUN make defconfig && \
    make oldconfig && \
    make -j "$(getconf _NPROCESSORS_ONLN)" KCFLAGS="-fno-pie"
RUN make INSTALL_MOD_PATH=/tmp/kernel-modules modules_install && \
    make INSTALL_HDR_PATH=/tmp headers_install && \
    ( cd /tmp && tar cf /kernel-headers.tar include ) && \
    ( cd /tmp/kernel-modules && tar cf /kernel-modules.tar . )

RUN printf "KERNEL_SOURCE=${KERNEL_SOURCE}\n" > /kernel-source-info
