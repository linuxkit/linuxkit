FROM --platform=linux/amd64 linuxkit/grub-dev:a61d5ca6f0c0da0983e4e9c90dfa17c35e725817 AS grub-build-amd64
FROM --platform=linux/arm64 linuxkit/grub-dev:a61d5ca6f0c0da0983e4e9c90dfa17c35e725817 AS grub-build-arm64
FROM --platform=linux/riscv64 linuxkit/grub-dev:a61d5ca6f0c0da0983e4e9c90dfa17c35e725817 AS grub-build-riscv64

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=grub-build-amd64 /*.EFI /
COPY --from=grub-build-arm64 /*.EFI /
COPY --from=grub-build-riscv64 /*.EFI /
