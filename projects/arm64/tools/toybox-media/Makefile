.PHONY: tag push
default: push

TRUST=0
ORG=mor1
IMAGE=toybox-media
BASE=$(ORG)/c-compile:d8aa4bec68dcd4e6a48762ed9de09ee8051ed2d3
SHA_IMAGE=aarch64/alpine:3.5
DEPS=Dockerfile Makefile

hash: $(DEPS)
	DOCKER_CONTENT_TRUST=$(TRUST) docker pull $(BASE)
	find $^ -type f | xargs cat | docker run --rm -i $(SHA_IMAGE) sha1sum - | sed 's/ .*//' > hash

tag: hash
	docker pull $(ORG)/$(IMAGE):$(shell cat hash) || \
		(docker build --no-cache -t $(IMAGE):build . && \
		 docker tag $(IMAGE):build $(ORG)/$(IMAGE):$(shell cat hash))

push: tag
	docker pull $(ORG)/$(IMAGE):$(shell cat hash) || \
		docker push $(ORG)/$(IMAGE):$(shell cat hash)
	rm -f hash
	docker rmi $(IMAGE):build || true

signed-tag: hash
	DOCKER_CONTENT_TRUST=$(TRUST) docker pull $(ORG)/$(IMAGE):$(shell cat hash) || \
		(DOCKER_CONTENT_TRUST=$(TRUST) docker pull $(BASE) && \
		 docker build --no-cache -t $(IMAGE):build . && \
		 docker tag $(IMAGE):build $(ORG)/$(IMAGE):$(shell cat hash))

sign: signed-tag
	DOCKER_CONTENT_TRUST=$(TRUST) docker pull $(ORG)/$(IMAGE):$(shell cat hash) || \
		DOCKER_CONTENT_TRUST=$(TRUST) docker push $(ORG)/$(IMAGE):$(shell cat hash)
	rm -f hash
	docker rmi $(IMAGE):build || true

clean:
	rm -f hash
	docker rmi $(IMAGE):build || true

.DELETE_ON_ERROR:
