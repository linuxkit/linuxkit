.PHONY: tag push

TRUST=0
ORG=mor1
BASE=aarch64/alpine:3.5
IMAGE=golang-arm64

default: push

hash: Dockerfile go-wrapper no-pic.patch
	DOCKER_CONTENT_TRUST=$(TRUST) docker pull $(BASE)
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build -
	docker run --rm --entrypoint=sh $(IMAGE):build -c "cat /Dockerfile /usr/local/bin/go-wrapper /no-pic.patch /lib/apk/db/installed /go/bin/golint | sha1sum" | sed 's/ .*//' > hash

push: hash
	docker pull $(ORG)/$(IMAGE):$(shell cat hash) || \
		(docker tag $(IMAGE):build $(ORG)/$(IMAGE):$(shell cat hash) && \
		 docker push $(ORG)/$(IMAGE):$(shell cat hash))
	docker rmi $(IMAGE):build
	rm -f hash

tag: hash
	docker pull $(ORG)/$(IMAGE):$(shell cat hash) || \
		docker tag $(IMAGE):build $(ORG)/$(IMAGE):$(shell cat hash)
	docker rmi $(IMAGE):build
	rm -f hash

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

.DELETE_ON_ERROR:
