FROM --platform=linux/amd64 linuxkit/grub-dev:4184bd7644a0edf73d4fe8a55171fe06f4b4d738 AS grub-build-amd64
FROM --platform=linux/arm64 linuxkit/grub-dev:4184bd7644a0edf73d4fe8a55171fe06f4b4d738 AS grub-build-arm64
FROM --platform=linux/riscv64 linuxkit/grub-dev:4184bd7644a0edf73d4fe8a55171fe06f4b4d738 AS grub-build-riscv64

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=grub-build-amd64 /*.EFI /
COPY --from=grub-build-arm64 /*.EFI /
COPY --from=grub-build-riscv64 /*.EFI /
