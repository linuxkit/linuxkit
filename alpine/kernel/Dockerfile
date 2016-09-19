FROM mobylinux/debian-build-kernel:a009a11bf0ad

ARG KERNEL_VERSION=4.4.21

ENV KERNEL_SOURCE=https://www.kernel.org/pub/linux/kernel/v4.x/linux-${KERNEL_VERSION}.tar.xz

RUN curl -sSL -o linux-${KERNEL_VERSION}.tar.xz ${KERNEL_SOURCE}

RUN cat linux-${KERNEL_VERSION}.tar.xz | tar --absolute-names -xJ &&  mv /linux-${KERNEL_VERSION} /linux

# this is aufs4.4 20160912
ENV AUFS_REPO       https://github.com/sfjro/aufs4-standalone
ENV AUFS_BRANCH     aufs4.4
ENV AUFS_COMMIT     7d174ae40b4c9c876ee51aa50fa4ee1f3747de23

# Download AUFS
RUN git clone -b "$AUFS_BRANCH" "$AUFS_REPO" /aufs && \
    cd /aufs && \
    git checkout -q "$AUFS_COMMIT"

# aufs-util 20151116
ENV AUFS_TOOLS_REPO https://github.com/Distrotech/aufs-util.git
ENV AUFS_TOOLS_COMMIT b5e7e204036da5e815ddec15a847a03c232771b4

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
    cat /linux/debug_config >> /linux/arch/x86/configs/x86_64_defconfig; \
    fi

# Apply local patches
COPY patches /patches
RUN cd /linux && \
    set -e && for patch in /patches/*.patch; do \
        echo "Applying $patch"; \
        patch -p1 < "$patch"; \
    done

RUN jobs=$(nproc); \
    cd /linux && \
    make defconfig && \
    make oldconfig && \
    make -j ${jobs} && \
    make INSTALL_MOD_PATH=/tmp/kernel-modules modules_install && \
    ( cd /tmp/kernel-modules && tar cf /kernel-modules.tar . ) && \
    make INSTALL_HDR_PATH=/tmp/kernel-headers headers_install

# Build aufs tools, do this here as they need kernel headers and to match aufs
# Fortunately they are built statically linked
RUN cd /aufs-util && \
    CPPFLAGS="-I/tmp/kernel-headers/include" CFLAGS=$CPPFLAGS LDFLAGS=$CPPFLAGS make && \
    DESTDIR=/tmp/aufs-utils make install && \
    rm -rf /tmp/aufs-utils/usr/lib /tmp/aufs-utils/usr/share && \
    cd /tmp/aufs-utils && tar cf /aufs-utils.tar .

RUN printf "KERNEL_SOURCE=${KERNEL_SOURCE}\nAUFS_REPO=${AUFS_REPO}\nAUFS_BRANCH=${AUFS_BRANCH}\nAUFS_COMMIT=${AUFS_COMMIT}\nAUFS_TOOLS_REPO=${AUFS_TOOLS_REPO}\nAUFS_TOOLS_COMMIT=${AUFS_TOOLS_COMMIT}\n" > /kernel-source-info
