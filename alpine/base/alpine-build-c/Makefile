.PHONY: tag push

BASE=alpine:3.4
IMAGE=alpine-build-c

default: push

hash:
	docker pull $(BASE)
	tar cf - Dockerfile | docker build --no-cache -t $(IMAGE):build -
	docker run --rm $(IMAGE):build sha1sum /lib/apk/db/installed | sed 's/ .*//' > hash

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
