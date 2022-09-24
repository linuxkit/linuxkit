.PHONY: build push show-tag

ORG?=linuxkit
IMAGE=guestfs

ARCH := $(shell uname -m)
ifeq ($(ARCH), x86_64)
SUFFIX=-amd64
endif
ifeq ($(ARCH), aarch64)
SUFFIX=-arm64
endif
ifeq ($(ARCH), s390x)
SUFFIX=-s390x
endif

default: push

show-tag:
	@sed -n -e '1s/# \(.*\/.*:[0-9a-f]\{40\}\)/\1/p;q' versions.$(ARCH)

iid: Dockerfile
	tar cf - $^ | docker build --no-cache --iidfile iid -

hash: Makefile iid
	docker run --rm $(shell cat iid) sh -c 'apt list --installed 2>/dev/null | tail -n+2 | cat Dockerfile - | sha1sum' | sed 's/ .*//' | sed 's/$$/$(SUFFIX)/'> $@

versions.$(ARCH): hash iid
	@echo "# $(ORG)/$(IMAGE):$(shell cat hash)" > $@
	@echo "# automatically generated list of installed packages" >> $@
	docker run --rm $(shell cat iid) apt list --installed 2>/dev/null | tail -n+2 >> $@

push: hash iid versions.$(ARCH)
	docker pull $(ORG)/$(IMAGE):$(shell cat hash) || \
		(docker tag $(shell cat iid) $(ORG)/$(IMAGE):$(shell cat hash) && \
		 docker push $(ORG)/$(IMAGE):$(shell cat hash))
	rm -f iid

build: hash iid versions.$(ARCH)
	docker pull $(ORG)/$(IMAGE):$(shell cat hash) || \
		docker tag $(shell cat iid) $(ORG)/$(IMAGE):$(shell cat hash)
	rm -f iid

clean:
	rm -f hash iid

.DELETE_ON_ERROR:
