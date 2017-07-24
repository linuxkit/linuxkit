.PHONY: tag push

ORG?=linuxkit
IMAGE=alpine

DOCKER_CONTENT_VAR=1
PKG_DEPS=packages

ARCH := $(shell uname -m)

ifeq ($(ARCH), x86_64)
BASE=alpine:3.6
PKG_DEPS += packages.x86_64
endif

ifeq ($(ARCH), aarch64)
BASE=arm64v8/alpine:3.6
PKG_DEPS += packages.aarch64
DOCKER_CONTENT_VAR=0
endif

default: push

show-tag:
	@sed -n -e '1s/# \(.*\/.*:[0-9a-f]\{40\}\)/\1/p;q' versions.$(ARCH)

iid: Dockerfile Makefile $(PKG_DEPS)
	DOCKER_CONTENT_TRUST=$(DOCKER_CONTENT_VAR) docker pull $(BASE)
	docker build --no-cache --build-arg BASE=$(BASE) --iidfile iid .

hash: Makefile iid
	docker run --rm $(shell cat iid) sh -c 'echo Dockerfile /lib/apk/db/installed $$(find /mirror -name '*.apk' -type f) $$(find /go/bin -type f) | xargs cat | sha1sum' | sed 's/ .*//' > $@

versions.$(ARCH): Makefile hash iid
	echo "# $(ORG)/$(IMAGE):$(shell cat hash)" > versions.$(ARCH)
	docker run --rm $(shell cat iid) find /mirror -name '*.apk' -exec basename '{}' .apk \; | sort | (echo '# automatically generated list of installed packages'; cat -) >> versions.$(ARCH)

push: hash iid versions.$(ARCH)
	DOCKER_CONTENT_TRUST=$(DOCKER_CONTENT_VAR) docker pull $(ORG)/$(IMAGE):$(shell cat hash) || \
		(docker tag $(shell cat iid) $(ORG)/$(IMAGE):$(shell cat hash) && \
		 DOCKER_CONTENT_TRUST=$(DOCKER_CONTENT_VAR) docker push $(ORG)/$(IMAGE):$(shell cat hash))
	rm -f iid

tag: hash iid versions.$(ARCH)
	docker pull $(ORG)/$(IMAGE):$(shell cat hash) || \
		docker tag $(shell cat iid) $(ORG)/$(IMAGE):$(shell cat hash)
	rm -f iid

clean:
	rm -f hash iid
