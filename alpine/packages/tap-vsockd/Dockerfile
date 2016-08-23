FROM mobylinux/alpine-build-c

RUN mkdir -p /tap-vsockd
WORKDIR /tap-vsockd

COPY . /tap-vsockd

RUN make tap-vsockd
