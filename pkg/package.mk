.PHONY: image tag show-tag
default: push

ORG?=linuxkit
HASH?=$(shell git ls-tree HEAD -- ../$(notdir $(CURDIR)) | awk '{print $$3}')
BASE_DEPS=Dockerfile Makefile

# Add '-dirty' to hash if the repository is not clean. make does not
# concatenate strings without spaces, so we use the documented trick
# of replacing the space with nothing.
DIRTY=$(shell git diff-index --quiet HEAD --; echo $$?)
ifneq ($(DIRTY),0)
HASH+=-dirty
nullstring :=
space := $(nullstring) $(nullstring)
TAG=$(subst $(space),,$(HASH))
else
TAG=$(HASH)
endif

# Get a release tag, if present
RELEASE=$(shell git tag -l --points-at HEAD)

ifdef NETWORK
NET_OPT=
else
NET_OPT=--network=none
endif

show-tag:
	@echo $(ORG)/$(IMAGE):$(TAG)

tag: $(BASE_DEPS) $(DEPS)
	DOCKER_CONTENT_TRUST=1 docker pull $(ORG)/$(IMAGE):$(TAG) || \
	docker build $(NET_OPT) -t $(ORG)/$(IMAGE):$(TAG) .

push: tag
ifneq ($(DIRTY),0)
	$(error Your repository is not clean. Will not push package image.)
endif
	DOCKER_CONTENT_TRUST=1 docker pull $(ORG)/$(IMAGE):$(TAG) || \
	DOCKER_CONTENT_TRUST=1 docker push $(ORG)/$(IMAGE):$(TAG)
ifneq ($(RELEASE),)
	docker tag $(ORG)/$(IMAGE):$(TAG) $(ORG)/$(IMAGE):$(RELEASE)	
	DOCKER_CONTENT_TRUST=1 docker push $(ORG)/$(IMAGE):$(RELEASE)
endif
