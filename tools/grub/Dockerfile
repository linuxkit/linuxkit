# this is really hard to build. Do not change this version unless you must
FROM --platform=linux/amd64 linuxkit/grub-dev:e94da02aac3a39fec34047051a40d367e61cd0a0 AS grub-build-amd64
FROM --platform=linux/arm64 linuxkit/grub-dev:e94da02aac3a39fec34047051a40d367e61cd0a0 AS grub-build-arm64

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=grub-build-amd64 /*.EFI /
COPY --from=grub-build-arm64 /*.EFI /
