.PHONY: tag push

BASE=alpine:edge
IMAGE=alpine

default: push

define NEWLINE

endef

hash: Dockerfile packages
	DOCKER_CONTENT_TRUST=1 docker pull $(BASE)
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build -
	docker run --rm $(IMAGE):build sh -c 'find /mirror -name '*.apk' -type f | xargs cat | cat /lib/apk/db/installed - | sha1sum' | sed 's/ .*//' > $@

push: hash
	docker pull linuxkit/$(IMAGE):$(shell cat hash) || \
		(docker tag $(IMAGE):build linuxkit/$(IMAGE):$(shell cat hash) && \
		 docker push linuxkit/$(IMAGE):$(shell cat hash))
	docker run --rm $(IMAGE):build find /mirror -name '*.apk' -exec basename '{}' .apk \; | sort | (echo '# automatically generated list of installed packages'; cat -) > versions
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
