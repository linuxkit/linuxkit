.PHONY: tag push
default: push

IMAGE=tini
BASE=linuxkit/c-compile:f52f485825c890d581e82a62af6906c1d33d8e5d
SHA_IMAGE=alpine:3.5@sha256:dfbd4a3a8ebca874ebd2474f044a0b33600d4523d03b0df76e5c5986cb02d7e8
DEPS=Dockerfile Makefile

hash: $(DEPS)
	DOCKER_CONTENT_TRUST=1 docker pull $(BASE)
	find $^ -type f | xargs cat | docker run --rm -i $(SHA_IMAGE) sha1sum - | sed 's/ .*//' > hash

tag: hash
	docker pull linuxkit/$(IMAGE):$(shell cat hash) || \
		(docker build --no-cache -t $(IMAGE):build . && \
		 docker tag $(IMAGE):build linuxkit/$(IMAGE):$(shell cat hash))

push: tag
	docker pull linuxkit/$(IMAGE):$(shell cat hash) || \
		docker push linuxkit/$(IMAGE):$(shell cat hash)
	rm -f hash
	docker rmi $(IMAGE):build || true

sign: tag
	DOCKER_CONTENT_TRUST=1 docker pull linuxkit/$(IMAGE):$(shell cat hash) || \
		DOCKER_CONTENT_TRUST=1 docker push linuxkit/$(IMAGE):$(shell cat hash)
	rm -f hash
	docker rmi $(IMAGE):build || true

clean:
	rm -f hash
	docker rmi $(IMAGE):build || true

.DELETE_ON_ERROR:
