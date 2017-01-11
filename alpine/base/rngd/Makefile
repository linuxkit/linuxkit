.PHONY: tag push

IMAGE=rngd

# Tag 7cb780fd6b60c089964e81efd6553853c491d59f
TINI_IMAGE=mobylinux/tini@sha256:7da8c5b371e0d7d3fb1778e96c0bc634e39ace7bf1e7a73bffbf1f8360127fdb
TINI_BINARY=tini

default: push

hash: Dockerfile fix-textrels-on-PIC-x86.patch sha256sums $(TINI_BINARY)
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build -
	docker run --rm --entrypoint sh $(IMAGE):build -c 'cat bin/tini usr/sbin/rngd | sha1sum' | sed 's/ .*//' > hash

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

$(TINI_BINARY): Dockerfile
	docker run --rm --net=none $(TINI_IMAGE) tar cf - -C /bin $@ | tar xf -

clean:
	rm -f hash $(TINI_BINARY)

.DELETE_ON_ERROR:
