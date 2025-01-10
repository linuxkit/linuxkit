FROM --platform=linux/amd64 linuxkit/grub-dev:7bcb9d21eb1d817a40fe2f1a84fdbf8655ad2f40 AS grub-build-amd64
FROM --platform=linux/arm64 linuxkit/grub-dev:7bcb9d21eb1d817a40fe2f1a84fdbf8655ad2f40 AS grub-build-arm64
FROM --platform=linux/riscv64 linuxkit/grub-dev:7bcb9d21eb1d817a40fe2f1a84fdbf8655ad2f40 AS grub-build-riscv64

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=grub-build-amd64 /*.EFI /
COPY --from=grub-build-arm64 /*.EFI /
COPY --from=grub-build-riscv64 /*.EFI /
