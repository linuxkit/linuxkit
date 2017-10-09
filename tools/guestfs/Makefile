.PHONY: build push

IMAGE=guestfs

export DOCKER_CONTENT_TRUST=1

default: push

iid: Dockerfile
	tar cf - $^ | docker build --no-cache --iidfile iid -

hash: Makefile iid
	docker run --rm $(shell cat iid) sh -c 'apt list --installed 2>/dev/null | sha1sum' | sed 's/ .*//' > hash

push: hash iid
	docker pull linuxkit/$(IMAGE):$(shell cat hash) || \
		(docker tag $(shell cat iid) linuxkit/$(IMAGE):$(shell cat hash) && \
		 docker push linuxkit/$(IMAGE):$(shell cat hash))
	rm -f iid

build: hash iid
	docker pull linuxkit/$(IMAGE):$(shell cat hash) || \
		docker tag $(shell cat iid) linuxkit/$(IMAGE):$(shell cat hash)
	rm -f iid

clean:
	rm -f hash iid

.DELETE_ON_ERROR:
