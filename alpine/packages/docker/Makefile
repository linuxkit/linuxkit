DOCKER_VERSION?=1.12.5
FORCE_CURL?=1
ARCH?=x86_64
OS?=Linux
DOCKER_EXPERIMENTAL?=0

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
	rm -f usr/bin/* ok

download: cleanusr
ifdef DOCKER_BIN_URL
	(curl -sSL ${DOCKER_BIN_URL} && touch ok) | tar xzf -
else
	(curl -sSL https://${DOCKER_HOST}/builds/${OS}/${ARCH}/docker-${DOCKER_VERSION}.tgz && touch ok) | tar xzf -
endif
	rm ok
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
	(docker run --rm $(DOCKER_IMAGE) tar cf - -C /usr/local/bin . && touch ok) | tar xf - -C usr/bin
	rm ok
	rm -f usr/bin/docker-entrypoint.sh

clean:
	rm -rf usr/ docker/ ok

.DELETE_ON_ERROR:
