.PHONY: tag push

BASE=docker:1.13.1
IMAGE=docker2tar

default: push

hash: Dockerfile docker2tar.sh
	DOCKER_CONTENT_TRUST=1 docker pull $(BASE)
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build -
	docker run --entrypoint=/bin/sh --rm $(IMAGE):build -c 'cat $^ /usr/local/bin/* /lib/apk/db/installed | sha1sum' | sed 's/ .*//' > $@

push: hash
	docker pull mobylinux/$(IMAGE):$(shell cat hash) || \
		 (docker tag $(IMAGE):build mobylinux/$(IMAGE):$(shell cat hash) && \
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
