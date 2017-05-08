.PHONY: tag push
default: push

ORG?=linuxkit
IMAGE=init
DEPS=Dockerfile init $(wildcard etc/*) $(wildcard etc/init.d/*)

HASH?=$(shell git ls-tree HEAD -- ../$(notdir $(CURDIR)) | awk '{print $$3}')

tag: $(DEPS)
	docker build --no-cache --network=none -t $(ORG)/$(IMAGE):$(HASH) .

push: tag
	docker pull $(ORG)/$(IMAGE):$(HASH) || \
	docker push $(ORG)/$(IMAGE):$(HASH)
