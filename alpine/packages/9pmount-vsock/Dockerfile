FROM mobylinux/alpine-build-c

RUN mkdir -p /9pmount-vsock
WORKDIR /9pmount-vsock

COPY . /9pmount-vsock

RUN make 9pmount-vsock
