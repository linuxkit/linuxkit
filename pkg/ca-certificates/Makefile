.PHONY: tag push

IMAGE=ca-certificates

HASH?=$(shell git ls-tree HEAD -- ../$(notdir $(CURDIR)) | awk '{print $$3}')

default: push

tag: Dockerfile
	docker build --network=none -t linuxkit/$(IMAGE):$(HASH) .

push: tag
	docker pull linuxkit/$(IMAGE):$(HASH) || docker push linuxkit/$(IMAGE):$(HASH)
