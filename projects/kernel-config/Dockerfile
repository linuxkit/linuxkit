FROM linuxkit/kernel-compile:1b396c221af673757703258159ddc8539843b02b@sha256:6b32d205bfc6407568324337b707d195d027328dbfec554428ea93e7b0a8299b AS kernel-build

ARG KERNEL_VERSION
ARG KERNEL_SERIES
ARG ARCH
ARG DEBUG

ENV KERNEL_SOURCE=https://www.kernel.org/pub/linux/kernel/v4.x/linux-${KERNEL_VERSION}.tar.xz

RUN curl -fsSL -o linux-${KERNEL_VERSION}.tar.xz ${KERNEL_SOURCE}

RUN cat linux-${KERNEL_VERSION}.tar.xz | tar --absolute-names -xJ &&  mv /linux-${KERNEL_VERSION} /linux

RUN mkdir /config
COPY kernel_config.* /config/
COPY *.sh /config/

# Apply local patches
COPY patches-${KERNEL_SERIES} /patches
WORKDIR /linux
RUN set -e && for patch in /patches/*.patch; do \
        echo "Applying $patch"; \
        patch -p1 < "$patch"; \
    done

RUN \
  if [ -n "${DEBUG}" ]; then dbg=kernel_config.debug; else dbg=; fi && \
  /config/makeconfig.sh \
    kernel_config.base \
    "kernel_config.${ARCH}" \
    "kernel_config.${ARCH}.${KERNEL_SERIES}" \
    "kernel_config.${KERNEL_SERIES}" \
    "$dbg"
RUN /config/check-kernel-config.sh /linux/.config

RUN mkdir /out
RUN printf "KERNEL_SOURCE=${KERNEL_SOURCE}\n" > /out/kernel-source-info

# Kernel
RUN make -j "$(getconf _NPROCESSORS_ONLN)" KCFLAGS="-fno-pie" && \
    cp arch/x86_64/boot/bzImage /out/kernel && \
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

FROM linuxkit/toybox-media:eee3dd4d72cd784801e95b1781e6c4f9d8a5e5eb@sha256:7f940e687164ee2676e11c61705c79f7dd2d144ee87ad17a494848a7045f5f53
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=kernel-build /out/* /
