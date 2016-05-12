.PHONY: all

DEPS=tap-vsockd.c compat.h protocol.c protocol.h

all: Dockerfile $(DEPS)
	docker build -t tap-vsockd:build .
	docker run --rm tap-vsockd:build cat tap-vsockd > tap-vsockd
	chmod 755 tap-vsockd

tap-vsockd: protocol.o tap-vsockd.o
	gcc -Wall -Werror -o tap-vsockd tap-vsockd.o protocol.o -lpthread

protocol.o: protocol.c
	gcc -Wall -Werror -c protocol.c

tap-vsockd.o: tap-vsockd.c compat.h
	gcc -Wall -Werror -c tap-vsockd.c

clean:
	rm -f tap-vsockd
	docker images -q tap-vsockd:build | xargs docker rmi -f
