all: 9pudfuse

9pudfuse: Dockerfile fuse_socket_9p.ml
	docker build -t 9pudfuse:test .
	docker run 9pudfuse:test \
		cat fuse_socket_9p.native > 9pudfuse
	chmod 755 9pudfuse

clean:
	rm -f 9pudfuse
