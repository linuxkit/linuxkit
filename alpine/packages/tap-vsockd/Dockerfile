# Tag: b77cfc4ad0033d4366df830ed697afc7bab458a2
FROM mobylinux/alpine-build-c@sha256:53739ea6042cb0ac39cf6e262012c1c4224206b2c9b719569fe7efa3a381348c

COPY . /tap-vsockd

WORKDIR /tap-vsockd

RUN cc -Wall -Werror tap-vsockd.c hvsock.c protocol.c -lpthread -o tap-vsockd

CMD ["tar", "cf", "-", "tap-vsockd"]
