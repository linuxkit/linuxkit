.PHONY: tag push

BASE=alpine:3.5
IMAGE=alpine-build-kernel

default: push

hash:
	DOCKER_CONTENT_TRUST=1 docker pull $(BASE)
	tar cf - Dockerfile | docker build --no-cache -t $(IMAGE):build -
	docker run --rm $(IMAGE):build sha1sum /lib/apk/db/installed | sed 's/ .*//' > hash

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

clean:
	rm -f hash

.DELETE_ON_ERROR:
