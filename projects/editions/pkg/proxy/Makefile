GO_COMPILE=linuxkit/go-compile:4513068d9a7e919e4ec42e2d7ee879ff5b95b7f5@sha256:bdfadbe3e4ec699ca45b67453662321ec270f2d1a1dbdbf09625776d3ebd68c5

default: tag

DEPS=$(wildcard *.go libproxy/*.go) 
VENDOR_PARENT=../../../../
HASH?=$(shell git ls-tree HEAD -- ../$(notdir $(CURDIR)) | awk '{print $$3}')

slirp-proxy: $(DEPS) $(VENDOR_PARENT)/vendor.conf
	mkdir -p tmp
	tar cf - $(DEPS) -C $(VENDOR_PARENT) vendor | tar xf - -C tmp
	tar cf - $(DEPS) -C $(VENDOR_PARENT) vendor | docker run --rm --net=none --log-driver=none -i $(GO_COMPILE) -o proxy | tar xf -
	mv proxy slirp-proxy

proxy-vsockd: slirp-proxy
	cp slirp-proxy $@

tag: slirp-proxy proxy-vsockd Dockerfile
	tar -cf - slirp-proxy Dockerfile | docker build --no-cache --network=none -t linuxkit/slirp-proxy:$(HASH) -
	tar -cf - proxy-vsockd Dockerfile | docker build --no-cache --network=none -t linuxkit/proxy-vsockd:$(HASH) -

push: tag
	docker pull linuxkit/slirp-proxy:$(HASH) || \
	docker push linuxkit/slirp-proxy:$(HASH)
	docker pull linuxkit/proxy-vsockd:$(HASH) || \
	docker push linuxkit/proxy-vsockd:$(HASH)

clean:
	rm -rf proxy slirp-proxy proxy-vsockd

.DELETE_ON_ERROR:
