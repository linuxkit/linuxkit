FROM mobylinux/alpine-build-c:7303e33e9dcd5276b8bb5269644a9bf3354008c8

COPY . /tap-vsockd

WORKDIR /tap-vsockd

RUN cc -Wall -Werror tap-vsockd.c hvsock.c protocol.c -lpthread -o tap-vsockd

CMD ["tar", "cf", "-", "tap-vsockd"]
