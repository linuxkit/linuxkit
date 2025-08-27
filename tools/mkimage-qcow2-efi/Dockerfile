FROM linuxkit/grub:5adb6fee45facb51f3f3b51af382fd21620d4f76 AS grub

FROM linuxkit/alpine:6090baae063eb5023c9601966e88df831f789a70 AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
  alpine-baselayout \
  binutils \
  busybox \
  dosfstools \
  libarchive-tools \
  mtools \
  qemu-img \
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
