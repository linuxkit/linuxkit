.PHONY: tag push

IMAGE=runc

HASH?=$(shell git ls-tree HEAD -- ../$(notdir $(CURDIR)) | awk '{print $$3}')

default: push

tag: Dockerfile
	docker build -t linuxkit/$(IMAGE):$(HASH) .

push: tag
	docker pull linuxkit/$(IMAGE):$(HASH) || docker push linuxkit/$(IMAGE):$(HASH)
