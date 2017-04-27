FROM linuxkit/kernel-compile:1b396c221af673757703258159ddc8539843b02b@sha256:6b32d205bfc6407568324337b707d195d027328dbfec554428ea93e7b0a8299b AS kernel-build

ARG KERNEL_VERSION
ARG KERNEL_SERIES
ARG DEBUG=0

ENV KERNEL_SOURCE=https://www.kernel.org/pub/linux/kernel/v4.x/linux-${KERNEL_VERSION}.tar.xz

RUN curl -fsSL -o linux-${KERNEL_VERSION}.tar.xz ${KERNEL_SOURCE}

RUN cat linux-${KERNEL_VERSION}.tar.xz | tar --absolute-names -xJ &&  mv /linux-${KERNEL_VERSION} /linux

COPY kernel_config-${KERNEL_SERIES} /linux/arch/x86/configs/x86_64_defconfig
COPY kernel_config.debug /linux/debug_config

RUN if [ $DEBUG -ne "0" ]; then \
    sed -i 's/CONFIG_PANIC_ON_OOPS=y/# CONFIG_PANIC_ON_OOPS is not set/' /linux/arch/x86/configs/x86_64_defconfig; \
    cat /linux/debug_config >> /linux/arch/x86/configs/x86_64_defconfig; \
    fi

# Apply local patches
COPY patches-${KERNEL_SERIES} /patches
RUN cd /linux && \
    set -e && for patch in /patches/*.patch; do \
        echo "Applying $patch"; \
        patch -p1 < "$patch"; \
    done

RUN cd /linux && \
    make defconfig && \
    make oldconfig && \
    make -j "$(getconf _NPROCESSORS_ONLN)" KCFLAGS="-fno-pie"
RUN cd /linux && \
    make INSTALL_MOD_PATH=/tmp/kernel-modules modules_install && \
    ( DVER=$(basename $(find /tmp/kernel-modules/lib/modules/ -mindepth 1 -maxdepth 1)) && \
      cd /tmp/kernel-modules/lib/modules/$DVER && \
      rm build source && \
      ln -s /usr/src/linux-headers-$DVER build ) && \
    mkdir -p /tmp/kernel-headers/usr && \
    make INSTALL_HDR_PATH=/tmp/kernel-headers/usr headers_install && \
    ( cd /tmp/kernel-headers && tar cf /kernel-headers.tar usr ) && \
    ( cd /tmp/kernel-modules && tar cf /kernel-modules.tar lib ) && \
    cp vmlinux arch/x86_64/boot/bzImage /

RUN DVER=$(basename $(find /tmp/kernel-modules/lib/modules/ -mindepth 1 -maxdepth 1)) && \
    dir=/tmp/usr/src/linux-headers-$DVER && \
    mkdir -p $dir && \
    cp /linux/.config $dir && \
    cd /linux && \
    cp -a include "$dir" && \
    mkdir -p "$dir"/arch/x86 && cp -a arch/x86/include "$dir"/arch/x86/ && \
    ( cd /tmp && tar cf /kernel-dev.tar usr/src )

RUN printf "KERNEL_SOURCE=${KERNEL_SOURCE}\n" > /kernel-source-info


FROM linuxkit/toybox-media:eee3dd4d72cd784801e95b1781e6c4f9d8a5e5eb@sha256:7f940e687164ee2676e11c61705c79f7dd2d144ee87ad17a494848a7045f5f53
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=kernel-build bzImage /kernel
COPY --from=kernel-build kernel-headers.tar /
COPY --from=kernel-build kernel-modules.tar /kernel.tar
COPY --from=kernel-build kernel-dev.tar /
COPY --from=kernel-build kernel-source-info /
