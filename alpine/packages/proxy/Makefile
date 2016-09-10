all: usr/bin/slirp-proxy sbin/proxy-vsockd

files:
	ls Dockerfile *.go > files
	find libproxy >> files
	printf -- '-C\n..\n' >> files
	(cd .. && find vendor) >> files

proxy: Dockerfile $(wildcard *.go libproxy/*.go) ../vendor/manifest files
	tar cf - -T files | docker build -t proxy:build -
	docker run --rm --net=none proxy:build | tar xf -

usr/bin/slirp-proxy: proxy
	mkdir -p usr/bin
	cp proxy $@

sbin/proxy-vsockd: proxy
	mkdir -p sbin
	cp proxy $@

clean:
	rm -rf proxy files sbin usr

.DELETE_ON_ERROR:
