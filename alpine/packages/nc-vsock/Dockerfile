FROM mobylinux/alpine-build-c:1ea690a7438cdd8a5965eaa034e0a0c521d9cb40

COPY . /nc-vsock

WORKDIR /nc-vsock

RUN cc -Wall -Werror -o nc-vsock nc-vsock.c -luuid

CMD ["tar", "cf", "-", "nc-vsock"]
