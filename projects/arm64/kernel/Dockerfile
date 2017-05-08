FROM mor1/kernel-compile:3a358614b3822fdc0bc9cc6bfe2302588e4f7d09@sha256:e4f2d2149347c131a1b978632980715082ab090368ee015bac8143c8254fcbad AS kernel-build

ARG KERNEL_VERSION
ARG KERNEL_SERIES
ARG DEBUG
ARG ARCH

ENV KERNEL_SOURCE=https://www.kernel.org/pub/linux/kernel/v4.x/linux-${KERNEL_VERSION}.tar.xz

RUN curl -fsSL -o linux-${KERNEL_VERSION}.tar.xz ${KERNEL_SOURCE}

RUN cat linux-${KERNEL_VERSION}.tar.xz | tar --absolute-names -xJ &&  mv /linux-${KERNEL_VERSION} /linux

COPY ${ARCH}/kernel_config-${KERNEL_SERIES} /linux/arch/${ARCH}/configs/defconfig
COPY ${ARCH}/kernel_config.debug /linux/debug_config

RUN if [ -n "${DEBUG}" ]; then \
    sed -i 's/CONFIG_PANIC_ON_OOPS=y/# CONFIG_PANIC_ON_OOPS is not set/' /linux/arch/${ARCH}/configs/defconfig; \
    cat /linux/debug_config >> /linux/arch/${ARCH}/configs/defconfig; \
    fi

# Apply local patches
COPY ${ARCH}/patches-${KERNEL_SERIES} /patches
WORKDIR /linux
RUN set -e && for patch in /patches/*.patch; do \
        echo "Applying $patch"; \
        patch -p1 < "$patch"; \
    done

RUN mkdir /out

# Kernel
RUN make defconfig && \
    make oldconfig && \
    make -j "$(getconf _NPROCESSORS_ONLN)" KCFLAGS="-fno-pie" && \
    cp arch/${ARCH}/boot/Image.gz /out/kernel && \
    cp System.map /out

# Modules
RUN make INSTALL_MOD_PATH=/tmp/kernel-modules modules_install && \
    ( DVER=$(basename $(find /tmp/kernel-modules/lib/modules/ -mindepth 1 -maxdepth 1)) && \
      cd /tmp/kernel-modules/lib/modules/$DVER && \
      rm build source && \
      ln -s /usr/src/linux-headers-$DVER build ) && \
    ( cd /tmp/kernel-modules && tar cf /out/kernel.tar lib )

# Headers (userspace API)
RUN mkdir -p /tmp/kernel-headers/usr && \
    make INSTALL_HDR_PATH=/tmp/kernel-headers/usr headers_install && \
    ( cd /tmp/kernel-headers && tar cf /out/kernel-headers.tar usr )

# Headers (kernel development)
RUN DVER=$(basename $(find /tmp/kernel-modules/lib/modules/ -mindepth 1 -maxdepth 1)) && \
    dir=/tmp/usr/src/linux-headers-$DVER && \
    mkdir -p $dir && \
    cp /linux/.config $dir && \
    cp /linux/Module.symvers $dir && \
    find . -path './include/*' -prune -o \
           -path './arch/*/include' -prune -o \
           -path './scripts/*' -prune -o \
           -type f \( -name 'Makefile*' -o -name 'Kconfig*' -o -name 'Kbuild*' -o \
                      -name '*.lds' -o -name '*.pl' -o -name '*.sh' \) | \
         tar cf - -T - | (cd $dir; tar xf -) && \
    ( cd /tmp && tar cf /out/kernel-dev.tar usr/src )

RUN printf "KERNEL_SOURCE=${KERNEL_SOURCE}\n" > /out/kernel-source-info


FROM mor1/toybox-media:a469cee93aab04ff69cd8e4f01f22c2dd5ffbe2f@sha256:828ddd0b855682184ac8c2e24f47ebd65d2774fc28a5c10af1561063eb5065a6
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=kernel-build /out/* /
