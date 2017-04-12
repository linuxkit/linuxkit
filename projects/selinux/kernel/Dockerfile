FROM linuxkit/alpine-build-kernel:cfdd576c36a52ed2dd62f237f79eeedc2dd3697b@sha256:3fe08db373a9373ba1616a485858f01ebd2d7a3cb364a099d0ed8b45fa419da2

ARG KERNEL_VERSION
ARG DEBUG=0

ENV KERNEL_SOURCE=https://www.kernel.org/pub/linux/kernel/v4.x/linux-${KERNEL_VERSION}.tar.xz

RUN curl -fsSL -o linux-${KERNEL_VERSION}.tar.xz ${KERNEL_SOURCE}

RUN cat linux-${KERNEL_VERSION}.tar.xz | tar --absolute-names -xJ &&  mv /linux-${KERNEL_VERSION} /linux

COPY kernel_config /linux/arch/x86/configs/x86_64_defconfig
COPY kernel_config.debug /linux/debug_config

RUN if [ $DEBUG -ne "0" ]; then \
    sed -i 's/CONFIG_PANIC_ON_OOPS=y/# CONFIG_PANIC_ON_OOPS is not set/' /linux/arch/x86/configs/x86_64_defconfig; \
    cat /linux/debug_config >> /linux/arch/x86/configs/x86_64_defconfig; \
    fi

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
