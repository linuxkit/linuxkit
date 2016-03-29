.PHONY: all

HDR=transfused.h transfused_log.h
SRC=transfused.c transfused_log.c
DEPS=$(HDR) $(SRC)


all: Dockerfile $(DEPS)
	docker build -t transfused:build .
	docker run --rm transfused:build cat transfused > transfused
	chmod 755 transfused

transfused: $(DEPS)
	gcc -g -static -Wall -Werror -o transfused $(SRC)

clean:
	rm -f transfused
	docker images -q transfused:build | xargs docker rmi -f
