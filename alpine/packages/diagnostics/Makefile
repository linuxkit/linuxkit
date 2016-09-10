all: diagnostics-server

files:
	ls Dockerfile *.go > files
	printf -- '-C\n..\n' >> files
	(cd .. && find vendor) >> files

diagnostics-server: Dockerfile $(wildcard *.go) ../vendor/manifest files
	tar cf - -T files | docker build -t diagnostics-server:build -
	docker run --rm diagnostics-server:build | tar xf -

clean:
	rm -rf diagnostics-server files

.DELETE_ON_ERROR:
