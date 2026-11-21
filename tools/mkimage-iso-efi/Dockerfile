FROM linuxkit/grub:640b9fbabc4caf90c7552191611f4c42beda316a AS grub

FROM linuxkit/alpine:d497d5be5b78bfd78268f3466cc4080aee5d7389 AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
  alpine-baselayout \
  binutils \
  busybox \
  dosfstools \
  libarchive-tools \
  mtools \
  xorriso \
  && true
RUN mv /out/etc/apk/repositories.upstream /out/etc/apk/repositories

FROM scratch
WORKDIR /
COPY --from=mirror /out/ /
COPY --from=grub /BOOT*.EFI /usr/local/share/
COPY . .
ENTRYPOINT [ "/make-efi" ]
