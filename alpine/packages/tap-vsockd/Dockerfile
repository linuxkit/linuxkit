FROM mobylinux/alpine-build-c:dfe78f283402fc2d6940c43763e2921491fb09fa

COPY . /tap-vsockd

WORKDIR /tap-vsockd

RUN cc -Wall -Werror tap-vsockd.c hvsock.c protocol.c -lpthread -o tap-vsockd

CMD ["tar", "cf", "-", "tap-vsockd"]
