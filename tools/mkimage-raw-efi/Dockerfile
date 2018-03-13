FROM linuxkit/alpine:96ad1eb5ec262b4cd0eef574cdc0b225b502d9ee AS grub-build

RUN apk add \
  automake \
  make \
  bison \
  gettext \
  flex \
  gcc \
  git \
  libtool \
  libc-dev \
  linux-headers \
  python3 \

  autoconf

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

FROM linuxkit/alpine:96ad1eb5ec262b4cd0eef574cdc0b225b502d9ee AS make-img

RUN \
  apk update && apk upgrade && \
  apk add --no-cache \
  dosfstools \
  libarchive-tools \
  binutils \
  mtools \
  sfdisk \
  sgdisk \
  xfsprogs \
  && true

COPY . .
COPY --from=grub-build /grub-lib/BOOT*.EFI /usr/local/share/

ENTRYPOINT [ "/make-efi" ]
