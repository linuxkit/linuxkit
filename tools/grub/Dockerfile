# this is really hard to build. Do not change this version unless you must
FROM --platform=linux/amd64 linuxkit/grub-dev:854df7920df18567e23940a1f9cf573980dcfe9b AS grub-build-amd64
FROM --platform=linux/arm64 linuxkit/grub-dev:854df7920df18567e23940a1f9cf573980dcfe9b AS grub-build-arm64

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=grub-build-amd64 /*.EFI /
COPY --from=grub-build-arm64 /*.EFI /
