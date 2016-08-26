DOCKER_VERSION?=1.12.1
ARCH?=x86_64
OS?=Linux
DOCKER_EXPERIMENTAL?=1
INCLUDE_DOCKER_PROXY?=false

all: bin

TEST_HOST=$(shell if echo "$(DOCKER_VERSION)" | grep -q -- '-rc'; then echo "test.docker.com"; else echo "get.docker.com"; fi)
DOCKER_HOST?=$(shell [ "${DOCKER_EXPERIMENTAL}" -eq 1 ] && printf "experimental.docker.com" || printf "${TEST_HOST}")

# Note we copy docker-proxy in conditionally as Docker for Mac and Windows
# over-ride with their own version.

bin:
	mkdir -p bin
	curl -sSL -o docker.tgz https://${DOCKER_HOST}/builds/${OS}/${ARCH}/docker-${DOCKER_VERSION}.tgz
	tar xzf docker.tgz && \
	mv docker/docker-containerd-ctr \
	   docker/docker \
	   docker/docker-containerd \
	   docker/dockerd \
	   docker/docker-runc \
	   docker/docker-containerd-shim \
	bin/
ifeq ($(INCLUDE_DOCKER_PROXY), true)
	mv docker/docker-proxy bin/
endif
	rm -rf docker/ docker.tgz
	chmod +x bin/*

arm:
	mkdir -p bin
	rm -f bin/docker*
	cp docker-arm bin/docker

clean:
	rm -rf bin docker/ docker.tgz
