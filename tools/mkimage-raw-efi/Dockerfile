FROM linuxkit/systemd-boot:7277832a6e3fb790d022c1c37edc4ea88c873b01 AS systemd-boot

FROM linuxkit/alpine:7f3944798557de5518a56e3437d7ed982701f224 AS mirror
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
