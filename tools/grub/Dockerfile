FROM linuxkit/alpine:6ed32ba2b29a12b3d75f3d5b9be1b4ac00e7d479 AS grub-build
RUN apk add \
  automake \
  make \
  bison \
  coreutils \
  gettext \
  gettext-dev \
  flex \
  gcc \
  git \
  libtool \
  libc-dev \
  linux-headers \
  patch \
  pkgconfig \
  python3 \
  autoconf

# because python is not available
RUN ln -s python3 /usr/bin/python

ENV GRUB_MODULES="ext2 iso9660 gzio linux acpi normal cpio crypto disk boot crc64 part_gpt tftp xzio xfs video"
ENV GRUB_COMMIT=5bc41db756c5b342aa22fc95508bde105e05a095
ENV SRCDIR=/src

# separate steps for better caching
RUN mkdir /grub-lib ${SRCDIR} && \
  set -e && \
  cd ${SRCDIR} && \
  git clone git://git.savannah.gnu.org/grub.git && \
  cd grub && \
  git checkout -b grub-build ${GRUB_COMMIT}

WORKDIR ${SRCDIR}/grub

# this is long and slow, so keep separate step
RUN ./bootstrap

# run these in separate steps for caching. 
RUN ./configure --libdir=/grub-lib --with-platform=efi CFLAGS="-Os -Wno-unused-value" 

RUN make -j "$(getconf _NPROCESSORS_ONLN)" && \
  make install

RUN \
  arch=$(uname -m); \
  outarch="${arch}" ;\
  case "$arch" in \
  x86_64) \
    outarch=x86_64; \
    outfile=BOOTX64.EFI; \
    ;; \
  aarch64|arm64) \
    outarch=arm64; \
    outfile=BOOTAA64.EFI; \
    ;; \
  esac ; \
  grub-mkimage -O ${outarch}-efi -d /grub-lib/grub/${outarch}-efi -o /grub-lib/${outfile} -p /EFI/BOOT ${GRUB_MODULES} 

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=grub-build /grub-lib/*.EFI /
