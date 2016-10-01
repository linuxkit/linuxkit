FROM mobylinux/alpine-build-c:f97e13b3a7339af5da5d620fc053c6a6c81b00a6

COPY . /tap-vsockd

WORKDIR /tap-vsockd

RUN cc -Wall -Werror tap-vsockd.c hvsock.c protocol.c -lpthread -o tap-vsockd

CMD ["tar", "cf", "-", "tap-vsockd"]
