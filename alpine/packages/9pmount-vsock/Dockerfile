FROM mobylinux/alpine-build-c:1b53976210ba819823eda5b1d0d3ea337772e97b

RUN mkdir -p /9pmount-vsock
WORKDIR /9pmount-vsock

COPY . /9pmount-vsock

RUN make 9pmount-vsock
