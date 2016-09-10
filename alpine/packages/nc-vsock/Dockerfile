FROM mobylinux/alpine-build-c:1b53976210ba819823eda5b1d0d3ea337772e97b

COPY . /nc-vsock

WORKDIR /nc-vsock

RUN cc -Wall -Werror -o nc-vsock nc-vsock.c -luuid

CMD ["tar", "cf", "-", "nc-vsock"]
