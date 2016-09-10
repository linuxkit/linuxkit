DOCKER_VERSION?=1.12.1
ARCH?=x86_64
OS?=Linux
DOCKER_EXPERIMENTAL?=1

all: usr/bin/docker

TEST_HOST=$(shell if echo "$(DOCKER_VERSION)" | grep -q -- '-rc'; then echo "test.docker.com"; else echo "get.docker.com"; fi)
DOCKER_HOST?=$(shell [ "${DOCKER_EXPERIMENTAL}" -eq 1 ] && printf "experimental.docker.com" || printf "${TEST_HOST}")
DOCKER_BIN_URL?="https://${DOCKER_HOST}/builds/${OS}/${ARCH}/docker-${DOCKER_VERSION}.tgz"

usr/bin/docker:
	mkdir -p usr/bin
	curl -sSL ${DOCKER_BIN_URL} | tar xzf - && \
	mv docker/docker-containerd-ctr \
	   docker/docker \
	   docker/docker-containerd \
	   docker/dockerd \
	   docker/docker-proxy \
	   docker/docker-runc \
	   docker/docker-containerd-shim \
	usr/bin/

clean:
	rm -rf usr/ docker/

.DELETE_ON_ERROR:
