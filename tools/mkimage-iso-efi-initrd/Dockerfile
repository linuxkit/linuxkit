FROM linuxkit/grub:b498542e496dc2301ac1718d9096c2b26f041635 AS grub

FROM linuxkit/alpine:8c02670fd2e5c28d03fd39db0bddc1379b3af6b2 AS mirror
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
