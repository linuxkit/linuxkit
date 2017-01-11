DOCKER_VERSION?=1.13.0-rc6
FORCE_CURL?=1
ARCH?=x86_64
OS?=Linux

all: usr/bin/docker

RELEASE_CANDIDATE=$(findstring -rc,$(DOCKER_VERSION))
ifeq ($(RELEASE_CANDIDATE),-rc)
DOCKER_HOST=test.docker.com
else
DOCKER_HOST=get.docker.com
endif

DOCKER_IMAGE?=docker:$(DOCKER_VERSION)

.PHONY: download hub cleanusr

ifeq ($(DOCKER_BIN_URL)$(FORCE_CURL)$(RELEASE_CANDIDATE),)
usr/bin/docker: Makefile hub
else
usr/bin/docker: Makefile download
endif

cleanusr:
	mkdir -p usr/bin
	rm -f usr/bin/* ok

download: cleanusr
ifdef DOCKER_BIN_URL
	(curl -fsSL ${DOCKER_BIN_URL} && touch ok) | tar xzf -
else
	(curl -fsSL https://${DOCKER_HOST}/builds/${OS}/${ARCH}/docker-${DOCKER_VERSION}.tgz && touch ok) | tar xzf -
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
	([ -d docker/completion ] && rm -rf docker/completion) || true

hub: cleanusr
	(DOCKER_CONTENT_TRUST=1 docker run --rm $(DOCKER_IMAGE) tar cf - -C /usr/local/bin . && touch ok) | tar xf - -C usr/bin
	rm ok
	rm -f usr/bin/docker-entrypoint.sh

clean:
	rm -rf usr/ docker/ ok

.DELETE_ON_ERROR:
