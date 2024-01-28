FROM linuxkit/grub:292deb743d85eb79cbd5b163841db8ccd2500677 AS grub

FROM linuxkit/alpine:7aea0bb7faf70f35c6b7d2d244c91da1d4b3ea93 AS mirror
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
ENTRYPOINT [ "/make-efi-initrd" ]
