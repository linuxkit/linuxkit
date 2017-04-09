FROM alpine:3.5

# Docker daemon only minimal Alpine install

# set up Docker group
# set up subuid/subgid so that "--userns-remap=default" works out-of-the-box
RUN set -x \
	&& addgroup -S docker \
	&& addgroup -S dockremap \
	&& adduser -S -G dockremap dockremap \
	&& echo 'dockremap:165536:65536' >> /etc/subuid \
	&& echo 'dockremap:165536:65536' >> /etc/subgid

# https://github.com/docker/docker/blob/master/project/PACKAGERS.md#runtime-dependencies
# sfdisk and jq used by disk mounting code at present
RUN apk add --no-cache \
	ca-certificates \
	curl \
	iptables \
	xz \
	sfdisk \
	jq

# removed xfsprogs e2fs btrfs as we do not support dm or btrfs yet
# removed openssl as I do not think server needs it

ENV DOCKER_BUCKET get.docker.com
ENV DOCKER_VERSION 17.04.0-ce
ENV DOCKER_SHA256 c52cff62c4368a978b52e3d03819054d87bcd00d15514934ce2e0e09b99dd100

# we could avoid installing client here I suppose
RUN set -x \
	&& curl -fSL "https://${DOCKER_BUCKET}/builds/Linux/x86_64/docker-${DOCKER_VERSION}.tgz" -o docker.tgz \
	&& echo "${DOCKER_SHA256} *docker.tgz" | sha256sum -c - \
	&& tar -xzvf docker.tgz \
	&& mv docker/* /usr/bin/ \
	&& rmdir docker \
	&& rm docker.tgz \
	&& docker -v

# use the Docker copy of tini as our init for zombie reaping
ENTRYPOINT ["/usr/bin/docker-init", "/usr/bin/dockerd"]
