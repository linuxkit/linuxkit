FROM linuxkit/alpine:3683c9a66cd4da40bd7d6c7da599b2dcd738b559 AS build
RUN apk add --no-cache git

# Make sure you also update the FW_COMMIT in ../firmware/Dockerfile
ENV FW_URL=git://git.kernel.org/pub/scm/linux/kernel/git/firmware/linux-firmware.git
ENV FW_COMMIT=d1147327232ec4616a66ab898df84f9700c816c1

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
    
