FROM linuxkit/alpine:c89ded458506dc402c349a18b757bc5634d6010a as build
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

# u-boot compile. The patch is needed to handle larger kernels
ENV UBOOT_COMMIT=v2019.04
COPY u-boot.patch .
RUN git clone -b $UBOOT_COMMIT --depth 1 https://github.com/u-boot/u-boot.git
WORKDIR /u-boot
RUN patch -p 1 < /u-boot.patch && \
    make rpi_3_defconfig all && \
    mkdir -p /out/boot && \
    cp u-boot.bin /out/boot && \
    mkdir -p /out/bin && \
    cp tools/mkimage /out/bin

# fetch the Raspberry Pi 3 firmware (latest master)
ENV RPI_COMMIT=e1900836948f6c6bdf4571da1b966a9085c95d37
RUN mkdir -p /out/boot && \
    cd /out/boot && \
    curl -fsSLO https://github.com/raspberrypi/firmware/raw/$RPI_COMMIT/boot/LICENCE.broadcom && \
    curl -fsSLO https://github.com/raspberrypi/firmware/raw/$RPI_COMMIT/boot/bootcode.bin && \
    curl -fsSLO https://github.com/raspberrypi/firmware/raw/$RPI_COMMIT/boot/fixup_cd.dat && \
    curl -fsSLO https://github.com/raspberrypi/firmware/raw/$RPI_COMMIT/boot/fixup.dat && \
    curl -fsSLO https://github.com/raspberrypi/firmware/raw/$RPI_COMMIT/boot/fixup_x.dat && \
    curl -fsSLO https://github.com/raspberrypi/firmware/raw/$RPI_COMMIT/boot/start_cd.elf && \
    curl -fsSLO https://github.com/raspberrypi/firmware/raw/$RPI_COMMIT/boot/start.elf && \
    curl -fsSLO https://github.com/raspberrypi/firmware/raw/$RPI_COMMIT/boot/start_x.elf

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build /out/ /
COPY config.txt boot.script /boot/
COPY make-rpi3 /
ENTRYPOINT [ "/make-rpi3" ]
