.PHONY: all

DEPS=tap-vsockd.c hvsock.c hvsock.h protocol.c protocol.h

all: Dockerfile $(DEPS)
	docker build -t tap-vsockd:build .
	docker run --rm tap-vsockd:build cat tap-vsockd > tap-vsockd
	chmod 755 tap-vsockd

tap-vsockd: hvsock.o protocol.o tap-vsockd.o
	gcc -Wall -Werror -o tap-vsockd tap-vsockd.o protocol.o hvsock.o -lpthread

hvsock.o: hvsock.c hvsock.h
	gcc -Wall -Werror -c hvsock.c

protocol.o: protocol.c
	gcc -Wall -Werror -c protocol.c

tap-vsockd.o: tap-vsockd.c hvsock.h
	gcc -Wall -Werror -c tap-vsockd.c

clean:
	rm -f tap-vsockd
	docker images -q tap-vsockd:build | xargs docker rmi -f
