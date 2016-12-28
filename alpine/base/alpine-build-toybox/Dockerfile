# Tag: b77cfc4ad0033d4366df830ed697afc7bab458a2
FROM mobylinux/alpine-build-c@sha256:53739ea6042cb0ac39cf6e262012c1c4224206b2c9b719569fe7efa3a381348c
COPY . .
# 0.7.2
ENV LDFLAGS=--static
RUN git clone https://github.com/landley/toybox.git && \
  cd toybox && git checkout b27d5d9ad0c56014d8661d91f69ee498bbbe4cf9 && \
  make defconfig
WORKDIR /toybox
ENTRYPOINT ["/build.sh"]
