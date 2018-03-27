FROM linuxkit/alpine:96ad1eb5ec262b4cd0eef574cdc0b225b502d9ee AS build
RUN apk add --no-cache git

# Make sure you also update the FW_COMMIT in ../firmware/Dockerfile
ENV FW_URL=git://git.kernel.org/pub/scm/linux/kernel/git/firmware/linux-firmware.git
ENV FW_COMMIT=65b1c68c63f974d72610db38dfae49861117cae2

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
    
