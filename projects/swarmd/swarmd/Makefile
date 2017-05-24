.PHONY: tag push
all: push

ORG?=linuxkit
IMAGE=swarmd

HASH?=$(shell git ls-tree HEAD -- ../$(notdir $(CURDIR)) | awk '{print $$3}')

default: push

tag: Dockerfile
	docker build --no-cache -t $(ORG)/$(IMAGE):$(HASH) .

push: tag
	docker pull $(ORG)/$(IMAGE):$(HASH) || docker push $(ORG)/$(IMAGE):$(HASH)
