FROM linuxkit/alpine:33063834cf72d563cd8703467836aaa2f2b5a300 AS build
RUN apk add --no-cache git

# Make sure you also update the FW_COMMIT in ../firmware/Dockerfile
ENV FW_URL=git://git.kernel.org/pub/scm/linux/kernel/git/firmware/linux-firmware.git
ENV FW_COMMIT=edf390c23a4e185ff36daded36575f669f5059f7

RUN git clone ${FW_URL} && \
    cd /linux-firmware && \
    git checkout ${FW_COMMIT}

RUN mkdir -p /out/lib/firmware && \
    cd /linux-firmware && \
    ./copy-firmware.sh /out/lib/firmware

FROM scratch
WORKDIR /
ENTRYPOINT []
COPY --from=build /out/lib/ /lib/
    
