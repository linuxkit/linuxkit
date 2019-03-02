FROM linuxkit/alpine:518c2ed0f398c5508969ac5e033607201fb419ed AS build
RUN apk add --no-cache git

# Make sure you also update the FW_COMMIT in ../firmware/Dockerfile
ENV FW_URL=git://git.kernel.org/pub/scm/linux/kernel/git/firmware/linux-firmware.git
ENV FW_COMMIT=9ee52be785cf91fc6a3c6aa27d484873f8270b72

RUN mkdir -p /out/lib && \
    cd /out/lib && \
    git clone ${FW_URL} firmware && \
    cd firmware && \
    git checkout ${FW_COMMIT} && \
    rm -rf .git 

FROM scratch
WORKDIR /
ENTRYPOINT []
COPY --from=build /out/lib/ /lib/
    
