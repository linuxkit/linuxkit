FROM linuxkit/grub:c8864b3c16a130c808a90ae937c3aa726a907b85 AS grub

FROM linuxkit/alpine:e9313150d2cf07ad92b0adb86799dedd9a0476e4 AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
  alpine-baselayout \
  binutils \
  busybox \
  dosfstools \
  libarchive-tools \
  mtools \
  sfdisk \
  sgdisk \
  xfsprogs \
  && true
RUN mv /out/etc/apk/repositories.upstream /out/etc/apk/repositories

FROM scratch
WORKDIR /
COPY --from=mirror /out/ /
COPY --from=grub /BOOT*.EFI /usr/local/share/
COPY . .
ENTRYPOINT [ "/make-efi" ]
