BUNDLE?=
ifdef BUNDLE
DOCKER_VERSION?=${BUNDLE}
else
DOCKER_VERSION?=17.03.0-ce-rc1
endif
FORCE_CURL?=1
ARCH?=x86_64
OS?=Linux


AWS_CLI_IMAGE=mobylinux/aws-cli:9432b650794d38a70cf00be4da971367c52d1d5b@sha256:679f6f45fb8598cab90888733a07ddeeb26a27a7889114f89aaf712eaa3abe06

all: usr/bin/docker

RELEASE_CANDIDATE=$(findstring -rc,$(DOCKER_VERSION))
ifeq ($(RELEASE_CANDIDATE),-rc)
DOCKER_HOST=test.docker.com
else
DOCKER_HOST=get.docker.com
endif

DOCKER_IMAGE?=docker:$(DOCKER_VERSION)

ifeq ($(DOCKER_VERSION),master)
ifeq ($(OS),Linux)
OS=linux
endif
ifeq ($(ARCH),x86_64)
ARCH=amd64
endif
MASTER_VERSION:=$(shell curl -fsSL https://master.dockerproject.org/version)
DOCKER_BIN_URL=https://master.dockerproject.org/$(OS)/$(ARCH)/docker-$(MASTER_VERSION).tgz
endif

.PHONY: download hub cleanusr

ifeq ($(DOCKER_BIN_URL)$(FORCE_CURL)$(RELEASE_CANDIDATE),)
usr/bin/docker: Makefile hub
else
usr/bin/docker: Makefile download
endif

cleanusr:
	mkdir -p usr/bin
	rm -f usr/bin/* ok

S3_PREFIX=s3://

download: cleanusr
ifdef DOCKER_BIN_URL
ifeq ($(filter $(S3_PREFIX)%, $(DOCKER_BIN_URL)),)
	(curl -fsSL "${DOCKER_BIN_URL}" && touch ok) | tar xzf -
else
	(docker run --rm -e AWS_SECRET_ACCESS_KEY -e AWS_ACCESS_KEY_ID $(AWS_CLI_IMAGE) s3 cp '$(DOCKER_BIN_URL)' -) | tar xzf -
endif
else
	(curl -fsSL https://${DOCKER_HOST}/builds/${OS}/${ARCH}/docker-${DOCKER_VERSION}.tgz && touch ok) | tar xzf -
endif
	rm ok
ifdef BUNDLE
	mv bundles/${BUNDLE}/binary-daemon/docker-containerd-ctr \
		bundles/${BUNDLE}/binary-client/docker \
		bundles/${BUNDLE}/binary-daemon/docker-containerd \
		bundles/${BUNDLE}/binary-daemon/dockerd \
		bundles/${BUNDLE}/binary-daemon/docker-proxy \
		bundles/${BUNDLE}/binary-daemon/docker-runc \
		bundles/${BUNDLE}/binary-daemon/docker-containerd-shim \
	usr/bin/
else
	mv docker/docker-containerd-ctr \
	   docker/docker \
	   docker/docker-containerd \
	   docker/dockerd \
	   docker/docker-proxy \
	   docker/docker-runc \
	   docker/docker-containerd-shim \
	usr/bin/
endif
	([ -e docker/docker-init ] && mv docker/docker-init usr/bin/) || true
	([ -d docker/completion ] && rm -rf docker/completion) || true

hub: cleanusr
	(DOCKER_CONTENT_TRUST=1 docker run --rm $(DOCKER_IMAGE) tar cf - -C /usr/local/bin . && touch ok) | tar xf - -C usr/bin
	rm ok
	rm -f usr/bin/docker-entrypoint.sh

clean:
	rm -rf usr/ docker/ ok

.DELETE_ON_ERROR:
