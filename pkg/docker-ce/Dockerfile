FROM linuxkit/alpine:9bcf61f605ef0ce36cc94d59b8eac307862de6e1 AS mirror

# https://github.com/docker/docker/blob/master/project/PACKAGERS.md#runtime-dependencies
# removed openssl as I do not think server needs it
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
	alpine-baselayout \
	btrfs-progs \
	busybox \
	ca-certificates \
	curl \
	e2fsprogs \
	e2fsprogs-extra \
	iptables \
	musl \
	xfsprogs \
	xz
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
COPY --from=mirror /out/ /

# set up Docker group
# set up subuid/subgid so that "--userns-remap=default" works out-of-the-box
RUN set -x \
	&& addgroup -S docker \
	&& addgroup -S dockremap \
	&& adduser -S -G dockremap dockremap \
	&& echo 'dockremap:165536:65536' >> /etc/subuid \
	&& echo 'dockremap:165536:65536' >> /etc/subgid

# DOCKER_TYPE is stable, edge or test
ENV DOCKER_TYPE stable
ENV DOCKER_VERSION 17.06.0-ce
ENV DOCKER_SHA256 e582486c9db0f4229deba9f8517145f8af6c5fae7a1243e6b07876bd3e706620

# we could avoid installing client here I suppose
RUN set -x \
	&& curl -fSL "https://download.docker.com/linux/static/${DOCKER_TYPE}/$(uname -m)/docker-${DOCKER_VERSION}.tgz" -o docker.tgz \
	&& echo "${DOCKER_SHA256} *docker.tgz" | sha256sum -c - \
	&& tar -xzvf docker.tgz \
	&& mv docker/* /usr/bin/ \
	&& rmdir docker \
	&& rm docker.tgz \
	&& docker -v

# use the Docker copy of tini as our init for zombie reaping
ENTRYPOINT ["/usr/bin/docker-init", "/usr/bin/dockerd"]
