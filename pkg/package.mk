.PHONY: image tag show-tag
default: push

ORG?=linuxkit
ifeq ($(HASH),)
HASH_COMMIT?=HEAD # Setting this is only really useful with the show-tag target
HASH?=$(shell git ls-tree --full-tree $(HASH_COMMIT) -- $(CURDIR) | awk '{print $$3}')

ifneq ($(HASH_COMMIT),HEAD) # Others can't be dirty by definition
DIRTY:=$(shell git update-index -q --refresh && git diff-index --quiet HEAD -- $(CURDIR) || echo "-dirty")
endif
endif

TAG:=$(ORG)/$(IMAGE):$(HASH)$(DIRTY)

BASE_DEPS=Dockerfile Makefile

# Get a release tag, if present
RELEASE:=$(shell git tag -l --points-at HEAD)

ifdef NETWORK
NET_OPT=
else
NET_OPT=--network=none
endif

ifeq ($(DOCKER_CONTENT_TRUST),)
ifndef NOTRUST
export DOCKER_CONTENT_TRUST=1
endif
endif

show-tag:
	@echo $(TAG)

tag: $(BASE_DEPS) $(DEPS)
	docker pull $(TAG) || docker build $(NET_OPT) -t $(TAG) .

push: tag
ifneq ($(DIRTY),)
	$(error Your repository is not clean. Will not push package image.)
endif
	docker pull $(TAG) || docker push $(TAG)
ifneq ($(RELEASE),)
	docker tag $(TAG) $(ORG)/$(IMAGE):$(RELEASE)
	docker push $(ORG)/$(IMAGE):$(RELEASE)
endif
