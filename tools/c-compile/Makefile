.PHONY: tag push

BASE=alpine:3.5
IMAGE=c-compile

default: push

hash: Dockerfile compile.sh 
	DOCKER_CONTENT_TRUST=1 docker pull $(BASE)
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build -
	docker run --rm --entrypoint=/bin/sh $(IMAGE):build -c 'cat /lib/apk/db/installed /usr/bin/compile.sh | sha1sum' | sed 's/ .*//' > hash

push: hash
	docker pull linuxkit/$(IMAGE):$(shell cat hash) || \
		(docker tag $(IMAGE):build linuxkit/$(IMAGE):$(shell cat hash) && \
		 docker push linuxkit/$(IMAGE):$(shell cat hash))
	docker rmi $(IMAGE):build
	rm -f hash

tag: hash
	docker pull linuxkit/$(IMAGE):$(shell cat hash) || \
		docker tag $(IMAGE):build linuxkit/$(IMAGE):$(shell cat hash)
	docker rmi $(IMAGE):build
	rm -f hash

signed-tag: hash
	DOCKER_CONTENT_TRUST=1 docker pull linuxkit/$(IMAGE):$(shell cat hash) || \
		(DOCKER_CONTENT_TRUST=1 docker pull $(BASE) && \
		 docker build --no-cache -t $(IMAGE):build . && \
		 docker tag $(IMAGE):build linuxkit/$(IMAGE):$(shell cat hash))

sign: signed-tag
	DOCKER_CONTENT_TRUST=1 docker pull linuxkit/$(IMAGE):$(shell cat hash) || \
		DOCKER_CONTENT_TRUST=1 docker push linuxkit/$(IMAGE):$(shell cat hash)
	rm -f hash
	docker rmi $(IMAGE):build || true

clean:
	rm -f hash

.DELETE_ON_ERROR:
