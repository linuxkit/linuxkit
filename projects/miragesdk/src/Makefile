.PHONY: tag push
default: push

ORG?=linuxkitprojects
IMAGE=dhcp-client
NOCACHE?=

DEPS=Dockerfile \
	$(shell find . -name jbuild) \
	$(shell find . -name '*.ml') \
	$(shell find . -name '*.mli') \
	$(shell find . -name '*.c') \
	$(shell find . -name '*.json')

HASH?=$(shell git ls-tree HEAD -- ../$(notdir $(CURDIR)) | awk '{print $$3}')

tag: $(DEPS)
	docker build --squash $(NOCACHE) -t $(ORG)/$(IMAGE):$(HASH) .

push: tag
	DOCKER_CONTENT_TRUST=1 docker pull $(ORG)/$(IMAGE):$(HASH) || \
	DOCKER_CONTENT_TRUST=1 docker push $(ORG)/$(IMAGE):$(HASH)

#### DEV

.PHONY: test

test:
	jbuilder runtest --dev

clean:
	jbuilder clean

dev:
	jbuilder build sdk.install dhcp-client.install --dev
	jbuilder build test/test.exe --dev
