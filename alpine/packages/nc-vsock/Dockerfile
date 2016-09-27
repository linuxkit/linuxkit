FROM mobylinux/alpine-build-c:41e7634e137b022e6fef9d651166783966de6725

COPY . /nc-vsock

WORKDIR /nc-vsock

RUN cc -Wall -Werror -o nc-vsock nc-vsock.c -luuid

CMD ["tar", "cf", "-", "nc-vsock"]
