.PHONY: tag push
default: push

ORG?=mobytest
IMAGE=init-container
DEPS=Dockerfile $(wildcard etc/init.d/*) $(wildcard bin/*)

HASH?=$(shell git ls-tree HEAD -- ../$(notdir $(CURDIR)) | awk '{print $$3}')

tag: $(DEPS)
	docker build --no-cache --network=none -t $(ORG)/$(IMAGE):$(HASH) .

push: tag
	docker pull $(ORG)/$(IMAGE):$(HASH) || \
	docker push $(ORG)/$(IMAGE):$(HASH)

forcepush: tag
	docker push $(ORG)/$(IMAGE):$(HASH)
