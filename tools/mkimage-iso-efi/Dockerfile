FROM linuxkit/alpine:07f7d136e427dc68154cd5edbb2b9576f9ac5213 AS grub-build

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
ENV GRUB_COMMIT=6782f6d431d22b4e9ab14e94d263795c7991e160

COPY patches/* /patches/
RUN mkdir /grub-lib && \
  set -e && \
  git clone https://github.com/coreos/grub.git && \
  cd grub && \
  git checkout -b grub-build ${GRUB_COMMIT} && \
  for patch in /patches/*.patch; do \
    echo "Applying $patch"; \
    patch -p1 < "$patch"; \
  done && \
  ./autogen.sh && \
  ./configure --libdir=/grub-lib --with-platform=efi CFLAGS="-Os -Wno-unused-value" && \
  make -j "$(getconf _NPROCESSORS_ONLN)" && \
  make install && \
# create the grub core image
  case $(uname -m) in \
  x86_64) \
    ./grub-mkimage -O x86_64-efi -d /grub-lib/grub/x86_64-efi -o /grub-lib/BOOTX64.EFI -p /EFI/BOOT ${GRUB_MODULES} linuxefi; \
    ;; \
  aarch64) \
    ./grub-mkimage -O arm64-efi -d /grub-lib/grub/arm64-efi -o /grub-lib/BOOTAA64.EFI -p /EFI/BOOT ${GRUB_MODULES}; \
    ;; \
  esac

FROM linuxkit/alpine:07f7d136e427dc68154cd5edbb2b9576f9ac5213 AS make-img

RUN \
  apk update && apk upgrade && \
  apk add --no-cache \
  dosfstools \
  libarchive-tools \
  binutils \
  mtools \
  xorriso \
  && true

COPY . .
COPY --from=grub-build /grub-lib/BOOT*.EFI /usr/local/share/

ENTRYPOINT [ "/make-efi" ]
