FROM mobylinux/alpine-build-c

RUN mkdir -p /nc-vsock
WORKDIR /nc-vsock

COPY . /nc-vsock

RUN make nc-vsock
