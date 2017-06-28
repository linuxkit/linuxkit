.PHONY: image tag show-tag
default: push

ORG?=linuxkit
HASH?=$(shell git ls-tree --full-tree HEAD -- $(CURDIR) | awk '{print $$3}')
BASE_DEPS=Dockerfile Makefile

DIRTY=$(shell git diff-index --quiet HEAD -- $(CURDIR) || echo "-dirty")
TAG=$(ORG)/$(IMAGE):$(HASH)$(DIRTY)

# Get a release tag, if present
RELEASE=$(shell git tag -l --points-at HEAD)

ifdef NETWORK
NET_OPT=
else
NET_OPT=--network=none
endif

show-tag:
	@echo $(TAG)

tag: $(BASE_DEPS) $(DEPS)
	DOCKER_CONTENT_TRUST=1 docker pull $(TAG) || \
	docker build $(NET_OPT) -t $(TAG) .

push: tag
ifneq ($(DIRTY),)
	$(error Your repository is not clean. Will not push package image.)
endif
	DOCKER_CONTENT_TRUST=1 docker pull $(TAG) || \
	DOCKER_CONTENT_TRUST=1 docker push $(TAG)
ifneq ($(RELEASE),)
	docker tag $(TAG) $(ORG)/$(IMAGE):$(RELEASE)
	DOCKER_CONTENT_TRUST=1 docker push $(ORG)/$(IMAGE):$(RELEASE)
endif
