.PHONY: tag push clean container
default: push

#
# Build the tools and extract the rngd binary
#
RNG_TOOLS_DEPS=Dockerfile.build Makefile fix-textrels-on-PIC-x86.patch sha256sums
RNGD_BINARY=usr/sbin/rngd
image.id: $(RNG_TOOLS_DEPS)
	@echo "== Compiling rng-tools"
	BUILD=$$(docker build -f $< -q .) && [ -n "$$BUILD" ] && echo "Built $$BUILD" && echo "$$BUILD" > $@

$(RNGD_BINARY): image.id
	@echo "== Extracting rngd"
	mkdir -p $(dir $@)
	docker run --rm --net=none --log-driver=none $(shell cat $<) > $@
	chmod ugo+x $@

#
# Build the package
#
IMAGE=rngd

TINI_IMAGE=mobylinux/tini:6b25b62f4d893de8721fd2581411039b17e8a253@sha256:39b4a459018ffc155a9fcbbf952fa625c77f5a8d7599b326eade529d3dc723fc
TINI_BINARY=bin/tini

SHA_IMAGE=alpine:3.5@sha256:dfbd4a3a8ebca874ebd2474f044a0b33600d4523d03b0df76e5c5986cb02d7e8

$(TINI_BINARY):
	@echo "== Extracting tini binary"
	mkdir -p $(dir $@)
	docker run --rm --net=none $(TINI_IMAGE) tar cf - $@ | tar xf -

DIRS=dev proc sys
$(DIRS):
	mkdir -p $@

DEPS=Dockerfile.pkg $(DIRS) $(TINI_BINARY) $(RNGD_BINARY) Makefile

hash: $(DEPS) $(RNG_TOOLS_DEPS)
	@echo "== Calculating hash from inputs"
	find $^ -type f | xargs cat | docker run --rm -i $(SHA_IMAGE) sha1sum - | sed 's/ .*//' > $@

container: $(DEPS) $(RNG_TOOLS_DEPS) hash
	@echo "== Creating package container"
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build -f $< -

push: hash container
	@echo "== Tag container with hash and push (if not already there)"
	docker pull mobylinux/$(IMAGE):$(shell cat hash) || \
	(docker tag $(IMAGE):build mobylinux/$(IMAGE):$(shell cat hash) && \
		docker push mobylinux/$(IMAGE):$(shell cat hash))
	docker rmi $(IMAGE):build
	rm -f hash

tag: hash container
	@echo "== Tag package container with hash"
	docker pull mobylinux/$(IMAGE):$(shell cat hash) || \
	docker tag $(IMAGE):build mobylinux/$(IMAGE):$(shell cat hash)
	docker rmi $(IMAGE):build
	rm -f hash

clean:
	rm -rf image.id hash $(DIRS) usr bin

.DELETE_ON_ERROR:
