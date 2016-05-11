.PHONY: all

DEPS=tap-vsockd.c compat.h

all: Dockerfile $(DEPS)
	docker build -t tap-vsockd:build .
	docker run --rm tap-vsockd:build cat tap-vsockd > tap-vsockd
	chmod 755 tap-vsockd

tap-vsockd: $(DEPS)
	gcc -Wall -Werror -o tap-vsockd tap-vsockd.c -lpthread

clean:
	rm -f tap-vsockd
	docker images -q tap-vsockd:build | xargs docker rmi -f
