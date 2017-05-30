FROM linuxkit/alpine:451603daf499e3a40308dbf5571dcffed2343ffa AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    bash \
    curl
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
WORKDIR /
COPY --from=mirror /out/ /

# Add docker
ENV DOCKER_BUCKET get.docker.com
ENV DOCKER_VERSION 17.05.0-ce
ENV DOCKER_SHA256 340e0b5a009ba70e1b644136b94d13824db0aeb52e09071410f35a95d94316d9

# Install just the client
RUN set -x \
        && curl -fSL "https://${DOCKER_BUCKET}/builds/Linux/x86_64/docker-${DOCKER_VERSION}.tgz" -o docker.tgz \
        && echo "${DOCKER_SHA256} *docker.tgz" | sha256sum -c - \
        && tar -xzvf docker.tgz \
        && mv docker/docker /usr/bin/ \
        && rm -rf docker \
        && rm docker.tgz \
        && docker -v

COPY bench_runner.sh ./bench_runner.sh

ENTRYPOINT ["/bin/sh", "/bench_runner.sh"]
