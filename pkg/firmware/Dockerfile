# Make modules from a recentish kernel available
FROM linuxkit/kernel:4.14.28 AS kernel

FROM linuxkit/alpine:3683c9a66cd4da40bd7d6c7da599b2dcd738b559 AS build
RUN apk add --no-cache git kmod

# Clone the firmware repository
# Make sure you also update the FW_COMMIT in ../firmware-all/Dockerfile
ENV FW_URL=git://git.kernel.org/pub/scm/linux/kernel/git/firmware/linux-firmware.git
ENV FW_COMMIT=d1147327232ec4616a66ab898df84f9700c816c1
WORKDIR /
RUN git clone ${FW_URL} && \
    cd /linux-firmware && \
    git checkout ${FW_COMMIT}

# Copy files we always need/want: Licenses, docs and AMD CPU microcode
WORKDIR /linux-firmware
RUN set -e && \
    mkdir -p /out/lib/firmware && \
    cp README WHENCE /out/lib/firmware && \
    cp GPL-? LICENSE.* LICENCE.* /out/lib/firmware && \
    case $(uname -m) in \
    x86_64) \
        cp -r amd-ucode /out/lib/firmware; \
        ;; \
    esac

# Extract kernel modules for
WORKDIR /
COPY --from=kernel /kernel.tar /kernel.tar
RUN tar xf /kernel.tar

# Copy files required by the modules
RUN set -e && \
    for fw in $(find /lib/modules -name \*.ko -exec modinfo --field=firmware {} \;); do \
        mkdir -p "/out/lib/firmware/$fw" && \
        cp "/linux-firmware/$fw" "/out/lib/firmware/$fw"; \
    done

FROM scratch
WORKDIR /
ENTRYPOINT []
COPY --from=build /out/lib/ /lib/

