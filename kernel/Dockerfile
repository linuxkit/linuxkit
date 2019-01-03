FROM linuxkit/alpine:4768505d40f23e198011b6f2c796f985fe50ec39 AS kernel-build
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
    libressl \
    libressl-dev \
    linux-headers \
    mpc1-dev \
    mpfr-dev \
    ncurses-dev \
    patch \
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
ARG DEBUG

ENV KERNEL_SOURCE=https://www.kernel.org/pub/linux/kernel/v4.x/linux-${KERNEL_VERSION}.tar.xz
ENV KERNEL_SHA256_SUMS=https://www.kernel.org/pub/linux/kernel/v4.x/sha256sums.asc
ENV KERNEL_PGP2_SIGN=https://www.kernel.org/pub/linux/kernel/v4.x/linux-${KERNEL_VERSION}.tar.sign

ENV WIREGUARD_VERSION=0.0.20181218
ENV WIREGUARD_SHA256="2e9f86acefa49dbfb7fa6f5e10d543f1885a2d5460cd5e102696901107675735"
ENV WIREGUARD_URL=https://git.zx2c4.com/WireGuard/snapshot/WireGuard-${WIREGUARD_VERSION}.tar.xz

# We copy the entire directory. This copies some unneeded files, but
# allows us to check for the existence /patches-${KERNEL_SERIES} to
# build kernels without patches.
COPY / /

# Download and verify kernel
# PGP keys: 589DA6B1 (greg@kroah.com) & 6092693E (autosigner@kernel.org) & 00411886 (torvalds@linux-foundation.org)
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

WORKDIR /linux
# Apply local specific patches if present
RUN set -e && \
    if [ -n "${EXTRA}" ] && [ -d /patches-${KERNEL_SERIES}${EXTRA} ]; then \
	echo "Patching ${EXTRA} kernel"; \
	for patch in /patches-${KERNEL_SERIES}${EXTRA}/*.patch; do \
            echo "Applying $patch"; \
            patch -t -F0 -N -u -p1 < "$patch"; \
	done; \
    fi

# Apply local common patches if present
RUN set -e && \
    if [ -d /patches-${KERNEL_SERIES} ]; then \
        for patch in /patches-${KERNEL_SERIES}/*.patch; do \
            echo "Applying $patch"; \
            patch -t -F0 -N -u -p1 < "$patch"; \
        done; \
    fi

RUN mkdir -p /out/src

# Save kernel source
RUN tar cJf /out/src/linux.tar.xz /linux

# Kernel config
RUN case $(uname -m) in \
    x86_64) \
        KERNEL_DEF_CONF=/linux/arch/x86/configs/x86_64_defconfig; \
        ;; \
    aarch64) \
        KERNEL_DEF_CONF=/linux/arch/arm64/configs/defconfig; \
        ;; \
    s390x) \
        KERNEL_DEF_CONF=/linux/arch/s390/defconfig; \
        ;; \
    esac  && \
    cp /config-${KERNEL_SERIES}-$(uname -m) ${KERNEL_DEF_CONF}; \
    if [ -n "${EXTRA}" ] && [ -f "/config-${KERNEL_SERIES}-$(uname -m)${EXTRA}" ]; then \
        cat /config-${KERNEL_SERIES}-$(uname -m)${EXTRA} >> ${KERNEL_DEF_CONF}; \
    fi; \
    sed -i "s/CONFIG_LOCALVERSION=\"-linuxkit\"/CONFIG_LOCALVERSION=\"-linuxkit${EXTRA}${DEBUG}\"/" ${KERNEL_DEF_CONF}; \
    if [ -n "${DEBUG}" ]; then \
        sed -i 's/CONFIG_PANIC_ON_OOPS=y/# CONFIG_PANIC_ON_OOPS is not set/' ${KERNEL_DEF_CONF}; \
        cat /config${DEBUG} >> ${KERNEL_DEF_CONF}; \
    fi && \
    make defconfig && \
    make oldconfig && \
    if [ -z "${EXTRA}" ] && [ -z "${DEBUG}" ]; then diff -u .config ${KERNEL_DEF_CONF}; fi


# Kernel
RUN make -j "$(getconf _NPROCESSORS_ONLN)" KCFLAGS="-fno-pie" && \
    case $(uname -m) in \
    x86_64) \
        cp arch/x86_64/boot/bzImage /out/kernel; \
        ;; \
    aarch64) \
        cp arch/arm64/boot/Image.gz /out/kernel; \
        ;; \
    s390x) \
        cp arch/s390/boot/bzImage /out/kernel; \
        ;; \
    esac && \
    cp System.map /out && \
    ([ -n "${DEBUG}" ] && cp vmlinux /out || true)

# WireGuard
RUN curl -fsSL -o /wireguard.tar.xz "${WIREGUARD_URL}" && \
    echo "${WIREGUARD_SHA256}  /wireguard.tar.xz" | sha256sum -c - && \
    cp /wireguard.tar.xz /out/src/ && \
    tar -C / --one-top-level=wireguard --strip-components=2 -xJf /wireguard.tar.xz "WireGuard-${WIREGUARD_VERSION}/src" && \
    make -j "$(getconf _NPROCESSORS_ONLN)" M="/wireguard" modules

# Modules and Device Tree binaries
RUN make INSTALL_MOD_PATH=/tmp/kernel-modules modules_install && \
    make INSTALL_MOD_PATH=/tmp/kernel-modules M="/wireguard" modules_install && \
    ( DVER=$(basename $(find /tmp/kernel-modules/lib/modules/ -mindepth 1 -maxdepth 1)) && \
      cd /tmp/kernel-modules/lib/modules/$DVER && \
      rm build source && \
      ln -s /usr/src/linux-headers-$DVER build ) && \
    case $(uname -m) in \
    aarch64) \
        make INSTALL_DTBS_PATH=/tmp/kernel-modules/boot/dtb dtbs_install; \
        ;; \
    esac && \
    ( cd /tmp/kernel-modules && tar cf /out/kernel.tar . )

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
                      -name '*.lds' -o -name '*.pl' -o -name '*.sh' -o \
                      -name 'objtool' -o -name 'fixdep' -o -name 'randomize_layout_seed.h' \) | \
         tar cf - -T - | (cd $dir; tar xf -) && \
    ( cd /tmp && tar cf /out/kernel-dev.tar usr/src )

RUN printf "KERNEL_SOURCE=${KERNEL_SOURCE}\n" > /out/kernel-source-info

# perf
# Only build perf for the latest LTS and the latest stable kernel.
# Skip arm64 for now as perf for 4.19.x fails to build.
RUN if [ "${KERNEL_SERIES}" != 4.9.x ] && [ $(uname -m) != aarch64 ]; then \
       mkdir -p /build/perf && \
       make -C tools/perf LDFLAGS=-static O=/build/perf && \
       strip /build/perf/perf && \
       cp /build/perf/perf /out; \
     fi

# Download Intel ucode and create a CPIO archive for it
ENV UCODE_URL=https://downloadmirror.intel.com/28039/eng/microcode-20180807.tgz
RUN set -e && \
    if [ $(uname -m) == x86_64 ]; then \
        cd /ucode && \
        curl -fsSL -o microcode.tar.gz ${UCODE_URL} && \
        md5sum -c intel-ucode-md5sums && \
        tar xf microcode.tar.gz && \
        rm -f intel-ucode/list && \
        iucode_tool --normal-earlyfw --write-earlyfw=/out/intel-ucode.cpio ./intel-ucode && \
        cp intel-ucode-license.txt /out; \
    fi

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=kernel-build /out/* /
