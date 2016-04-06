.PHONY: all

DEPS=nc-vsock.c

all: Dockerfile $(DEPS)
	docker build -t nc-vsock:build .
	docker run --rm nc-vsock:build cat nc-vsock > nc-vsock
	chmod 755 nc-vsock

nc-vsock: $(DEPS)
	gcc -Wall -Werror -o nc-vsock nc-vsock.c

clean:
	rm -f nc-vsock
	docker images -q nc-vsock:build | xargs docker rmi -f
