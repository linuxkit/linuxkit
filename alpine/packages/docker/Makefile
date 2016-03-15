all: docker

DOCKER_VERSION=1.10.3
ARCH?=x86_64
OS?=Linux

DOCKER_HOST=$(shell if echo "$(DOCKER_VERSION)" | grep -q -- '-rc'; then echo "test.docker.com"; else echo "get.docker.com"; fi)
docker:
	if [ "${ARCH}" != "arm" ] ; then curl -f -L -o docker https://${DOCKER_HOST}/builds/${OS}/${ARCH}/docker-${DOCKER_VERSION} ; else cp docker-arm docker; fi
	chmod +x docker

clean:
	rm -f docker
