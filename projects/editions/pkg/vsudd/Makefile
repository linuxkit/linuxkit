.PHONY: clean push
GO_COMPILE=linuxkit/go-compile:4513068d9a7e919e4ec42e2d7ee879ff5b95b7f5@sha256:bdfadbe3e4ec699ca45b67453662321ec270f2d1a1dbdbf09625776d3ebd68c5
IMAGE=vsudd
ORG?=edtestorg

default: tag

DEPS=$(wildcard *.go) ../../../../vendor ../../../../vendor.conf
HASH?=$(shell git ls-tree HEAD -- ../$(notdir $(CURDIR)) | awk '{print $$3}')

vsudd: $(DEPS) Dockerfile Makefile
	tar cf - $(DEPS) | docker run --rm --net=none --log-driver=none -i $(GO_COMPILE) -o $@ | tar xf -

tag: vsudd Dockerfile
	tar -cf - $^ | docker build --no-cache --network=none -t $(ORG)/$(IMAGE):$(HASH) -

push: tag
	docker pull $(ORG)/$(IMAGE):$(HASH) || \
	docker push $(ORG)/$(IMAGE):$(HASH)

clean:
	rm -f vsudd

.DELETE_ON_ERROR:
