all: docker

DOCKER_VERSION=1.11.0-rc3
ARCH?=x86_64
OS?=Linux

DOCKER_HOST=$(shell if echo "$(DOCKER_VERSION)" | grep -q -- '-rc'; then echo "test.docker.com"; else echo "get.docker.com"; fi)
docker:
	mkdir -p bin
	if [ "${ARCH}" != "arm" ] ; then curl -f -L -o docker.tgz https://${DOCKER_HOST}/builds/${OS}/${ARCH}/docker-${DOCKER_VERSION}.tgz && tar xzf docker.tgz && mv docker/* bin && rm -rf docker/ docker.tgz; else cp docker-arm bin/docker; fi
	chmod +x bin/*

clean:
	rm -rf bin
