DOCKER_BIN_URL?=https://s3.amazonaws.com/editions-stage-us-east-1-150610-005505/linux/static/stable/x86_64/docker-17.03.0-ce.tgz
DOCKER_VERSION?=17.03.0-ce
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

S3_PREFIX=s3://

usr/bin/docker: Makefile
	mkdir -p usr/bin
	rm -f usr/bin/* ok
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
	if [ -d "./bundles" ]; then \
	mv bundles/${DOCKER_VERSION}/binary-daemon/docker-containerd-ctr \
		bundles/${DOCKER_VERSION}/binary-client/docker \
		bundles/${DOCKER_VERSION}/binary-daemon/docker-containerd \
		bundles/${DOCKER_VERSION}/binary-daemon/dockerd \
		bundles/${DOCKER_VERSION}/binary-daemon/docker-proxy \
		bundles/${DOCKER_VERSION}/binary-daemon/docker-runc \
		bundles/${DOCKER_VERSION}/binary-daemon/docker-containerd-shim \
	usr/bin/ ;\
	else \
	mv docker/docker-containerd-ctr \
	   docker/docker \
	   docker/docker-containerd \
	   docker/dockerd \
	   docker/docker-proxy \
	   docker/docker-runc \
	   docker/docker-containerd-shim \
	usr/bin/ ; fi 
	([ -e docker/docker-init ] && mv docker/docker-init usr/bin/) || true
	([ -d docker/completion ] && rm -rf docker/completion) || true

clean:
	rm -rf usr/ docker/ ok

.DELETE_ON_ERROR:
