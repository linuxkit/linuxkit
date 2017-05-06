FROM mor1/c-compile:d8aa4bec68dcd4e6a48762ed9de09ee8051ed2d3 AS toybox-build
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
