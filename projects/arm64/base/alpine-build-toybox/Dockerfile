FROM aarch64/alpine:3.5
RUN apk --no-cache add --update --upgrade --purge   \
  automake                                          \
  bash                                              \
  build-base                                        \
  git
COPY . .
# 0.7.2
ENV LDFLAGS=--static
RUN git clone https://github.com/landley/toybox.git && \
  cd toybox && git checkout b27d5d9ad0c56014d8661d91f69ee498bbbe4cf9 && \
  make defconfig
WORKDIR /toybox
ENTRYPOINT ["/build.sh"]
