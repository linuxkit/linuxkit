FROM linuxkit/alpine:c5021a113803f7608e3908014d316c3490183d0b@sha256:4c7a80173c71ca5019dc56f40442b3a8345f141dd46593bd1fe6d130294d07cb AS mirror

FROM alpine:3.5@sha256:dfbd4a3a8ebca874ebd2474f044a0b33600d4523d03b0df76e5c5986cb02d7e8
COPY --from=mirror /etc/apk/repositories /etc/apk/repositories
COPY --from=mirror /etc/apk/keys /etc/apk/keys/
COPY --from=mirror /mirror /mirror/

# https://github.com/docker/docker/blob/master/project/PACKAGERS.md#runtime-dependencies
# removed openssl as I do not think server needs it
RUN apk add --no-cache \
	btrfs-progs \
	ca-certificates \
	curl \
	e2fsprogs \
	e2fsprogs-extra \
	iptables \
	xfsprogs \
	xz
RUN rm -rf /mirror /etc/apk/repositories /etc/apk/keys

# set up Docker group
# set up subuid/subgid so that "--userns-remap=default" works out-of-the-box
RUN set -x \
	&& addgroup -S docker \
	&& addgroup -S dockremap \
	&& adduser -S -G dockremap dockremap \
	&& echo 'dockremap:165536:65536' >> /etc/subuid \
	&& echo 'dockremap:165536:65536' >> /etc/subgid

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
