FROM linuxkit/alpine:6090baae063eb5023c9601966e88df831f789a70 AS build
ENV GRUB_MODULES="part_gpt fat ext2 iso9660 gzio linux acpi normal cpio crypto disk boot crc64 \
search_fs_uuid tftp xzio xfs video efi_gop"

RUN apk add \
  grub \
  grub-efi

FROM build as build-arm64
RUN grub-mkimage -O arm64-efi -o /BOOTAA64.EFI -p /EFI/BOOT ${GRUB_MODULES}

FROM build as build-amd64
RUN grub-mkimage -O x86_64-efi -o /BOOTX64.EFI -p /EFI/BOOT ${GRUB_MODULES}

FROM build as build-riscv64
RUN grub-mkimage -O riscv64-efi -o /BOOTRISCV64.EFI -p /EFI/BOOT ${GRUB_MODULES}

FROM build-${TARGETARCH} AS collate
ARG TARGETARCH
COPY /*.EFI /

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=collate /*.EFI /
