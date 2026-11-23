FROM linuxkit/systemd-boot:b61b97512901589bc8e86301e243dc5430d11753 AS systemd-boot

FROM linuxkit/alpine:d497d5be5b78bfd78268f3466cc4080aee5d7389 AS mirror
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
  py3-pefile \
  && true
RUN mv /out/etc/apk/repositories.upstream /out/etc/apk/repositories

FROM scratch
WORKDIR /
COPY --from=mirror /out/ /
COPY --from=systemd-boot . .
COPY . .
ENTRYPOINT [ "/make-efi" ]
