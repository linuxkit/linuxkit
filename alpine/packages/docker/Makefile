all: docker

DOCKER_VERSION=1.10.1
ARCH?=x86_64
OS?=Linux

DOCKER_HOST=$(shell if echo "$(DOCKER_VERSION)" | grep -q -- '-rc'; then echo "test.docker.com"; else echo "get.docker.com"; fi)
docker:
	[ "${ARCH}" != "arm" ] && \
		curl -L -o docker https://${DOCKER_HOST}/builds/${OS}/${ARCH}/docker-${DOCKER_VERSION} || \
		cp docker-arm docker
	chmod +x docker

clean:
	rm -f docker
