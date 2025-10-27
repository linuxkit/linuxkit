FROM --platform=linux/amd64 linuxkit/grub-dev:b006e88dd60d9f5863b93ca869801f3148788fa6 AS grub-build-amd64
FROM --platform=linux/arm64 linuxkit/grub-dev:b006e88dd60d9f5863b93ca869801f3148788fa6 AS grub-build-arm64
FROM --platform=linux/riscv64 linuxkit/grub-dev:b006e88dd60d9f5863b93ca869801f3148788fa6 AS grub-build-riscv64

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=grub-build-amd64 /*.EFI /
COPY --from=grub-build-arm64 /*.EFI /
COPY --from=grub-build-riscv64 /*.EFI /
