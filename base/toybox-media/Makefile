.PHONY: tag push
BASE=linuxkit/alpine-build-toybox:1855c262e3b2b252540bab17dc09f0bb292b0633@sha256:ef748ed99678ac0a72ccccb975c45bc1c7b706fe9047c3ac62b2b80025b33d3f
IMAGE=toybox-media

default: push

hash: 
	docker run --rm $(BASE) ls tar sh find sha1sum | \
		docker build -q -t $(IMAGE):build -
	docker run --rm $(IMAGE):build tar cf - bin | docker run -i $(IMAGE):build sha1sum -b - > hash

push: hash
	docker pull linuxkit/$(IMAGE):$(shell cat hash) || \
		(docker tag $(IMAGE):build linuxkit/$(IMAGE):$(shell cat hash) && \
		 docker push linuxkit/$(IMAGE):$(shell cat hash))
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
