FROM linuxkit/alpine:c5021a113803f7608e3908014d316c3490183d0b@sha256:4c7a80173c71ca5019dc56f40442b3a8345f141dd46593bd1fe6d130294d07cb AS build
RUN apk add \
    bash \
    gcc \
    git \
    make \
    musl-dev

ENV FILES="ls tar sh find sha1sum"
# 0.7.2
ENV LDFLAGS=--static
RUN git clone https://github.com/landley/toybox.git && \
  cd toybox && git checkout b27d5d9ad0c56014d8661d91f69ee498bbbe4cf9
WORKDIR /toybox
RUN make defconfig
RUN make ${FILES}
RUN mkdir -p /out/bin
RUN cp ${FILES} /out/bin

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build out/bin bin/
