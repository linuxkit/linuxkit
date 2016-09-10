all: usr/bin/diagnostics-server

files:
	ls Dockerfile *.go > files
	printf -- '-C\n..\n' >> files
	(cd .. && find vendor) >> files

usr/bin/diagnostics-server: Dockerfile $(wildcard *.go) ../vendor/manifest files
	tar cf - -T files | docker build -t diagnostics-server:build -
	docker run --rm --net=none diagnostics-server:build | tar xf - -C usr/bin

clean:
	rm -f usr/bin/diagnostics-server files

.DELETE_ON_ERROR:
