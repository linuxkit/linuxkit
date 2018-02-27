.PHONY: build push

ORG?=linuxkit
IMAGE=alpine
DEPS=packages

ifeq ($(DOCKER_CONTENT_TRUST),)
ifndef NOTRUST
export DOCKER_CONTENT_TRUST=1
endif
endif

ARCH := $(shell uname -m)
ifeq ($(ARCH), x86_64)
DEPS += packages.x86_64
SUFFIX=-amd64
endif
ifeq ($(ARCH), aarch64)
DEPS += packages.aarch64
SUFFIX=-arm64
endif
ifeq ($(ARCH), s390x)
DEPS += packages.s390x
SUFFIX=-s390x
endif

default: push

show-tag:
	@sed -n -e '1s/# \(.*\/.*:[0-9a-f]\{40\}\)/\1/p;q' versions.$(ARCH)

iid: Dockerfile Makefile $(DEPS)
	docker build --no-cache --iidfile iid .

hash: Makefile iid
	docker run --rm $(shell cat iid) sh -c 'echo Dockerfile /lib/apk/db/installed $$(find /mirror -name '*.apk' -type f) $$(find /go/bin -type f) | xargs cat | sha1sum' | sed 's/ .*//' | sed 's/$$/$(SUFFIX)/'> $@

versions.$(ARCH): Makefile hash iid
	echo "# $(ORG)/$(IMAGE):$(shell cat hash)" > versions.$(ARCH)
	docker run --rm $(shell cat iid) find /mirror -name '*.apk' -exec basename '{}' .apk \; | LANG=C sort | (echo '# automatically generated list of installed packages'; cat -) >> versions.$(ARCH)

push: hash iid versions.$(ARCH)
	docker pull $(ORG)/$(IMAGE):$(shell cat hash) || \
		(docker tag $(shell cat iid) $(ORG)/$(IMAGE):$(shell cat hash) && \
		 docker push $(ORG)/$(IMAGE):$(shell cat hash))
	./push-manifest.sh $(ORG) $(IMAGE)
	rm -f iid

build: hash iid versions.$(ARCH)
	docker pull $(ORG)/$(IMAGE):$(shell cat hash) || \
		docker tag $(shell cat iid) $(ORG)/$(IMAGE):$(shell cat hash)
	rm -f iid

clean:
	rm -f hash iid
