FROM --platform=linux/amd64 linuxkit/grub-dev:16ec86ade56d079c356fa72f0f6ce73aced0a322 AS grub-build-amd64
FROM --platform=linux/arm64 linuxkit/grub-dev:16ec86ade56d079c356fa72f0f6ce73aced0a322 AS grub-build-arm64
FROM --platform=linux/riscv64 linuxkit/grub-dev:16ec86ade56d079c356fa72f0f6ce73aced0a322 AS grub-build-riscv64

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=grub-build-amd64 /*.EFI /
COPY --from=grub-build-arm64 /*.EFI /
COPY --from=grub-build-riscv64 /*.EFI /
