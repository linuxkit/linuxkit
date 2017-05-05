.PHONY: tag push

TRUST=0
ORG=mor1
BASE=aarch64/alpine:3.5
IMAGE=go-compile

default: push

hash: Dockerfile compile.sh
	DOCKER_CONTENT_TRUST=$(TRUST) docker pull $(BASE)
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build -
	docker run --rm --entrypoint=/bin/sh $(IMAGE):build -c "cat $^ /lib/apk/db/installed /go/bin/golint | sha1sum" | sed 's/ .*//' > hash

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
