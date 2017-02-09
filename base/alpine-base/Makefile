.PHONY: tag push

BASE=alpine:3.5
IMAGE=alpine-base

default: push

hash: Dockerfile repositories selinux-config
	DOCKER_CONTENT_TRUST=1 docker pull $(BASE)
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build -
	docker run --rm $(IMAGE):build sh -c 'cat /lib/apk/db/installed /etc/selinux/config | sha1sum' | sed 's/ .*//' > $@

push: hash
	docker pull mobylinux/$(IMAGE):$(shell cat hash) || \
		(docker tag $(IMAGE):build mobylinux/$(IMAGE):$(shell cat hash) && \
		 docker push mobylinux/$(IMAGE):$(shell cat hash))
	docker run --rm $(IMAGE):build cat /lib/apk/db/installed | grep -E '^(P|V)' | \
	  awk  '/^P/{printf substr($$1, 3),$$0;next} /^V/{print " " substr($$1, 3);next}' | sort > packages
	docker rmi -f $(IMAGE):build
	rm -f hash

tag: hash
	docker pull mobylinux/$(IMAGE):$(shell cat hash) || \
		docker tag $(IMAGE):build mobylinux/$(IMAGE):$(shell cat hash)
	docker rmi -f $(IMAGE):build
	rm -f hash

clean:
	rm -f hash

.DELETE_ON_ERROR:
