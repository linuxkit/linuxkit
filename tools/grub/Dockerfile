FROM --platform=linux/amd64 linuxkit/grub-dev:f59274029edcceac4e45d19eef495164c2c63216 AS grub-build-amd64
FROM --platform=linux/arm64 linuxkit/grub-dev:f59274029edcceac4e45d19eef495164c2c63216 AS grub-build-arm64
FROM --platform=linux/riscv64 linuxkit/grub-dev:f59274029edcceac4e45d19eef495164c2c63216 AS grub-build-riscv64

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=grub-build-amd64 /*.EFI /
COPY --from=grub-build-arm64 /*.EFI /
COPY --from=grub-build-riscv64 /*.EFI /
