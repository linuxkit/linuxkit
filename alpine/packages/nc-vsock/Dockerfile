FROM mobylinux/alpine-build-c:1b53976210ba819823eda5b1d0d3ea337772e97b

RUN mkdir -p /nc-vsock
WORKDIR /nc-vsock

COPY . /nc-vsock

RUN make nc-vsock
