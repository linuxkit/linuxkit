.PHONY: tag push

ORG?=linuxkit
IMAGE=alpine
BASE=alpine:3.6

default: push

hash: Dockerfile Makefile packages
	DOCKER_CONTENT_TRUST=1 docker pull $(BASE)
	docker build --no-cache -t $(IMAGE):build .
	docker run --rm $(IMAGE):build sh -c 'echo Dockerfile /lib/apk/db/installed $$(find /mirror -name '*.apk' -type f) $$(find /go/bin -type f) | xargs cat | sha1sum' | sed 's/ .*//' > $@

push: hash
	DOCKER_CONTENT_TRUST=1 docker pull $(ORG)/$(IMAGE):$(shell cat hash) || \
		(docker tag $(IMAGE):build $(ORG)/$(IMAGE):$(shell cat hash) && \
		 DOCKER_CONTENT_TRUST=1 docker push $(ORG)/$(IMAGE):$(shell cat hash))
	docker run --rm $(IMAGE):build find /mirror -name '*.apk' -exec basename '{}' .apk \; | sort | (echo '# automatically generated list of installed packages'; cat -) > versions
	docker rmi $(IMAGE):build
	rm -f hash

tag: hash
	docker pull $(ORG)/$(IMAGE):$(shell cat hash) || \
		docker tag $(IMAGE):build $(ORG)/$(IMAGE):$(shell cat hash)
	docker rmi $(IMAGE):build
	rm -f hash

clean:
	rm -f hash
