all: vsudd

files:
	ls Dockerfile *.go > files
	printf -- '-C\n..\n' >> files
	(cd .. && find vendor) >> files

vsudd: Dockerfile $(wildcard *.go) ../vendor/manifest files
	tar cf - -T files | docker build -t vsudd:build -
	docker run --rm vsudd:build | tar xf -

clean:
	rm -rf vsudd files

.DELETE_ON_ERROR:
