.PHONY: tag push

IMAGE=riddler

BASE=golang:1.7-alpine

default: push

hash: Dockerfile riddler.sh
	DOCKER_CONTENT_TRUST=1 docker pull $(BASE)
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build -
	docker run --entrypoint=/bin/sh --rm $(IMAGE):build -c 'cat /Dockerfile /usr/bin/riddler.sh /lib/apk/db/installed | sha1sum' | sed 's/ .*//' > hash

push: hash
	docker pull mobylinux/$(IMAGE):$(shell cat hash) || \
		 (docker tag $(IMAGE):build mobylinux/$(IMAGE):latest && \
		  docker push mobylinux/$(IMAGE):$(shell cat hash))
	docker rmi $(IMAGE):build
	rm -f hash

tag: hash
	docker pull mobylinux/$(IMAGE):$(shell cat hash) || \
		docker tag $(IMAGE):build mobylinux/$(IMAGE):$(shell cat hash)
	docker rmi $(IMAGE):build
	rm -f hash

clean:
	rm -f hash

.DELETE_ON_ERROR:
