# this is really hard to build. Do not change this version unless you must
FROM linuxkit/alpine:86cd4f51b49fb9a078b50201d892a3c7973d48ec AS grub-build

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
ENV GRUB_COMMIT=2f868ac992be2ae3ab838951aa3e260c045f20f9

COPY patches/* /patches/

WORKDIR /src
RUN git clone https://github.com/coreos/grub.git grub
WORKDIR /src/grub
RUN git checkout -b grub-build ${GRUB_COMMIT}
RUN  for patch in /patches/*.patch; do \
    echo "Applying $patch"; \
    patch -p1 < "$patch"; \
  done

RUN ./autogen.sh

RUN ./configure --libdir=/grub-lib --with-platform=efi CFLAGS="-Os -Wno-unused-value"
RUN make -j "$(getconf _NPROCESSORS_ONLN)"
RUN make install
RUN case $(uname -m) in \
  x86_64) \
    ./grub-mkimage -O x86_64-efi -d /grub-lib/grub/x86_64-efi -o /grub-lib/BOOTX64.EFI -p /EFI/BOOT ${GRUB_MODULES} linuxefi; \
    ;; \
  aarch64) \
    ./grub-mkimage -O arm64-efi -d /grub-lib/grub/arm64-efi -o /grub-lib/BOOTAA64.EFI -p /EFI/BOOT ${GRUB_MODULES}; \
    ;; \
  esac

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=grub-build /grub-lib/*.EFI /
