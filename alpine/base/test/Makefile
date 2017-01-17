# Tag b784b685b2c39b9bed4d58a989d06d8f0281b87c
MKSH_IMAGE=mobylinux/mksh@sha256:b3ca9febef294d002894b91e0ce0f794235db73a3024be3cae16d34022913de3
MKSH_BINARY=bin/mksh
SH_BINARY=bin/sh

# Tag: 41e4b91c9a619e46f76ce2d024067c09b62f07b4
CACERT_IMAGE=mobylinux/ca-certificates@sha256:6ca2dca9cfb8534a55f3a17f8797943527db5bbac08c98a5c9a4836250f4c548
CACERT_FILE=etc/ssl/certs/ca-certificates.crt

TEST_SCRIPT=bin/test.sh
MOBY_WHALE=etc/moby

.PHONY: tag push clean container
default: push

$(MKSH_BINARY):
	mkdir -p $(dir $@)
	docker run --rm --net=none $(MKSH_IMAGE) tar cf - $@ | tar xf -

$(SH_BINARY): $(MKSH_BINARY)
	mv $(MKSH_BINARY) $(SH_BINARY)

$(CACERT_FILE):
	mkdir -p $(dir $@)
	docker run --rm --net=none $(CACERT_IMAGE) tar cf - $@ | tar xf -

# Tag: alpine:3.5
SHA_IMAGE=alpine@sha256:dfbd4a3a8ebca874ebd2474f044a0b33600d4523d03b0df76e5c5986cb02d7e8

IMAGE=test

DIRS=dev proc sys
$(DIRS):
	mkdir -p $@

DEPS=$(DIRS) $(SH_BINARY) $(CACERT_FILE) $(TEST_SCRIPT) $(MOBY_WHALE)

container: Dockerfile $(DEPS)
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build -

hash: Dockerfile $(DEPS)
	find $^ -type f | xargs cat | docker run --rm -i $(SHA_IMAGE) sha1sum - | sed 's/ .*//' > hash

push: hash container
	docker pull mobylinux/$(IMAGE):$(shell cat hash) || \
		(docker tag $(IMAGE):build mobylinux/$(IMAGE):$(shell cat hash) && \
		 docker push mobylinux/$(IMAGE):$(shell cat hash))
	docker rmi $(IMAGE):build
	rm -f hash

tag: hash container
	docker pull mobylinux/$(IMAGE):$(shell cat hash) || \
		docker tag $(IMAGE):build mobylinux/$(IMAGE):$(shell cat hash)
	docker rmi $(IMAGE):build
	rm -f hash

clean:
	rm -rf hash $(DIRS) $(MKSH_BINARY) etc/ssl

.DELETE_ON_ERROR:
