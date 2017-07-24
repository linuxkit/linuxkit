.PHONY: tag push

ORG?=linuxkit
IMAGE=alpine

DOCKER_CONTENT_VAR=1
PKG_DEPS=packages

ifeq ($(shell uname -m), x86_64)
BASE=alpine:3.6
PKG_DEPS += packages.x86_64
endif

ifeq ($(shell uname -m), aarch64)
BASE=arm64v8/alpine:3.6
PKG_DEPS += packages.aarch64
DOCKER_CONTENT_VAR=0
endif

default: push

show-tag:
	@sed -n -e '1s/# \(.*\/.*:[0-9a-f]\{40\}\)/\1/p;q' versions.$(shell uname -m)

hash: Dockerfile Makefile $(PKG_DEPS)
	DOCKER_CONTENT_TRUST=$(DOCKER_CONTENT_VAR) docker pull $(BASE)
	docker build --no-cache --build-arg BASE=$(BASE) -t $(IMAGE):build .
	docker run --rm $(IMAGE):build sh -c 'echo Dockerfile /lib/apk/db/installed $$(find /mirror -name '*.apk' -type f) $$(find /go/bin -type f) | xargs cat | sha1sum' | sed 's/ .*//' > $@

push: hash
	DOCKER_CONTENT_TRUST=$(DOCKER_CONTENT_VAR) docker pull $(ORG)/$(IMAGE):$(shell cat hash) || \
		(docker tag $(IMAGE):build $(ORG)/$(IMAGE):$(shell cat hash) && \
		 DOCKER_CONTENT_TRUST=$(DOCKER_CONTENT_VAR) docker push $(ORG)/$(IMAGE):$(shell cat hash))
	echo "# $(ORG)/$(IMAGE):$(shell cat hash)" > versions.$(shell uname -m)
	docker run --rm $(IMAGE):build find /mirror -name '*.apk' -exec basename '{}' .apk \; | sort | (echo '# automatically generated list of installed packages'; cat -) >> versions.$(shell uname -m)
	docker rmi $(IMAGE):build
	rm -f hash

tag: hash
	docker pull $(ORG)/$(IMAGE):$(shell cat hash) || \
		docker tag $(IMAGE):build $(ORG)/$(IMAGE):$(shell cat hash)
	docker rmi $(IMAGE):build
	rm -f hash

clean:
	rm -f hash
