FROM mobylinux/alpine-build-c:50a89f207d5fc98ce750d694077c75a31ab7937d

COPY . /tap-vsockd

WORKDIR /tap-vsockd

RUN cc -Wall -Werror tap-vsockd.c hvsock.c protocol.c -lpthread -o tap-vsockd

CMD ["tar", "cf", "-", "tap-vsockd"]
