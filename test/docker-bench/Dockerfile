FROM alpine:3.5
RUN apk update && apk upgrade && apk add --no-cache bash curl
ADD . ./

# Also add docker
ENV DOCKER_BUCKET get.docker.com
ENV DOCKER_VERSION 17.04.0-ce
ENV DOCKER_SHA256 c52cff62c4368a978b52e3d03819054d87bcd00d15514934ce2e0e09b99dd100

# Downloads docker but only installs the client
RUN set -x \
	&& curl -fSL "https://${DOCKER_BUCKET}/builds/$(uname -s)/$(uname -m)/docker-${DOCKER_VERSION}.tgz" -o docker.tgz \
	&& echo "${DOCKER_SHA256} *docker.tgz" | sha256sum -c - \
	&& tar -xzvf docker.tgz \
	&& mv docker/docker /usr/bin/ \
	&& rm -rf docker \
	&& rm docker.tgz \
	&& docker -v

COPY . ./

ENTRYPOINT ["/bin/sh", "/bench_runner.sh"]
