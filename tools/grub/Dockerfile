FROM --platform=linux/amd64 linuxkit/grub-dev:b3bc8c2d085b58efe1072693dd37ff9ad12d8b4a AS grub-build-amd64
FROM --platform=linux/arm64 linuxkit/grub-dev:b3bc8c2d085b58efe1072693dd37ff9ad12d8b4a AS grub-build-arm64
FROM --platform=linux/riscv64 linuxkit/grub-dev:b3bc8c2d085b58efe1072693dd37ff9ad12d8b4a AS grub-build-riscv64

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=grub-build-amd64 /*.EFI /
COPY --from=grub-build-arm64 /*.EFI /
COPY --from=grub-build-riscv64 /*.EFI /
