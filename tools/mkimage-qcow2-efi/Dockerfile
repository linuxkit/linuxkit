FROM linuxkit/alpine:3683c9a66cd4da40bd7d6c7da599b2dcd738b559 AS grub-build
RUN apk add \
  autoconf \
  automake \
  bison \
  gcc \
  gettext \
  git \
  flex \
  libc-dev \
  libtool \
  linux-headers \
  make \
  python3

# because python is not available
RUN ln -s python3 /usr/bin/python

ENV GRUB_MODULES="part_gpt fat ext2 iso9660 gzio linux acpi normal cpio crypto disk boot crc64 gpt \
search_disk_uuid tftp verify xzio xfs video"
ENV GRUB_COMMIT=d3fd939f18446b05d1d5456f23823498a1eb3fb4

RUN mkdir /grub-lib && \
  set -e && \
  git clone https://github.com/coreos/grub.git && \
  cd grub && \
  git checkout -b grub-build ${GRUB_COMMIT} && \
  ./autogen.sh && \
  ./configure --libdir=/grub-lib --with-platform=efi CFLAGS="-Os -Wno-unused-value" && \
  make -j "$(getconf _NPROCESSORS_ONLN)" && \
  make install && \
  case $(uname -m) in \
  x86_64) \
    ./grub-mkimage -O x86_64-efi -d /grub-lib/grub/x86_64-efi -o /grub-lib/BOOTX64.EFI -p /EFI/BOOT ${GRUB_MODULES} linuxefi; \
    ;; \
  aarch64) \
    ./grub-mkimage -O arm64-efi -d /grub-lib/grub/arm64-efi -o /grub-lib/BOOTAA64.EFI -p /EFI/BOOT ${GRUB_MODULES}; \
    ;; \
  esac

FROM linuxkit/alpine:3683c9a66cd4da40bd7d6c7da599b2dcd738b559 AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
  alpine-baselayout \
  binutils \
  busybox \
  dosfstools \
  libarchive-tools \
  mtools \
  qemu-img \
  sfdisk \
  sgdisk \
  xfsprogs \
  && true
RUN mv /out/etc/apk/repositories.upstream /out/etc/apk/repositories

FROM scratch
WORKDIR /
COPY --from=mirror /out/ /
COPY --from=grub-build /grub-lib/BOOT*.EFI /usr/local/share/
COPY . .
ENTRYPOINT [ "/make-efi" ]
