GO_COMPILE=mobylinux/go-compile:3afebc59c5cde31024493c3f91e6102d584a30b9@sha256:e0786141ea7df8ba5735b63f2a24b4ade9eae5a02b0e04c4fca33b425ec69b0a

SHA_IMAGE=alpine:3.5@sha256:dfbd4a3a8ebca874ebd2474f044a0b33600d4523d03b0df76e5c5986cb02d7e8

MEMLOGD_BINARY=usr/bin/memlogd
LOGWRITE_BINARY=usr/bin/logwrite
STARTMEMLOGD_BINARY=usr/bin/startmemlogd
LOGREAD_BINARY=sbin/logread

IMAGE=memlogd

.PHONY: tag push clean container
default: tag

DEPS=$(MEMLOGD_BINARY) $(LOGWRITE_BINARY) $(STARTMEMLOGD_BINARY) $(LOGREAD_BINARY)

$(MEMLOGD_BINARY): cmd/memlogd/main.go
	mkdir -p $(dir $@)
	tar -Ccmd/memlogd -cf - main.go | docker run --rm --net=none --log-driver=none -i $(GO_COMPILE) -o $@ | tar xf -

$(LOGWRITE_BINARY): cmd/logwrite/main.go
	mkdir -p $(dir $@)
	tar -Ccmd/logwrite -cf - main.go | docker run --rm --net=none --log-driver=none -i $(GO_COMPILE) -o $@ | tar xf -

$(STARTMEMLOGD_BINARY): cmd/startmemlogd/main.go
	mkdir -p $(dir $@)
	tar -Ccmd/startmemlogd -cf - main.go | docker run --rm --net=none --log-driver=none -i $(GO_COMPILE) -o $@ | tar xf -

$(LOGREAD_BINARY): cmd/logread/main.go
	mkdir -p $(dir $@)
	tar -Ccmd/logread -cf - main.go | docker run --rm --net=none --log-driver=none -i $(GO_COMPILE) -o $@ | tar xf -

containers: $(MEMLOGD_BINARY) Dockerfile.memlogd config.json
	mkdir -p containers/init/memlogd/rootfs
	tar -cf - $^ | docker build -f Dockerfile.memlogd -t $(IMAGE):build1 --no-cache -
	docker create --name $(IMAGE)-build1 $(IMAGE):build1
	docker export $(IMAGE)-build1 | tar -Ccontainers/init/memlogd/rootfs -xv -
	docker rm $(IMAGE)-build1
	docker rmi $(IMAGE):build1
	mv containers/init/memlogd/rootfs/Dockerfile.memlogd containers/init/memlogd/rootfs/Dockerfile
	mv containers/init/memlogd/rootfs/config.json containers/init/memlogd

container: Dockerfile $(LOGWRITE_BINARY) $(STARTMEMLOGD_BINARY) $(LOGREAD_BINARY) containers
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build -

hash: Dockerfile Dockerfile.memlogd $(DEPS)
	find $^ -type f | xargs cat | docker run --rm -i $(SHA_IMAGE) sha1sum - | sed 's/ .*//' > hash

push: hash container
	docker pull linuxkit/$(IMAGE):$(shell cat hash) || \
		(docker tag $(IMAGE):build linuxkit/$(IMAGE):$(shell cat hash) && \
		 docker push linuxkit/$(IMAGE):$(shell cat hash))
	docker rmi $(IMAGE):build
	rm -f hash

tag: hash container
	docker pull linuxkit/$(IMAGE):$(shell cat hash) || \
		docker tag $(IMAGE):build linuxkit/$(IMAGE):$(shell cat hash)
	docker rmi $(IMAGE):build
	rm -f hash

clean:
	rm -rf hash usr containers sbin

.DELETE_ON_ERROR:

