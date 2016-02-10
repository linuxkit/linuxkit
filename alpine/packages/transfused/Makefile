.PHONY: all

DEPS=transfused.c

all: Dockerfile $(DEPS)
	docker build -t transfused:build .
	docker run transfused:build cat transfused > transfused
	chmod 755 transfused

transfused: $(DEPS)
	gcc -g -static -Wall -Werror -o transfused transfused.c

clean:
	rm -f transfused
	docker images -q transfused:build | xargs docker rmi -f
