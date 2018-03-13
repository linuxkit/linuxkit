FROM linuxkit/alpine:96ad1eb5ec262b4cd0eef574cdc0b225b502d9ee as build
RUN apk add \
    bc \
    dtc \
    curl \
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
ENV UBOOT_COMMIT=v2017.09
COPY u-boot.patch .
RUN git clone https://github.com/u-boot/u-boot.git && \
    cd /u-boot && \
    git checkout $UBOOT_COMMIT
WORKDIR /u-boot
RUN patch -p 1 < /u-boot.patch && \
    make rpi_3_defconfig all && \
    mkdir -p /out/boot && \
    cp u-boot.bin /out/boot && \
    mkdir -p /out/bin && \
    cp tools/mkimage /out/bin

# fetch the Raspberry Pi 3 firmware (latest master)
ENV RPI_COMMIT=478d637c476e838ffcfa8535232ff0b86daf5918
RUN mkdir -p /out/boot && \
    cd /out/boot && \
    curl -fsSLO https://github.com/raspberrypi/firmware/raw/$RPI_COMMIT/boot/bootcode.bin && \
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
