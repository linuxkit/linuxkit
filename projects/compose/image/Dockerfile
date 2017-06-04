FROM docker/compose:1.13.0
# because compose requires all sorts of dynamic libs, including glibc, it is much easier to
#   add docker client to compose than the reverse

ENV DOCKER_BUCKET get.docker.com
ENV DOCKER_VERSION 17.05.0-ce
ENV DOCKER_SHA256 340e0b5a009ba70e1b644136b94d13824db0aeb52e09071410f35a95d94316d9

# we need docker compose and docker load
# also need curl to test availability of docker API
RUN apk add --update curl

# we only need the client
RUN set -x \
	&& curl -fSL "https://${DOCKER_BUCKET}/builds/Linux/x86_64/docker-${DOCKER_VERSION}.tgz" -o docker.tgz \
	&& echo "${DOCKER_SHA256} *docker.tgz" | sha256sum -c - \
	&& tar -xzvf docker.tgz \
	&& mv docker/docker /usr/bin/ \
	&& rm -rf docker docker.tgz \
	&& docker -v


RUN mkdir -p /compose /app
WORKDIR /app
COPY . /app
ENTRYPOINT ["/app/waitfordocker.sh"]
CMD ["/app/load-images-and-compose.sh"]
