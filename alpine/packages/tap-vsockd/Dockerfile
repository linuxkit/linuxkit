FROM mobylinux/alpine-build-c:1ea690a7438cdd8a5965eaa034e0a0c521d9cb40

COPY . /tap-vsockd

WORKDIR /tap-vsockd

RUN cc -Wall -Werror tap-vsockd.c hvsock.c protocol.c -lpthread -o tap-vsockd

CMD ["tar", "cf", "-", "tap-vsockd"]
