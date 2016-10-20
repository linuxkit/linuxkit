FROM mobylinux/alpine-build-c:7303e33e9dcd5276b8bb5269644a9bf3354008c8

COPY . /nc-vsock

WORKDIR /nc-vsock

RUN cc -Wall -Werror -o nc-vsock nc-vsock.c -luuid

CMD ["tar", "cf", "-", "nc-vsock"]
