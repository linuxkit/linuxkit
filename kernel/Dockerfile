FROM linuxkit/alpine:a120ad6aead3fe583eaa20e9b75a05ac1b3487da AS kernel-build
RUN apk add \
    argp-standalone \
    automake \
    bash \
    bc \
    binutils-dev \
    bison \
    build-base \
    curl \
    diffutils \
    flex \
    git \
    gmp-dev \
    gnupg \
    installkernel \
    kmod \
    libelf-dev \
    libressl-dev \
    linux-headers \
    ncurses-dev \
    sed \
    squashfs-tools \
    tar \
    xz \
    xz-dev \
    zlib-dev

# libunwind-dev pkg is missed from arm64 now, below statement will be removed if the pkg is available.
RUN [ $(uname -m) == x86_64 ] && apk add libunwind-dev || true

ARG KERNEL_VERSION
ARG KERNEL_SERIES
ARG EXTRA

ENV KERNEL_SOURCE=https://www.kernel.org/pub/linux/kernel/v4.x/linux-${KERNEL_VERSION}.tar.xz
ENV KERNEL_SHA256_SUMS=https://www.kernel.org/pub/linux/kernel/v4.x/sha256sums.asc
ENV KERNEL_PGP2_SIGN=https://www.kernel.org/pub/linux/kernel/v4.x/linux-${KERNEL_VERSION}.tar.sign

ENV WIREGUARD_VERSION=0.0.20170810
ENV WIREGUARD_SHA256=ab96230390625aad6f4816fa23aef6e9f7fee130f083d838919129ff12089bf7
ENV WIREGUARD_URL=https://git.zx2c4.com/WireGuard/snapshot/WireGuard-${WIREGUARD_VERSION}.tar.xz

# PGP keys: 589DA6B1 (greg@kroah.com) & 6092693E (autosigner@kernel.org) & 00411886 (torvalds@linux-foundation.org)
COPY keys.asc keys.asc

# Download and verify kernel
COPY sources/ /
RUN curl -fsSLO ${KERNEL_SHA256_SUMS} && \
    gpg2 -q --import keys.asc && \
    gpg2 --verify sha256sums.asc && \
    KERNEL_SHA256=$(grep linux-${KERNEL_VERSION}.tar.xz sha256sums.asc | cut -d ' ' -f 1) && \
    [ -f linux-${KERNEL_VERSION}.tar.xz ] || curl -fsSLO ${KERNEL_SOURCE} && \
    echo "${KERNEL_SHA256}  linux-${KERNEL_VERSION}.tar.xz" | sha256sum -c - && \
    xz -d linux-${KERNEL_VERSION}.tar.xz && \
    curl -fsSLO ${KERNEL_PGP2_SIGN} && \
    gpg2 --verify linux-${KERNEL_VERSION}.tar.sign linux-${KERNEL_VERSION}.tar && \
    cat linux-${KERNEL_VERSION}.tar | tar --absolute-names -x && mv /linux-${KERNEL_VERSION} /linux

# Apply local patches
COPY patches-${KERNEL_SERIES} /patches
WORKDIR /linux
RUN set -e && for patch in /patches/*.patch; do \
        echo "Applying $patch"; \
        patch -p1 < "$patch"; \
    done

# Kernel config
COPY kernel_config* /linux/

RUN case $(uname -m) in \
    x86_64) \
        KERNEL_DEF_CONF=/linux/arch/x86/configs/x86_64_defconfig; \
        ;; \
    aarch64) \
        KERNEL_DEF_CONF=/linux/arch/arm64/configs/defconfig; \
        ;; \
    esac  && \
    cp /linux/kernel_config-${KERNEL_SERIES}-$(uname -m) ${KERNEL_DEF_CONF}; \
    if [ -n "${EXTRA}" ]; then \
        sed -i "s/CONFIG_LOCALVERSION=\"-linuxkit\"/CONFIG_LOCALVERSION=\"-linuxkit${EXTRA}\"/" ${KERNEL_DEF_CONF}; \
        if [ "${EXTRA}" = "-dbg" ]; then \
            sed -i 's/CONFIG_PANIC_ON_OOPS=y/# CONFIG_PANIC_ON_OOPS is not set/' ${KERNEL_DEF_CONF}; \
        fi && \
        cat /linux/kernel_config${EXTRA} >> ${KERNEL_DEF_CONF}; \
    fi && \
    rm /linux/kernel_config* && \
    make defconfig && \
    make oldconfig && \
    if [ -z "${EXTRA}" ]; then diff .config ${KERNEL_DEF_CONF}; fi

RUN mkdir /out

# Kernel
RUN make -j "$(getconf _NPROCESSORS_ONLN)" KCFLAGS="-fno-pie" && \
    case $(uname -m) in \
    x86_64) \
        cp arch/x86_64/boot/bzImage /out/kernel; \
        ;; \
    aarch64) \
        cp arch/arm64/boot/Image.gz /out/kernel; \
        ;; \
    esac && \
    cp System.map /out && \
    ([ "${EXTRA}" = "-dbg" ] && cp vmlinux /out || true)

# WireGuard
RUN curl -sSL -o /wireguard.tar.xz "${WIREGUARD_URL}" && \
    echo "${WIREGUARD_SHA256}  /wireguard.tar.xz" | sha256sum -c - && \
    tar -C / --one-top-level=wireguard --strip-components=2 -xJf /wireguard.tar.xz "WireGuard-${WIREGUARD_VERSION}/src" && \
    make -j "$(getconf _NPROCESSORS_ONLN)" M="/wireguard" modules

# Modules
RUN make INSTALL_MOD_PATH=/tmp/kernel-modules modules_install && \
    make INSTALL_MOD_PATH=/tmp/kernel-modules M="/wireguard" modules_install && \
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

# perf (Don't compile for 4.4.x, it's broken and tedious to fix)
RUN if [ "${KERNEL_SERIES}" != "4.4.x" ]; then \
       mkdir -p /build/perf && \
       make -C tools/perf LDFLAGS=-static O=/build/perf && \
       strip /build/perf/perf && \
       cp /build/perf/perf /out; \
     fi

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=kernel-build /out/* /
