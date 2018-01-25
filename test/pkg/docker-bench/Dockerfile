FROM linuxkit/alpine:cba395fbc278daee841106801aba1e1bd7e0f2f7 AS mirror
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

# DOCKER_TYPE is stable, edge or test
ENV DOCKER_TYPE stable
ENV DOCKER_VERSION 17.06.0-ce
ENV DOCKER_SHA256 e582486c9db0f4229deba9f8517145f8af6c5fae7a1243e6b07876bd3e706620

# Install just the client
RUN set -x \
        && curl -fSL "https://download.docker.com/linux/static/${DOCKER_TYPE}/$(uname -m)/docker-${DOCKER_VERSION}.tgz" -o docker.tgz \
        && echo "${DOCKER_SHA256} *docker.tgz" | sha256sum -c - \
        && tar -xzvf docker.tgz \
        && mv docker/docker /usr/bin/ \
        && rm -rf docker \
        && rm docker.tgz \
        && docker -v

COPY bench_runner.sh ./bench_runner.sh

ENTRYPOINT ["/bin/sh", "/bench_runner.sh"]
