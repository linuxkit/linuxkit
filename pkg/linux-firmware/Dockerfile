FROM linuxkit/alpine:28254e4530703db4caa6b0199a025c30a987dfa1 AS mirror

RUN apk add --no-cache git
ENV HEAD=a61ac5cf8374edbfe692d12f805a1b194f7fead2

RUN git clone -n git://git.kernel.org/pub/scm/linux/kernel/git/firmware/linux-firmware.git /var/linux-firmware && \
    cd /var/linux-firmware && \
    git checkout $HEAD

FROM scratch
COPY --from=mirror /var/linux-firmware/ /lib/firmware/
