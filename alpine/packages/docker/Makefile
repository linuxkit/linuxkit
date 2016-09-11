DOCKER_VERSION?=1.12.1
ARCH?=x86_64
OS?=Linux
DOCKER_EXPERIMENTAL?=1

DOCKER_BINARIES?=docker dockerd docker-proxy docker-runc docker-containerd docker-containerd-ctr docker-containerd-shim

all: usr/bin/docker

TEST_HOST=$(shell if echo "$(DOCKER_VERSION)" | grep -q -- '-rc'; then echo "test.docker.com"; else echo "get.docker.com"; fi)
ifeq ($(DOCKER_EXPERIMENTAL),1)
DOCKER_HOST=experimental.docker.com
DOCKER_IMAGE?=docker:$(DOCKER_VERSION)-experimental
else
DOCKER_HOST=$(TEST_HOST)
DOCKER_IMAGE?=docker:$(DOCKER_VERSION)
endif

.PHONY: download hub cleanusr

ifeq ($(DOCKER_BIN_URL)$(FORCE_CURL),)
usr/bin/docker: hub
else
usr/bin/docker: download
endif

cleanusr:
	mkdir -p usr/bin
	rm -f usr/bin/*

download: cleanusr
ifdef DOCKER_BIN_URL
	curl -sSL ${DOCKER_BIN_URL} | tar xzf -
else
	curl -sSL https://${DOCKER_HOST}/builds/${OS}/${ARCH}/docker-${DOCKER_VERSION}.tgz | tar xzf -
endif
	mv docker/docker-containerd-ctr \
	   docker/docker \
	   docker/docker-containerd \
	   docker/dockerd \
	   docker/docker-proxy \
	   docker/docker-runc \
	   docker/docker-containerd-shim \
	usr/bin/

hub: cleanusr
	docker run $(DOCKER_IMAGE) tar cf - -C /usr/local/bin $(DOCKER_BINARIES) | tar xf - -C usr/bin

clean:
	rm -rf usr/ docker/

.DELETE_ON_ERROR:
