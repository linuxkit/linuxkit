FROM mobylinux/alpine-build-c:36aecb5cf4738737634140eec9abebe1f6559a39

ARG KERNEL_VERSION=4.8.8

ENV KERNEL_SOURCE=https://www.kernel.org/pub/linux/kernel/v4.x/linux-${KERNEL_VERSION}.tar.xz

RUN curl -fsSL -o linux-${KERNEL_VERSION}.tar.xz ${KERNEL_SOURCE}

RUN cat linux-${KERNEL_VERSION}.tar.xz | tar --absolute-names -xJ &&  mv /linux-${KERNEL_VERSION} /linux

# this is aufs4.8 20161010
ENV AUFS_REPO       https://github.com/sfjro/aufs4-standalone
ENV AUFS_BRANCH     aufs4.8
ENV AUFS_COMMIT     e9fd128dcb16167417683e199a5feb14f3c9eca8

# Download AUFS
RUN git clone -b "$AUFS_BRANCH" "$AUFS_REPO" /aufs && \
    cd /aufs && \
    git checkout -q "$AUFS_COMMIT"

# aufs-util 20151116
ENV AUFS_TOOLS_REPO https://github.com/ncopa/aufs-util.git
ENV AUFS_TOOLS_COMMIT 3b7c5e262b53598a8204a915e485489c46d4e7a4

# Download aufs tools
RUN git clone ${AUFS_TOOLS_REPO} && \
    cd /aufs-util && \
    git checkout "$AUFS_TOOLS_COMMIT" 

#BUILD
# patch kernel with aufs
RUN cd /linux && \
    cp -r /aufs/Documentation /linux && \
    cp -r /aufs/fs /linux && \
    cp -r /aufs/include/uapi/linux/aufs_type.h /linux/include/uapi/linux/ && \
    set -e && for patch in \
        /aufs/aufs*-kbuild.patch \
        /aufs/aufs*-base.patch \
        /aufs/aufs*-mmap.patch \
        /aufs/aufs*-standalone.patch \
        /aufs/aufs*-loopback.patch \
        /aufs/lockdep-debug.patch \
    ; do \
        patch -p1 < "$patch"; \
    done

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

RUN cd /linux && \
    make defconfig && \
    make oldconfig && \
    make -j "$(getconf _NPROCESSORS_ONLN)" KCFLAGS="-fno-pie" && \
    make INSTALL_MOD_PATH=/tmp/kernel-modules modules_install && \
    ( cd /tmp/kernel-modules && tar cf /kernel-modules.tar . ) && \
    make INSTALL_HDR_PATH=/tmp/kernel-headers headers_install

# Build aufs tools, do this here as they need kernel headers and to match aufs
# Fortunately they are built statically linked
RUN cd /aufs-util && \
    CPPFLAGS="-I/tmp/kernel-headers/include" CFLAGS=$CPPFLAGS LDFLAGS=$CPPFLAGS make && \
    DESTDIR=/tmp/aufs-utils make install && \
    rm -rf /tmp/aufs-utils/usr/lib /tmp/aufs-utils/usr/share && \
    cd /tmp/aufs-utils && rm libau* && tar cf /aufs-utils.tar .

RUN printf "KERNEL_SOURCE=${KERNEL_SOURCE}\nAUFS_REPO=${AUFS_REPO}\nAUFS_BRANCH=${AUFS_BRANCH}\nAUFS_COMMIT=${AUFS_COMMIT}\nAUFS_TOOLS_REPO=${AUFS_TOOLS_REPO}\nAUFS_TOOLS_COMMIT=${AUFS_TOOLS_COMMIT}\n" > /kernel-source-info
