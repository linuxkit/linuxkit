all: 9pudfuse

9pudfuse: Dockerfile fuse_socket_9p.ml
	docker build -t 9pudfuse:build .
	docker run --rm 9pudfuse:build \
		cat fuse_socket_9p.native > 9pudfuse
	chmod 755 9pudfuse

clean:
	rm -f 9pudfuse
	docker images -q 9pudfuse:build | xargs docker rmi -f
