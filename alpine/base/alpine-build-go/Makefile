.PHONY: tag push

BASE=golang:1.7-alpine
IMAGE=alpine-build-go

default: push

hash:
	docker pull $(BASE)
	tar cf - Dockerfile | docker build --no-cache -t $(IMAGE):build -
	docker run $(IMAGE):build sh -c 'cat /usr/local/go/bin/go /lib/apk/db/installed | sha1sum' | sed 's/ .*//' > hash

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
