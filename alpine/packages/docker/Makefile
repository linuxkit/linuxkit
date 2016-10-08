DOCKER_VERSION?=1.12.2-rc3
ARCH?=x86_64
OS?=Linux
DOCKER_EXPERIMENTAL?=1

all: usr/bin/docker

RELEASE_CANDIDATE=$(findstring -rc,$(DOCKER_VERSION))
ifeq ($(RELEASE_CANDIDATE),-rc)
TEST_HOST=test.docker.com
else
TEST_HOST=get.docker.com
endif

ifeq ($(DOCKER_EXPERIMENTAL),1)
DOCKER_HOST=experimental.docker.com
DOCKER_IMAGE?=docker:$(DOCKER_VERSION)-experimental
else
DOCKER_HOST=$(TEST_HOST)
DOCKER_IMAGE?=docker:$(DOCKER_VERSION)
endif

.PHONY: download hub cleanusr

ifeq ($(DOCKER_BIN_URL)$(FORCE_CURL)$(RELEASE_CANDIDATE),)
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
	([ -e docker/docker-init ] && mv docker/docker-init usr/bin/) || true

hub: cleanusr
	docker run --rm $(DOCKER_IMAGE) tar cf - -C /usr/local/bin . | tar xf - -C usr/bin
	rm -f usr/bin/docker-entrypoint.sh

clean:
	rm -rf usr/ docker/

.DELETE_ON_ERROR:
