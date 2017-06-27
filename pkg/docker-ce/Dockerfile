FROM linuxkit/alpine:c608d404c1cb36cce0c7d9303e30b52c9d81ccf0 AS mirror

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

FROM linuxkit/vpnkit-forwarder:e2776b82ddfe82ed7f90e55d7a2b424e62e9a279 AS vpnkit
FROM scratch
COPY --from=mirror /out/ /
COPY --from=vpnkit /vpnkit-expose-port /usr/bin/vpnkit-expose-port

# set up Docker group
# set up subuid/subgid so that "--userns-remap=default" works out-of-the-box
RUN set -x \
	&& addgroup -S docker \
	&& addgroup -S dockremap \
	&& adduser -S -G dockremap dockremap \
	&& echo 'dockremap:165536:65536' >> /etc/subuid \
	&& echo 'dockremap:165536:65536' >> /etc/subgid

ENV DOCKER_BUCKET get.docker.com
ENV DOCKER_VERSION 17.05.0-ce
ENV DOCKER_SHA256 340e0b5a009ba70e1b644136b94d13824db0aeb52e09071410f35a95d94316d9

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
