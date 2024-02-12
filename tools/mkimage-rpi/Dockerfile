FROM linuxkit/alpine:bc528cf9d4065d2e09aa44ff76909b94cfe8d867 as build
RUN apk add \
    bc \
    bison \
    dtc \
    curl \
    flex \
    make \
    gcc \
    git \
    musl-dev \
    patch

# Create small rootfs
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    libarchive-tools \
    musl \
    tar
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

ENV UBOOT_COMMIT=v2020.04
RUN git clone -b $UBOOT_COMMIT --depth 1 https://github.com/u-boot/u-boot.git
WORKDIR /u-boot
# u-boot compile. The patch is needed to handle larger kernels
COPY increase_mem.patch .
RUN patch -p 1 < increase_mem.patch && \
    make rpi_4_defconfig all && \
    mkdir -p /out/boot && \
    cp u-boot.bin /out/boot/u-boot-rpi-4.bin && \
    cp tools/mkimage /out/bin && \
    rm u-boot.bin && \
    make rpi_3_defconfig all && \
    cp u-boot.bin /out/boot/u-boot-rpi-3.bin

ENV RPI_COMMIT=62fc8c01165a80021054a430182b504f7b877c2d
# fetch the Raspberry Pi firmware
RUN mkdir -p /out/boot && \
    cd /out/boot && \
    curl -fsSLO https://github.com/raspberrypi/firmware/raw/${RPI_COMMIT}/boot/bootcode.bin && \
    curl -fsSLO https://github.com/raspberrypi/firmware/raw/${RPI_COMMIT}/boot/fixup.dat && \
    curl -fsSLO https://github.com/raspberrypi/firmware/raw/${RPI_COMMIT}/boot/start.elf && \
    curl -fsSLO https://github.com/raspberrypi/firmware/raw/${RPI_COMMIT}/boot/start4.elf && \
    curl -fsSLO https://github.com/raspberrypi/firmware/raw/${RPI_COMMIT}/boot/fixup4.dat

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build /out/ /
COPY cmdline.txt config.txt boot.script /boot/
COPY make-rpi /
ENTRYPOINT [ "/make-rpi" ]
