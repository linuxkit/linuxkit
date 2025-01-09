FROM linuxkit/grub:b498542e496dc2301ac1718d9096c2b26f041635 AS grub

FROM linuxkit/alpine:5ac048802293564a50414394e1b9896eac528e7d AS mirror
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
