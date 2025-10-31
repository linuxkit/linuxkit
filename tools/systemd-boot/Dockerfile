FROM --platform=linux/amd64 linuxkit/alpine:7f3944798557de5518a56e3437d7ed982701f224 AS alpine-amd64
RUN apk add --no-cache systemd-boot systemd-efistub

FROM --platform=linux/arm64 linuxkit/alpine:7f3944798557de5518a56e3437d7ed982701f224 AS alpine-arm64
RUN apk add --no-cache systemd-boot systemd-efistub

FROM --platform=linux/riscv64 linuxkit/alpine:7f3944798557de5518a56e3437d7ed982701f224 AS alpine-riscv64
WORKDIR /work
ADD https://github.com/systemd/systemd.git#v258.1 .
RUN apk add --no-cache bash meson build-base coreutils gperf libcap-dev py3-jinja2 py3-elftools
# patch for musl libc
RUN find src/boot -type f -exec sed -i s/wchar_t/uint16_t/g {} \;
RUN meson setup --reconfigure -Defi=true -Dbootloader=enabled builddir && meson compile -C builddir systemd-boot

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine-amd64 /usr/lib/systemd/boot/efi/* /usr/lib/systemd/boot/efi/
COPY --from=alpine-arm64 /usr/lib/systemd/boot/efi/* /usr/lib/systemd/boot/efi/

COPY --from=alpine-riscv64 /work/builddir/src/boot/*.efi.stub /usr/lib/systemd/boot/efi/
COPY --from=alpine-riscv64 /work/builddir/src/boot/*.elf.stub /usr/lib/systemd/boot/efi/
COPY --from=alpine-riscv64 /work/builddir/src/boot/*.efi /usr/lib/systemd/boot/efi/

# this is just a non-platform specific python file so we only need it once for all architectures
COPY --from=alpine-riscv64 --chmod=755 /work/src/ukify/ukify.py /usr/sbin/ukify
