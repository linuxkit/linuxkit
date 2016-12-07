# Tag: 36aecb5cf4738737634140eec9abebe1f6559a39
FROM mobylinux/alpine-build-c@sha256:d66b9625abc831f28f8c584991a9cb6975e85d3bb3d3768474b592f1cf32a3a6
COPY . .
# 0.7.2
ENV LDFLAGS=--static
RUN git clone https://github.com/landley/toybox.git && \
  cd toybox && git checkout b27d5d9ad0c56014d8661d91f69ee498bbbe4cf9 && \
  make defconfig
WORKDIR /toybox
ENTRYPOINT ["/build.sh"]
