all: proxy

files:
	ls Dockerfile *.go > files
	find libproxy >> files
	printf -- '-C\n..\n' >> files
	(cd .. && find vendor) >> files

proxy: Dockerfile $(wildcard *.go libproxy/*.go) ../vendor/manifest files
	tar cf - -T files | docker build -t proxy:build -
	docker run --rm proxy:build | tar xf -

clean:
	rm -rf proxy files

.DELETE_ON_ERROR:
