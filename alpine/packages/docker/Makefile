all: docker

DOCKER_VERSION=1.9.1
ARCH?=x86_64
OS?=Linux

docker:
	[ "${ARCH}" != "arm" ] && curl -L -o docker https://get.docker.com/builds/${OS}/${ARCH}/docker-${DOCKER_VERSION} || \
		cp docker-arm docker
	chmod +x docker

clean:
	rm -f docker
