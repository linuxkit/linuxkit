FROM linuxkit/alpine-build-kernel:cfdd576c36a52ed2dd62f237f79eeedc2dd3697b@sha256:3fe08db373a9373ba1616a485858f01ebd2d7a3cb364a099d0ed8b45fa419da2


ARG KERNEL_VERSION
ARG DEBUG=0

ENV KERNEL_SOURCE=https://www.kernel.org/pub/linux/kernel/v4.x/linux-${KERNEL_VERSION}.tar.xz

# Download kernel source code
RUN curl -fsSL -o linux-${KERNEL_VERSION}.tar.xz ${KERNEL_SOURCE}
RUN tar xf linux-${KERNEL_VERSION}.tar.xz && mv /linux-${KERNEL_VERSION} /linux
WORKDIR /linux

ENV DEF_CONFIG_FILE=/linux/arch/x86/configs/x86_64_defconfig
COPY kernel_config ${DEF_CONFIG_FILE}
COPY kernel_config.debug /linux/debug_config


# Enable debug
RUN if [ $DEBUG -ne "0" ]; then \
    sed -i 's/CONFIG_PANIC_ON_OOPS=y/# CONFIG_PANIC_ON_OOPS is not set/' \
								 ${DEF_CONFIG_FILE}; \
    cat /linux/debug_config >> ${DEF_CONFIG_FILE}; \
    fi


RUN cat ${DEF_CONFIG_FILE}

# Apply local patches
COPY patches-4.9 /patches
RUN cd /linux && \
    set -e && for patch in /patches/*.patch; do \
        echo "Applying $patch"; \
        patch -p1 < "$patch"; \
    done

# Build kernel
RUN make defconfig && \
    make oldconfig && \
    perl -p -i -e "s/^EXTRAVERSION.*/EXTRAVERSION = -linuxkit/" Makefile && \
    make -j "$(getconf _NPROCESSORS_ONLN)" KCFLAGS="-fno-pie"

#bzImage
#vmlinux
RUN cp vmlinux arch/x86_64/boot/bzImage /

# CC does not provide modules, not needed to distribute headers.
#kernel-headers.tar: provides kernel headers
RUN mkdir -p /tmp/kernel-headers/usr && \
	cd /tmp/kernel-headers && tar cf /kernel-headers.tar usr

# CC does no use modules do not ship it
#kernel-modules.tar: provides kernel modules
RUN mkdir -p /tmp/kernel-modules/lib/modules && \
	cd /tmp/kernel-modules && tar cf /kernel-modules.tar lib


WORKDIR /

#kernel-dev.tar: provides headers .config linux/include arch/x86/include
RUN  mkdir -p  /tmp/usr/src/linux-headers && \
	cd /tmp/ && tar cf /kernel-dev.tar usr/src

RUN printf "KERNEL_SOURCE=${KERNEL_SOURCE}\n" > /kernel-source-info
