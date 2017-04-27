.PHONY: tag push
default: push

IMAGE=binfmt
BASE=alpine:edge
SHA_IMAGE=alpine:3.5@sha256:dfbd4a3a8ebca874ebd2474f044a0b33600d4523d03b0df76e5c5986cb02d7e8
DEPS=Dockerfile Makefile main.go $(wildcard etc/binmft.d/*)

# Note, this isn't quite right as we pull in qemu binaries from
# alpine:edge and they may vary over time. BUT: We are going to change
# the hashing stuff soon anyway, so there is little point in faffing
# about with computing a hash during the build and get it out.
hash: $(DEPS)
	find $^ -type f | xargs cat | docker run --rm -i $(SHA_IMAGE) sha1sum - | sed 's/ .*//' > hash

tag: hash
	docker pull linuxkit/$(IMAGE):$(shell cat hash) || \
		(DOCKER_CONTENT_TRUST=1 docker pull $(BASE) && \
		 docker build --no-cache -t $(IMAGE):build . && \
		 docker tag $(IMAGE):build linuxkit/$(IMAGE):$(shell cat hash))

push: tag
	docker pull linuxkit/$(IMAGE):$(shell cat hash) || \
		docker push linuxkit/$(IMAGE):$(shell cat hash)
	rm -f hash
	docker rmi $(IMAGE):build || true

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
	docker rmi $(IMAGE):build || true

.DELETE_ON_ERROR:
