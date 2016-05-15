.PHONY: all

DEPS=9pmount-vsock.c compat.h

all: Dockerfile $(DEPS)
	docker build -t 9pmount-vsock:build .
	docker run --rm 9pmount-vsock:build cat 9pmount-vsock > 9pmount-vsock
	chmod 755 9pmount-vsock

9pmount-vsock: 9pmount-vsock.o
	gcc -Wall -Werror -o 9pmount-vsock 9pmount-vsock.o -lpthread

9pmount-vsock.o: 9pmount-vsock.c compat.h
	gcc -Wall -Werror -c 9pmount-vsock.c

clean:
	rm -f 9pmount-vsock
	docker images -q 9pmount-vsock:build | xargs docker rmi -f
