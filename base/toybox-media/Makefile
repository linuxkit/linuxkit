.PHONY: tag push
BASE=mobylinux/alpine-build-toybox:bec336327b738df1e225bd845f416762a6b12dfc@sha256:54c813bcdd13c1108bb69ffdc9f3c0135db94625f1e57a885277717703bbf22b
IMAGE=toybox-media

default: push

hash: 
	docker run --rm $(BASE) ls tar sh find sha1sum | \
		docker build -q -t $(IMAGE):build -
	docker run --rm $(IMAGE):build tar cf - bin | docker run -i $(IMAGE):build sha1sum -b - > hash

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
