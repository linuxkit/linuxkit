FROM mobylinux/alpine-build-c:36aecb5cf4738737634140eec9abebe1f6559a39

COPY . /tap-vsockd

WORKDIR /tap-vsockd

RUN cc -Wall -Werror tap-vsockd.c hvsock.c protocol.c -lpthread -o tap-vsockd

CMD ["tar", "cf", "-", "tap-vsockd"]
