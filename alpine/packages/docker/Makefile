all: docker

DOCKER_VERSION=1.11.0-rc2
ARCH?=x86_64
OS?=Linux

DOCKER_HOST=$(shell if echo "$(DOCKER_VERSION)" | grep -q -- '-rc'; then echo "test.docker.com"; else echo "get.docker.com"; fi)
docker:
	if [ "${ARCH}" != "arm" ] ; then curl -f -L -o docker.tgz https://${DOCKER_HOST}/builds/${OS}/${ARCH}/docker-${DOCKER_VERSION}.tgz && tar xjf docker.tgz && mv usr/local/bin/* . && rm -rf usr/ docker.tgz; else cp docker-arm docker; fi
	chmod +x docker docker-containerd docker-containerd-ctr docker-containerd-shim docker-runc

clean:
	rm -f docker docker-containerd docker-containerd-ctr docker-containerd-shim docker-runc
