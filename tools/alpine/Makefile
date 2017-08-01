.PHONY: tag push

ORG?=linuxkit
IMAGE=alpine
DEPS=packages

# The logic for content trust is a bit convoluted because:
# - The arm64 base image is currently not signed so we need to pull it
#   with content trust disabled. This is controlled by
#   DOCKER_CONTENT_PULL.
# - 'docker build' with the FROM image supplied as environment
#   variable *and* with DOCKER_CONTENT_TRUST=1 currently does not work
#   (https://github.com/moby/moby/issues/34199). We therefor build
#   with DOCKER_CONTENT_TRUST explicitly set to 0. However, we pull
#   the base image just before with content trust enabled (if
#   supported, see above).
# - By default we always pull and push the linuxkit/alpine image with
#   content trust, unless explicitly disabled with NOTRUST. Once the
#   above issues are resolved, this will be the only mechanism to control
#   content trust.
ifdef NOTRUST
DOCKER_CONTENT_PULL=0
else
DOCKER_CONTENT_PULL=1
export DOCKER_CONTENT_TRUST=1
endif

ARCH := $(shell uname -m)
ifeq ($(ARCH), x86_64)
BASE=alpine:3.6
DEPS += packages.x86_64
SUFFIX=-amd64
endif
ifeq ($(ARCH), aarch64)
BASE=arm64v8/alpine:3.6
DEPS += packages.aarch64
SUFFIX=-arm64
DOCKER_CONTENT_PULL=0
endif

default: push

show-tag:
	@sed -n -e '1s/# \(.*\/.*:[0-9a-f]\{40\}\)/\1/p;q' versions.$(ARCH)

iid: Dockerfile Makefile $(DEPS)
	DOCKER_CONTENT_TRUST=$(DOCKER_CONTENT_PULL) docker pull $(BASE)
	DOCKER_CONTENT_TRUST=0 docker build --no-cache --build-arg BASE=$(BASE) --iidfile iid .

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

tag: hash iid versions.$(ARCH)
	docker pull $(ORG)/$(IMAGE):$(shell cat hash) || \
		docker tag $(shell cat iid) $(ORG)/$(IMAGE):$(shell cat hash)
	rm -f iid

clean:
	rm -f hash iid
