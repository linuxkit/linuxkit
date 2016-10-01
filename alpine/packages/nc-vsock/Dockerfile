FROM mobylinux/alpine-build-c:f97e13b3a7339af5da5d620fc053c6a6c81b00a6

COPY . /nc-vsock

WORKDIR /nc-vsock

RUN cc -Wall -Werror -o nc-vsock nc-vsock.c -luuid

CMD ["tar", "cf", "-", "nc-vsock"]
