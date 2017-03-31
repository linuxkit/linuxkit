# FROM mor1/alpine-build-kernel-arm64:4614215ee9a7ccedb53cf0e7ab2efe2c99b57ba3
FROM mor1/alpine-build-kernel:cfdd576c36a52ed2dd62f237f79eeedc2dd3697b

ARG KERNEL_VERSION
ARG DEBUG=0

# Fetch Linux source
ENV KERNEL_SOURCE=https://www.kernel.org/pub/linux/kernel/v4.x/linux-${KERNEL_VERSION}.tar.xz
RUN curl -fsSL -o linux-${KERNEL_VERSION}.tar.xz ${KERNEL_SOURCE}
RUN cat linux-${KERNEL_VERSION}.tar.xz | tar --absolute-names -xJ &&  mv /linux-${KERNEL_VERSION} /linux

# Build config
COPY configs /configs
RUN cat                                                 \
      /configs/arm64_defconfig                          \
      /configs/kernel_config                            \
      $([$DEBUG -ne 0 ] && echo /configs/debug_config)  \
    | sort | uniq > /linux/arch/arm64/configs/defconfig

# Apply local patches
COPY patches-4.9 /patches
RUN cd /linux &&                                \
    for patch in /patches/*.patch ; do          \
        echo "Applying $patch";                 \
        patch -p1 < "$patch";                   \
    done

# Configure and build kernel
RUN cd /linux && \
    make defconfig && \
    make oldconfig && \
    make -j "$(getconf _NPROCESSORS_ONLN)" KCFLAGS="-fno-pie"

# Install and extract modules, headers, kernel
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
    cp vmlinux arch/arm64/boot/Image.gz /

RUN DVER=$(basename $(find /tmp/kernel-modules/lib/modules/ -mindepth 1 -maxdepth 1)) && \
    dir=/tmp/usr/src/linux-headers-$DVER && \
    mkdir -p $dir && \
    cp /linux/.config $dir && \
    cd /linux && \
    cp -a include "$dir" && \
    mkdir -p "$dir"/arch/arm64 && cp -a arch/x86/include "$dir"/arch/arm64/ && \
    ( cd /tmp && tar cf /kernel-dev.tar usr/src )

RUN printf "KERNEL_SOURCE=${KERNEL_SOURCE}\n" > /kernel-source-info
