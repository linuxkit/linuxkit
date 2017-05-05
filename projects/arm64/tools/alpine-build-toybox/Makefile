.PHONY: tag push

ORG=mor1
BASE=aarch64/alpine:3.5
IMAGE=alpine-build-toybox

default: push

hash: Dockerfile build.sh
	DOCKER_CONTENT_TRUST=0 docker pull $(BASE)
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build -
	docker run --rm --entrypoint sh $(IMAGE):build -c 'cat /Dockerfile /build.sh /lib/apk/db/installed | sha1sum' | sed 's/ .*//' > hash

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
