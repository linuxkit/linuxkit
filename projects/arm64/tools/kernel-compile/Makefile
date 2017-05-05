.PHONY: tag push

ORG=mor1
BASE=aarch64/alpine:3.5
IMAGE=alpine-build-kernel-arm64

default: push

hash:
	DOCKER_CONTENT_TRUST=0 docker pull $(BASE)
	tar cf - Dockerfile | docker build --no-cache -t $(IMAGE):build -
	docker run --rm $(IMAGE):build sha1sum /lib/apk/db/installed | sed 's/ .*//' > hash

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

clean:
	rm -f hash

.DELETE_ON_ERROR:
