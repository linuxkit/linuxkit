FROM linuxkit/c-compile:f52f485825c890d581e82a62af6906c1d33d8e5d@sha256:473fd283a090d3546e93915adacf00c69a23ff615c95c968c4a40e8723985853 AS toybox-build
COPY . .
ENV FILES="ls tar sh find sha1sum"
# 0.7.2
ENV LDFLAGS=--static
RUN git clone https://github.com/landley/toybox.git && \
  cd toybox && git checkout b27d5d9ad0c56014d8661d91f69ee498bbbe4cf9 && \
  make defconfig
WORKDIR /toybox
RUN make ${FILES}
RUN mkdir -p /out/bin
RUN cp ${FILES} /out/bin

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=toybox-build out/bin bin/
