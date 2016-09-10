all: vsudd

files:
	ls Dockerfile *.go > files
	printf -- '-C\n..\n' >> files
	(cd .. && find vendor) >> files

vsudd: Dockerfile $(wildcard *.go) ../vendor/manifest files
	mkdir -p sbin
	tar cf - -T files | docker build -t vsudd:build -
	docker run --rm --net=none vsudd:build | tar xf - -C sbin

clean:
	rm -rf sbin files

.DELETE_ON_ERROR:
