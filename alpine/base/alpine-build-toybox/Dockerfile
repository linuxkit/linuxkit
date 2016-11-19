FROM mobylinux/alpine-build-c:36aecb5cf4738737634140eec9abebe1f6559a39
COPY . .
# 0.7.2
ENV LDFLAGS=--static
RUN git clone https://github.com/landley/toybox.git && \
  cd toybox && git checkout b27d5d9ad0c56014d8661d91f69ee498bbbe4cf9 && \
  make defconfig
WORKDIR /toybox
ENTRYPOINT ["/build.sh"]
